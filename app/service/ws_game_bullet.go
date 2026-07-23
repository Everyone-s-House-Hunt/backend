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
	bulletGameMode     = "bullet"
	bulletTimeLimitSec = 60 // 全体で共有する制限時間
	bulletTargetHits   = 10 // 撃破に必要な命中（正解）数
	bulletMaxPlayers   = 5  // 5レーンに配置できる最大参加人数
	bulletFetchPool    = 30 // 正解数フィルタ用に多めに取得する件数
)

// answer_data(JSON文字列)のパース先。例 {"question":"赤い果物を10個答えよ","answers":["りんご",...]}
type bulletAnswerData struct {
	Question string   `json:"question"`
	Answers  []string `json:"answers"`
}

// ゾンビバレット（複数解答リレー）。GameLogic を実装。
// 仕様: お題の正解を順番に1つずつ答える。正解で次の人へ・不正解/重複は同じ人が続行。
// 全体共有の60秒以内に10命中で撃破クリア、時間切れでゲームオーバー。
type Bullet struct {
	questionRepo QuestionRepo
	mu           sync.Mutex
	players      []model.PlayerInfo // 参加順スナップショット（往復順の基準）
	turnIndex    int                // 現ターン = players[turnIndex]
	answerNorm   map[string]string  // 正規化した正解 → 表示用の元文字列
	used         map[string]bool    // 既に正解された正規化キー
	correctCount int                // 命中数（= len(used)）
	doneCh       chan struct{}      // 撃破（10命中）で閉じる
	finished     bool               // 終了後の submit を弾く
}

func NewBullet(qr QuestionRepo) *Bullet {
	return &Bullet{questionRepo: qr}
}

// 正解数から次の回答者を決める。両端を2回ずつ通る往復順を繰り返す。
// 例: 5人なら 0,1,2,3,4,4,3,2,1,0、3人なら 0,1,2,2,1,0,...
func bulletTurnIndex(correctCount, playerCount int) int {
	if playerCount <= 1 {
		return 0
	}
	step := correctCount % (playerCount * 2)
	if step < 0 {
		step += playerCount * 2
	}
	if step < playerCount {
		return step
	}
	return playerCount*2 - 1 - step
}

// ゲーム全体を進行させる。正解10個以上のお題を1問使い、全体タイマーで撃破か時間切れを待つ。
func (g *Bullet) Start(hub *Hub) error {
	players := hub.OrderedPlayers() // 参加順（JoinSeq 昇順）= 往復順の基準
	if len(players) == 0 {
		return errors.New("no players")
	}
	if len(players) > bulletMaxPlayers {
		return fmt.Errorf("zombie bullet supports at most %d players", bulletMaxPlayers)
	}

	// プールから取得し、正解が目標数以上ある問題を1つ採用する
	pool, err := g.questionRepo.GetRandomByGameMode(bulletGameMode, bulletFetchPool)
	if err != nil {
		return fmt.Errorf("failed to fetch questions: %w", err)
	}

	var chosen *bulletAnswerData
	for _, q := range pool {
		var ad bulletAnswerData
		if err := json.Unmarshal([]byte(q.AnswerData), &ad); err != nil {
			continue // 壊れた問題はスキップ
		}
		if len(ad.Answers) >= bulletTargetHits {
			chosen = &ad
			break
		}
	}
	if chosen == nil {
		return fmt.Errorf("no question with at least %d answers", bulletTargetHits)
	}

	// 正解を正規化して登録（重複検出・照合用）
	answerNorm := make(map[string]string, len(chosen.Answers))
	for _, a := range chosen.Answers {
		answerNorm[normalizeAnswer(a)] = a
	}

	roundCh := make(chan struct{})
	g.mu.Lock()
	g.players = players
	g.turnIndex = 0
	g.answerNorm = answerNorm
	g.used = make(map[string]bool)
	g.correctCount = 0
	g.doneCh = roundCh
	g.finished = false
	g.mu.Unlock()

	// 往復順の基準となる参加順を配信
	bps := make([]model.BulletPlayer, 0, len(players))
	for i, p := range players {
		bps = append(bps, model.BulletPlayer{Position: i, PlayerID: p.PlayerID, Nickname: p.Nickname})
	}

	hub.Broadcast(&model.OutgoingMessage{
		Type: model.MsgGameBulletStart,
		Payload: model.BulletStartPayload{
			Question:        chosen.Question,
			TargetHits:      bulletTargetHits,
			TimeLimitSec:    bulletTimeLimitSec,
			Players:         bps,
			CurrentPlayerID: players[0].PlayerID,
		},
	})

	// 撃破 or 時間切れ のどちらか早いほうまで待つ（進行は HandleMessage 側）
	ctx, cancel := context.WithTimeout(hub.Context(), bulletTimeLimitSec*time.Second)
	defer cancel()

	select {
	case <-roundCh: // 10命中 → 撃破クリア
		hub.Broadcast(&model.OutgoingMessage{
			Type:    model.MsgGameClear,
			Payload: model.GameClearPayload{TotalRounds: bulletTargetHits},
		})
	case <-ctx.Done():
		if hub.Context().Err() != nil { // ルーム破棄なら中止
			return nil
		}
		// 時間切れ → ゲームオーバー。以後の submit を弾く
		g.mu.Lock()
		g.finished = true
		g.mu.Unlock()
		hub.Broadcast(&model.OutgoingMessage{
			Type: model.MsgGameOver,
			Payload: model.GameOverPayload{
				Reason:     "timeout",
				FinalRound: g.correctCount,
			},
		})
	}

	hub.SetState(StateFinished)
	return nil
}

// ゲーム中メッセージのうち game:bullet_submit を処理する。GameLogic を実装。
// ここにゲーム進行（正誤判定・ターン移行・撃破判定）を集約する。
func (g *Bullet) HandleMessage(hub *Hub, playerID, msgType string, payload json.RawMessage) error {
	if msgType != model.MsgGameBulletSubmit {
		return fmt.Errorf("unsupported message type: %s", msgType)
	}
	var p model.BulletSubmitPayload
	if err := json.Unmarshal(payload, &p); err != nil {
		return errors.New("invalid game:bullet_submit payload")
	}
	norm := normalizeAnswer(p.Answer)

	g.mu.Lock()
	if g.finished {
		g.mu.Unlock()
		return errors.New("game already finished")
	}
	n := len(g.players)
	if g.players[g.turnIndex].PlayerID != playerID { // 自分の番でないと回答不可
		g.mu.Unlock()
		return errors.New("not your turn")
	}

	display, isAnswer := g.answerNorm[norm]

	// 不正解 or 重複 → ターン据え置きでミス通知
	if norm == "" || !isAnswer || g.used[norm] {
		reason := "wrong"
		if isAnswer && g.used[norm] {
			reason = "duplicate"
		}
		current := g.players[g.turnIndex].PlayerID
		g.mu.Unlock()

		hub.Broadcast(&model.OutgoingMessage{
			Type: model.MsgGameBulletMiss,
			Payload: model.BulletMissPayload{
				PlayerID:        playerID,
				Answer:          p.Answer,
				Reason:          reason,
				CurrentPlayerID: current,
			},
		})
		return nil
	}

	// 命中（正解）→ 記録し、クリア前なら往復順で次の回答者へ
	g.used[norm] = true
	g.correctCount++
	correctCount := g.correctCount
	cleared := correctCount >= bulletTargetHits
	if !cleared {
		g.turnIndex = bulletTurnIndex(correctCount, n)
	}
	nextPlayer := g.players[g.turnIndex].PlayerID

	used := make([]string, 0, len(g.used))
	for k := range g.answerNorm {
		if g.used[k] {
			used = append(used, g.answerNorm[k]) // 表示用の元文字列で返す
		}
	}
	if cleared {
		g.finished = true
	}
	ch := g.doneCh
	g.mu.Unlock()

	hub.Broadcast(&model.OutgoingMessage{
		Type: model.MsgGameBulletHit,
		Payload: model.BulletHitPayload{
			PlayerID:        playerID,
			Answer:          display,
			CorrectCount:    correctCount,
			TargetHits:      bulletTargetHits,
			CurrentPlayerID: nextPlayer,
			Used:            used,
		},
	})

	// 撃破ならチャネルを閉じて Start の待機を解除（二重 close 防止に select+default）
	if cleared {
		select {
		case <-ch:
		default:
			close(ch)
		}
	}

	return nil
}
