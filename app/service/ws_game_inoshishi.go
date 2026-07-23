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
	inoshishiGameMode       = "panic"
	inoshishiQuestions      = 10 // 出題数
	inoshishiTimeLimitSec   = 10 // 制限時間
	inoshishiResultPauseSec = 6  // 結果表示〜次ラウンドの間（フロント演出: 得票1.5s+扉3.5s に合わせる）
)

// answer_data(JSON文字列)のパース先。例 {"choices":["東京","大阪"],"correct_index":0}
type inoshishiAnswerData struct {
	Choices      []string `json:"choices"`
	CorrectIndex int      `json:"correct_index"`
}

// イノシシパニック（二択投票）。GameLogic を実装。
type InoshishiPanic struct {
	questionRepo QuestionRepo
	mu           sync.Mutex     // votes / allVotedCh を保護
	votes        map[string]int // プレイヤーID → 選択肢。ラウンドごとに作り直す
	allVotedCh   chan struct{}  // 全員投票完了を Start へ知らせる。ラウンドごとに作り直す
}

func NewInoshishiPanic(qr QuestionRepo) *InoshishiPanic {
	return &InoshishiPanic{questionRepo: qr}
}

// ゲーム全体を進行させる。1問ずつラウンドを回し、ゲームオーバーか全クリアで終わる。
func (g *InoshishiPanic) Start(hub *Hub, gameCtx context.Context, runID uint64) error {
	questions, err := g.questionRepo.GetRandomByGameMode(inoshishiGameMode, inoshishiQuestions)
	if err != nil {
		return fmt.Errorf("failed to fetch questions: %w", err)
	}
	if len(questions) < inoshishiQuestions { // 問題が足りなければ開始しない
		return fmt.Errorf("not enough questions: need %d, got %d", inoshishiQuestions, len(questions))
	}

	totalRounds := len(questions)

	// === ラウンドループ ===
	for i, q := range questions {
		roundNum := i + 1

		var answerData inoshishiAnswerData
		if err := json.Unmarshal([]byte(q.AnswerData), &answerData); err != nil {
			return fmt.Errorf("failed to parse question %d answer_data: %w", roundNum, err)
		}

		// このラウンド用に投票箱と完了通知チャネルを用意
		roundCh := make(chan struct{})
		g.mu.Lock()
		g.votes = make(map[string]int)
		g.allVotedCh = roundCh
		g.mu.Unlock()

		// 問題・選択肢・制限時間を配信
		hub.BroadcastGame(runID, &model.OutgoingMessage{
			Type: model.MsgGameRoundStart,
			Payload: model.GameRoundStartPayload{
				Round:        roundNum,
				TotalRounds:  totalRounds,
				Question:     q.Body,
				Choices:      answerData.Choices,
				TimeLimitSec: inoshishiTimeLimitSec,
			},
		})

		// 全員投票 or 時間切れ のどちらか早いほうまで待つ
		roundCtx, cancel := context.WithTimeout(gameCtx, inoshishiTimeLimitSec*time.Second)
		timedOut := false
		select {
		case <-roundCh: // 全員投票完了 → 即判定
			cancel()
		case <-roundCtx.Done(): // 時間切れ（またはルーム破棄）
			cancel()
			timedOut = true
		}

		if gameCtx.Err() != nil { // ルーム破棄・ゲーム中断なら中止
			return nil
		}

		// === 集計：選択肢ごとに投票者を振り分け、未投票は noVote へ ===
		playerIDs := hub.GetPlayerIDs()
		g.mu.Lock()
		votes := g.votes
		g.mu.Unlock()

		voteCounts := make(map[int][]string)
		noVote := []string{}
		for _, pid := range playerIDs {
			if choice, ok := votes[pid]; ok {
				voteCounts[choice] = append(voteCounts[choice], pid)
			} else {
				noVote = append(noVote, pid)
			}
		}

		votesStr := make(map[string][]string) // クライアント返却用にキーを文字列化
		for k, v := range voteCounts {
			votesStr[fmt.Sprintf("%d", k)] = v
		}

		count0 := len(voteCounts[0])
		count1 := len(voteCounts[1])

		// === 判定 ===
		result := "correct"
		gameEnd := false
		gameOverReason := ""

		if count0 == count1 {
			// 同票（0対0含む）→ 多数派なしで即ゲームオーバー
			result = "tie"
			gameEnd = true
			if timedOut {
				gameOverReason = "timeout_tie"
			} else {
				gameOverReason = "tie"
			}
		} else {
			majority := 0
			if count1 > count0 {
				majority = 1
			}
			if majority != answerData.CorrectIndex { // 多数派が不正解
				result = "wrong"
				gameEnd = true
				gameOverReason = "wrong_answer"
			}
		}

		// ラウンド結果を配信
		hub.BroadcastGame(runID, &model.OutgoingMessage{
			Type: model.MsgGameRoundResult,
			Payload: model.GameRoundResultPayload{
				Round:        roundNum,
				CorrectIndex: answerData.CorrectIndex,
				Votes:        votesStr,
				NoVote:       noVote,
				Result:       result,
			},
		})

		if gameEnd {
			hub.BroadcastGame(runID, &model.OutgoingMessage{
				Type: model.MsgGameOver,
				Payload: model.GameOverPayload{
					Reason:     gameOverReason,
					FinalRound: roundNum,
				},
			})
			return nil
		}

		// 次ラウンドまで少し待つ（破棄されたら中止）
		select {
		case <-gameCtx.Done():
			return nil
		case <-time.After(inoshishiResultPauseSec * time.Second):
		}
	}

	// 全問正解 → クリア
	hub.BroadcastGame(runID, &model.OutgoingMessage{
		Type:    model.MsgGameClear,
		Payload: model.GameClearPayload{TotalRounds: totalRounds},
	})
	return nil
}

// ゲーム中メッセージのうち game:vote を処理する。GameLogic を実装。
func (g *InoshishiPanic) HandleMessage(hub *Hub, runID uint64, playerID, msgType string, payload json.RawMessage) error {
	if msgType != model.MsgGameVote {
		return fmt.Errorf("unsupported message type: %s", msgType)
	}
	var p model.GameVotePayload
	if err := json.Unmarshal(payload, &p); err != nil {
		return errors.New("invalid game:vote payload")
	}
	return g.handleVote(hub, runID, playerID, p.ChoiceIndex)
}

// 退出者の票を集計対象から外し、残った全員が投票済みなら即集計へ進める。
func (g *InoshishiPanic) HandlePlayerLeave(
	playerID string,
	players []model.PlayerInfo,
) model.GamePlayerLeftPayload {
	g.mu.Lock()
	delete(g.votes, playerID)
	votedCount := len(g.votes)
	totalCount := len(players)
	ch := g.allVotedCh
	if ch != nil && votedCount >= totalCount {
		select {
		case <-ch:
		default:
			close(ch)
		}
	}
	g.mu.Unlock()

	return model.GamePlayerLeftPayload{
		DisconnectedPlayerID: playerID,
		Players:              players,
		VotedCount:           votedCount,
		TotalCount:           totalCount,
	}
}

// 1人分の投票を受け付ける。複数人が同時に投票しても壊れないようロックする。
func (g *InoshishiPanic) handleVote(hub *Hub, runID uint64, playerID string, choiceIndex int) error {
	if choiceIndex != 0 && choiceIndex != 1 {
		return errors.New("invalid choice_index: must be 0 or 1")
	}

	playerCount := hub.PlayerCount()

	g.mu.Lock()
	if _, already := g.votes[playerID]; already { // 二重投票不可
		g.mu.Unlock()
		return errors.New("already voted")
	}
	g.votes[playerID] = choiceIndex
	voteCount := len(g.votes)
	ch := g.allVotedCh
	g.mu.Unlock()

	// 投票進捗を全員に通知（内容は伏せる）
	hub.BroadcastGame(runID, &model.OutgoingMessage{
		Type: model.MsgGameVoteReceived,
		Payload: model.GameVoteReceivedPayload{
			PlayerID:   playerID,
			VotedCount: voteCount,
			TotalCount: playerCount,
		},
	})

	// 全員揃ったらチャネルを閉じて Start の待機を解除（二重 close 防止に select+default）
	if voteCount >= playerCount {
		select {
		case <-ch:
		default:
			close(ch)
		}
	}

	return nil
}
