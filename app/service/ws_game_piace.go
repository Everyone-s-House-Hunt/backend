package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"house-hunt/model"
)

const (
	piaceGameMode       = "piace"
	piaceRounds         = 5  // 出題数
	piaceTimeLimitSec   = 60 // 制限時間
	piaceResultPauseSec = 3  // 結果表示〜次ラウンドの間
	piaceFetchPool      = 50 // 文字数フィルタ用に多めに取得する件数
)

// answer_data(JSON文字列)のパース先。例 {"question":"日本の首都は？","answer":"とうきょう"}
type piaceAnswerData struct {
	Question string `json:"question"`
	Answer   string `json:"answer"` // 正解。この文字数 = マス数 = 人数
}

// コトバピース（みんなで1つの言葉を1人1文字ずつ埋める協力ゲーム）。GameLogic を実装。
// 仕様: マス数 = 参加人数。各プレイヤーは参加順に1マスを担当し、全マス正解でクリア。
type Piace struct {
	questionRepo QuestionRepo
	mu           sync.Mutex        // 下記をまとめて保護。ラウンドごとに作り直す
	answer       []rune            // このラウンドの正解（位置ごとの正解文字）
	posOf        map[string]int    // プレイヤーID → 担当position（1:1）
	filled       map[string]string // プレイヤーID → 入力された文字
	allFilledCh  chan struct{}     // 全員入力完了を Start へ知らせる
}

func NewPiace(qr QuestionRepo) *Piace {
	return &Piace{questionRepo: qr}
}

// ゲーム全体を進行させる。文字数が人数に一致する問題だけを使い、1問ずつラウンドを回す。
func (g *Piace) Start(hub *Hub) error {
	players := hub.OrderedPlayers() // 参加順（JoinSeq 昇順）
	n := len(players)
	if n == 0 {
		return errors.New("no players")
	}

	// 多めに取得してから「答えの文字数 == 人数」のものだけ採用する
	pool, err := g.questionRepo.GetRandomByGameMode(piaceGameMode, piaceFetchPool)
	if err != nil {
		return fmt.Errorf("failed to fetch questions: %w", err)
	}

	questions := make([]model.Question, 0, piaceRounds)
	answers := make([][]rune, 0, piaceRounds)
	for _, q := range pool {
		var ad piaceAnswerData
		if err := json.Unmarshal([]byte(q.AnswerData), &ad); err != nil {
			continue // 壊れた問題はスキップ
		}
		runes := []rune(ad.Answer)
		if len(runes) != n { // 文字数が人数と一致する問題だけ使う
			continue
		}
		questions = append(questions, q)
		answers = append(answers, runes)
		if len(questions) == piaceRounds {
			break
		}
	}
	if len(questions) < piaceRounds {
		return fmt.Errorf("not enough questions for %d players: need %d, got %d", n, piaceRounds, len(questions))
	}

	totalRounds := piaceRounds

	// === ラウンドループ ===
	for i := 0; i < totalRounds; i++ {
		roundNum := i + 1
		q := questions[i]
		answerRunes := answers[i]

		var ad piaceAnswerData
		_ = json.Unmarshal([]byte(q.AnswerData), &ad) // 上でパース済みなので失敗しない

		// 担当割り当て: players[j] → position j（1:1）
		posOf := make(map[string]int, n)
		slots := make([]model.PiaceSlot, 0, n)
		for j, p := range players {
			posOf[p.PlayerID] = j
			slots = append(slots, model.PiaceSlot{
				Position: j,
				PlayerID: p.PlayerID,
				Nickname: p.Nickname,
			})
		}

		// このラウンド用の入力箱と完了通知チャネルを用意
		roundCh := make(chan struct{})
		g.mu.Lock()
		g.answer = answerRunes
		g.posOf = posOf
		g.filled = make(map[string]string)
		g.allFilledCh = roundCh
		g.mu.Unlock()

		// お題・マス数・担当・制限時間を配信
		hub.Broadcast(&model.OutgoingMessage{
			Type: model.MsgGamePiaceRoundStart,
			Payload: model.PiaceRoundStartPayload{
				Round:        roundNum,
				TotalRounds:  totalRounds,
				Question:     ad.Question,
				SlotCount:    n,
				Slots:        slots,
				TimeLimitSec: piaceTimeLimitSec,
			},
		})

		// 全員入力 or 時間切れ のどちらか早いほうまで待つ
		roundCtx, cancel := context.WithTimeout(hub.Context(), piaceTimeLimitSec*time.Second)
		timedOut := false
		select {
		case <-roundCh: // 全員入力完了 → 即判定
			cancel()
		case <-roundCtx.Done(): // 時間切れ（またはルーム破棄）
			cancel()
			timedOut = true
		}

		if hub.Context().Err() != nil { // ルーム破棄なら中止
			return nil
		}

		// === 集計：各プレイヤーの入力を担当位置に当てはめて判定 ===
		g.mu.Lock()
		filled := g.filled
		g.mu.Unlock()

		assembled := make([]rune, n) // 表示用の組み立て結果（未入力は '＿'）
		slotResults := make([]model.PiaceSlotResult, 0, n)
		allCorrect := true
		for j, p := range players {
			input := filled[p.PlayerID]
			correct := string(answerRunes[j])
			ok := input == correct
			if !ok {
				allCorrect = false
			}
			if input == "" {
				assembled[j] = '＿'
			} else {
				assembled[j] = []rune(input)[0]
			}
			slotResults = append(slotResults, model.PiaceSlotResult{
				Position:    j,
				PlayerID:    p.PlayerID,
				Nickname:    p.Nickname,
				Char:        input,
				CorrectChar: correct,
				OK:          ok,
			})
		}

		// ラウンド結果を配信
		hub.Broadcast(&model.OutgoingMessage{
			Type: model.MsgGamePiaceRoundResult,
			Payload: model.PiaceRoundResultPayload{
				Round:         roundNum,
				Assembled:     string(assembled),
				CorrectAnswer: string(answerRunes),
				IsCorrect:     allCorrect,
				Slots:         slotResults,
			},
		})

		if !allCorrect {
			reason := "wrong_answer"
			if timedOut {
				reason = "timeout"
			}
			hub.Broadcast(&model.OutgoingMessage{
				Type: model.MsgGameOver,
				Payload: model.GameOverPayload{
					Reason:     reason,
					FinalRound: roundNum,
				},
			})
			hub.SetState(StateFinished)
			return nil
		}

		// 次ラウンドまで少し待つ（破棄されたら中止）
		select {
		case <-hub.Context().Done():
			return nil
		case <-time.After(piaceResultPauseSec * time.Second):
		}
	}

	// 全問正解 → クリア
	hub.Broadcast(&model.OutgoingMessage{
		Type:    model.MsgGameClear,
		Payload: model.GameClearPayload{TotalRounds: totalRounds},
	})
	hub.SetState(StateFinished)
	return nil
}

// ゲーム中メッセージのうち game:piace_submit を処理する。GameLogic を実装。
func (g *Piace) HandleMessage(hub *Hub, playerID, msgType string, payload json.RawMessage) error {
	if msgType != model.MsgGamePiaceSubmit {
		return fmt.Errorf("unsupported message type: %s", msgType)
	}
	var p model.PiaceSubmitPayload
	if err := json.Unmarshal(payload, &p); err != nil {
		return errors.New("invalid game:piace_submit payload")
	}

	char := normalizeAnswer(p.Char)
	if len([]rune(char)) != 1 { // 1文字だけ受け付ける
		return errors.New("char must be exactly one character")
	}

	g.mu.Lock()
	if _, ok := g.posOf[playerID]; !ok { // 参加者でなければ拒否
		g.mu.Unlock()
		return errors.New("you are not assigned a slot")
	}
	g.filled[playerID] = char
	filledCount := len(g.filled)
	slotCount := len(g.posOf)
	ch := g.allFilledCh
	g.mu.Unlock()

	// 入力進捗を全員に通知（内容は伏せる）
	hub.Broadcast(&model.OutgoingMessage{
		Type: model.MsgGamePiaceProgress,
		Payload: model.PiaceProgressPayload{
			FilledCount: filledCount,
			SlotCount:   slotCount,
		},
	})

	// 全員揃ったらチャネルを閉じて Start の待機を解除（二重 close 防止に select+default）
	if filledCount >= slotCount {
		select {
		case <-ch:
		default:
			close(ch)
		}
	}

	return nil
}
