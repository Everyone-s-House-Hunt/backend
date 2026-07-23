package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"
	"unicode"

	"house-hunt/model"
)

const (
	orderGameMode       = "order"
	orderTimeLimitSec   = 10 // 1ターンの制限時間
	orderResultPauseSec = 2  // 結果表示〜次ターンの間
)

// answer_data(JSON文字列)のパース先。
// 例 {"question":"紅葉","answers":["もみじ","こうよう"],"direction":"kanji_to_reading"}
type orderAnswerData struct {
	Question  string   `json:"question"`  // 表示する漢字 or 読み（空なら Question.Body を使う）
	Answers   []string `json:"answers"`   // 許容する正解（表記ゆれ・複数読みを許す）
	Direction string   `json:"direction"` // kanji_to_reading / reading_to_kanji
}

// モジオーダー（順番に1人1問ずつ回答するターン制）。GameLogic を実装。
type MojiOrder struct {
	questionRepo QuestionRepo
	mu           sync.Mutex  // currentID / answered / answerCh を保護
	currentID    string      // このターンの回答者ID。ターンごとに更新
	answered     bool        // 二重回答防止。ターンごとに false へ戻す
	answerCh     chan string // 現ターンの回答を Start へ渡す。ターンごとに作り直す
}

func NewMojiOrder(qr QuestionRepo) *MojiOrder {
	return &MojiOrder{questionRepo: qr}
}

// ゲーム全体を進行させる。参加順に1人1問ずつ出題し、誰か誤答で即終了、全員正答でクリア。
func (g *MojiOrder) Start(hub *Hub, gameCtx context.Context, runID uint64) error {
	players := hub.OrderedPlayers() // 参加順（JoinSeq 昇順）
	total := len(players)
	if total == 0 {
		return errors.New("no players")
	}

	// 総出題数 = 参加人数（1人1問）
	questions, err := g.questionRepo.GetRandomByGameMode(orderGameMode, total)
	if err != nil {
		return fmt.Errorf("failed to fetch questions: %w", err)
	}
	if len(questions) < total {
		return fmt.Errorf("not enough questions: need %d, got %d", total, len(questions))
	}

	totalRounds := total

	// === ターンループ：参加順に1人ずつ ===
	for i, p := range players {
		roundNum := i + 1
		q := questions[i]

		var answerData orderAnswerData
		if err := json.Unmarshal([]byte(q.AnswerData), &answerData); err != nil {
			return fmt.Errorf("failed to parse question %d answer_data: %w", roundNum, err)
		}

		// このターン用の回答箱を用意（バッファ1で HandleMessage 側をブロックさせない）
		turnCh := make(chan string, 1)
		g.mu.Lock()
		g.currentID = p.PlayerID
		g.answered = false
		g.answerCh = turnCh
		g.mu.Unlock()

		// 表示する問題文（answer_data 優先、無ければ Body）
		questionText := answerData.Question
		if questionText == "" {
			questionText = q.Body
		}

		// 誰の番か・問題・制限時間を全員へ配信
		hub.BroadcastGame(runID, &model.OutgoingMessage{
			Type: model.MsgGameTurnStart,
			Payload: model.GameTurnStartPayload{
				Round:           roundNum,
				TotalRounds:     totalRounds,
				CurrentPlayerID: p.PlayerID,
				Nickname:        p.Nickname,
				Question:        questionText,
				Direction:       answerData.Direction,
				TimeLimitSec:    orderTimeLimitSec,
			},
		})

		// 回答 or 時間切れ のどちらか早いほうまで待つ
		turnCtx, cancel := context.WithTimeout(gameCtx, orderTimeLimitSec*time.Second)
		var answer string
		timedOut := false
		select {
		case answer = <-turnCh: // 回答受信 → 即判定
			cancel()
		case <-turnCtx.Done(): // 時間切れ（またはルーム破棄）
			cancel()
			timedOut = true
		}

		if gameCtx.Err() != nil { // ルーム破棄・ゲーム中断なら中止
			return nil
		}

		// === 判定：時間切れは不正解扱い ===
		isCorrect := !timedOut && matchAnswer(answer, answerData.Answers)

		// 回答結果を配信
		hub.BroadcastGame(runID, &model.OutgoingMessage{
			Type: model.MsgGameAnswerResult,
			Payload: model.GameAnswerResultPayload{
				Round:         roundNum,
				PlayerID:      p.PlayerID,
				Answer:        answer,
				CorrectAnswer: strings.Join(answerData.Answers, " / "),
				IsCorrect:     isCorrect,
			},
		})

		if !isCorrect {
			reason := "wrong_answer"
			if timedOut {
				reason = "timeout"
			}
			hub.BroadcastGame(runID, &model.OutgoingMessage{
				Type: model.MsgGameOver,
				Payload: model.GameOverPayload{
					Reason:     reason,
					FinalRound: roundNum,
					PlayerID:   p.PlayerID,
				},
			})
			return nil
		}

		// 次ターンまで少し待つ（破棄されたら中止）
		select {
		case <-gameCtx.Done():
			return nil
		case <-time.After(orderResultPauseSec * time.Second):
		}
	}

	// 全員正答 → クリア
	hub.BroadcastGame(runID, &model.OutgoingMessage{
		Type:    model.MsgGameClear,
		Payload: model.GameClearPayload{TotalRounds: totalRounds},
	})
	return nil
}

// ゲーム中メッセージのうち game:answer を処理する。GameLogic を実装。
func (g *MojiOrder) HandleMessage(hub *Hub, runID uint64, playerID, msgType string, payload json.RawMessage) error {
	if msgType != model.MsgGameAnswer {
		return fmt.Errorf("unsupported message type: %s", msgType)
	}
	var p model.GameAnswerPayload
	if err := json.Unmarshal(payload, &p); err != nil {
		return errors.New("invalid game:answer payload")
	}

	g.mu.Lock()
	if playerID != g.currentID { // 自分の番でないと回答不可
		g.mu.Unlock()
		return errors.New("not your turn")
	}
	if g.answered { // このターンで既に回答済み
		g.mu.Unlock()
		return errors.New("already answered")
	}
	g.answered = true
	ch := g.answerCh
	g.mu.Unlock()

	// Start の待機を解除（バッファ1なのでブロックしない）
	ch <- p.Answer
	return nil
}

// 入力を正規化したうえで、許容解のいずれかと一致すれば正解。
func matchAnswer(input string, answers []string) bool {
	norm := normalizeAnswer(input)
	if norm == "" {
		return false
	}
	for _, a := range answers {
		if norm == normalizeAnswer(a) {
			return true
		}
	}
	return false
}

// 回答文字列を正規化する：前後空白除去 → 空白除去 → カタカナをひらがなへ寄せる。
func normalizeAnswer(s string) string {
	s = strings.TrimSpace(s)
	var b strings.Builder
	for _, r := range s {
		if unicode.IsSpace(r) {
			continue
		}
		// カタカナ(ァ〜ン)をひらがなへ寄せて表記ゆれを吸収
		if r >= 'ァ' && r <= 'ン' {
			r -= 0x60
		}
		b.WriteRune(r)
	}
	return b.String()
}
