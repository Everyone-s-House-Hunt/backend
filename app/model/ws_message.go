package model

import "encoding/json"

// メッセージ種別。JSON の {"type": "..."} に入る値。
const (
	// クライアント → サーバー
	MsgRoomJoin   = "room:join"   // 入室
	MsgGameStart  = "game:start"  // ゲーム開始（ホストのみ）
	MsgGameVote   = "game:vote"   // 投票（イノシシパニック）
	MsgGameAnswer = "game:answer" // 回答（モジオーダー：現ターンのプレイヤーのみ）

	// サーバー → クライアント
	MsgRoomJoined       = "room:joined"         // 入室確認（本人へ）
	MsgRoomPlayerJoined = "room:player_joined"  // 入室通知（全員へ）
	MsgRoomDestroyed    = "room:destroyed"      // ルーム破棄
	MsgGameRoundStart   = "game:round_start"    // ラウンド開始（イノシシ）
	MsgGameVoteReceived = "game:vote_received"  // 投票進捗（イノシシ）
	MsgGameRoundResult  = "game:round_result"   // ラウンド結果（イノシシ）
	MsgGameTurnStart    = "game:turn_start"     // ターン開始（モジオーダー）
	MsgGameAnswerResult = "game:answer_result"  // 回答結果（モジオーダー）
	MsgGameOver         = "game:over"           // ゲームオーバー
	MsgGameClear        = "game:clear"          // 全クリア
	MsgError            = "error"               // エラー
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

// ゲーム開始時にホストが選ぶゲームモード。例 "panic" / "order"
type GameStartPayload struct {
	GameMode string `json:"game_mode"`
}

// 0 = A, 1 = B
type GameVotePayload struct {
	ChoiceIndex int `json:"choice_index"`
}

// モジオーダーの回答（入力した文字列）
type GameAnswerPayload struct {
	Answer string `json:"answer"`
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

// モジオーダーのターン開始。誰の番か・問題・制限時間を配信する。
type GameTurnStartPayload struct {
	Round           int    `json:"round"`             // 全体の何問目か（1始まり）
	TotalRounds     int    `json:"total_rounds"`      // = 参加人数
	CurrentPlayerID string `json:"current_player_id"` // このターンの回答者
	Nickname        string `json:"nickname"`
	Question        string `json:"question"`          // 表示する漢字 or 読み
	Direction       string `json:"direction"`         // kanji_to_reading / reading_to_kanji
	TimeLimitSec    int    `json:"time_limit_sec"`
}

// モジオーダーの回答結果。
type GameAnswerResultPayload struct {
	Round         int    `json:"round"`
	PlayerID      string `json:"player_id"`
	Answer        string `json:"answer"`         // 実際に入力した文字
	CorrectAnswer string `json:"correct_answer"` // 許容解を " / " で連結
	IsCorrect     bool   `json:"is_correct"`
}

type GameOverPayload struct {
	Reason     string `json:"reason"`              // wrong_answer / tie / timeout_tie / timeout
	FinalRound int    `json:"final_round"`
	PlayerID   string `json:"player_id,omitempty"` // モジオーダーで脱落したプレイヤー
}

type GameClearPayload struct {
	TotalRounds int `json:"total_rounds"`
}

type ErrorPayload struct {
	Message string `json:"message"`
}
