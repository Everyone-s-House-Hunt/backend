package model

import "encoding/json"

// メッセージ種別。JSON の {"type": "..."} に入る値。
const (
	// クライアント → サーバー
	MsgRoomJoin  = "room:join"  // 入室
	MsgGameStart = "game:start" // ゲーム開始（ホストのみ）
	MsgGameVote  = "game:vote"  // 投票

	// サーバー → クライアント
	MsgRoomJoined       = "room:joined"        // 入室確認（本人へ）
	MsgRoomPlayerJoined = "room:player_joined" // 入室通知（全員へ）
	MsgRoomDestroyed    = "room:destroyed"     // ルーム破棄
	MsgGameRoundStart   = "game:round_start"   // ラウンド開始
	MsgGameVoteReceived = "game:vote_received" // 投票進捗
	MsgGameRoundResult  = "game:round_result"  // ラウンド結果
	MsgGameOver         = "game:over"          // ゲームオーバー
	MsgGameClear        = "game:clear"         // 全クリア
	MsgError            = "error"              // エラー
)

// 受信メッセージ。Payload は type を見てから個別の型へパースする。
type IncomingMessage struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

// 送信メッセージ。Payload に下記の各構造体を入れる。
type OutgoingMessage struct {
	Type    string      `json:"type"`
	Payload interface{} `json:"payload"`
}

// プレイヤーの公開情報
type PlayerInfo struct {
	PlayerID string `json:"player_id"`
	Nickname string `json:"nickname"`
	IsHost   bool   `json:"is_host"`
}

// --- 各メッセージの payload ---

type RoomJoinPayload struct {
	Nickname string `json:"nickname"`
}

// 0 = A, 1 = B
type GameVotePayload struct {
	ChoiceIndex int `json:"choice_index"`
}

type RoomJoinedPayload struct {
	PlayerID string       `json:"player_id"`
	IsHost   bool         `json:"is_host"`
	Players  []PlayerInfo `json:"players"`
}

type RoomPlayerJoinedPayload struct {
	PlayerID string       `json:"player_id"`
	Nickname string       `json:"nickname"`
	Players  []PlayerInfo `json:"players"`
}

type RoomDestroyedPayload struct {
	Reason               string `json:"reason"`
	DisconnectedPlayerID string `json:"disconnected_player_id"`
}

type GameRoundStartPayload struct {
	Round        int      `json:"round"`          // ラウンド番号（1始まり）
	TotalRounds  int      `json:"total_rounds"`
	Question     string   `json:"question"`
	Choices      []string `json:"choices"`        // [A, B]
	TimeLimitSec int      `json:"time_limit_sec"`
}

type GameVoteReceivedPayload struct {
	PlayerID   string `json:"player_id"`
	VotedCount int    `json:"voted_count"`
	TotalCount int    `json:"total_count"`
}

// Votes: 選択肢(文字列) → 投票者ID。例 {"0": ["id1"], "1": ["id2"]}
type GameRoundResultPayload struct {
	Round        int                 `json:"round"`
	CorrectIndex int                 `json:"correct_index"`
	Votes        map[string][]string `json:"votes"`
	NoVote       []string            `json:"no_vote"` // 棄権者
	Result       string              `json:"result"`  // correct / wrong / tie
}

type GameOverPayload struct {
	Reason     string `json:"reason"` // wrong_answer / tie / timeout_tie
	FinalRound int    `json:"final_round"`
}

type GameClearPayload struct {
	TotalRounds int `json:"total_rounds"`
}

type ErrorPayload struct {
	Message string `json:"message"`
}
