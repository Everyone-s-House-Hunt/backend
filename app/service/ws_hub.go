package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"sync"

	"github.com/coder/websocket"
	"github.com/google/uuid"
	"house-hunt/model"
)

// ゲームモードを差し替え可能にするインターフェース。他モードもこれを実装すれば追加できる。
type GameLogic interface {
	Start(hub *Hub) error
	// ゲーム中のクライアントメッセージを処理する。type の解釈は各ゲームに委ねる。
	HandleMessage(hub *Hub, playerID, msgType string, payload json.RawMessage) error
}

// game_mode 文字列から対応するゲームを生成する。新モードはここに足す。
func newGame(mode string, qr QuestionRepo) (GameLogic, error) {
	switch mode {
	case inoshishiGameMode:
		return NewInoshishiPanic(qr), nil
	case orderGameMode:
		return NewMojiOrder(qr), nil
	case pieceGameMode:
		return NewPiece(qr), nil
	case bulletGameMode:
		return NewBullet(qr), nil
	default:
		return nil, fmt.Errorf("unknown game_mode: %s", mode)
	}
}

// 問題取得だけを抽象化（service が repository を直接 import しないため）
type QuestionRepo interface {
	GetRandomByGameMode(gameMode string, limit int) ([]model.Question, error)
}

// 接続中の1プレイヤー
type Client struct {
	PlayerID string
	Nickname string
	IsHost   bool
	JoinSeq  int // 参加順（ターン制ゲームの順番に使う）
	Conn     *websocket.Conn
	ctx      context.Context // 切断で cancel される
	cancel   context.CancelFunc
	writeMu  sync.Mutex // 同時書き込み防止
}

func NewClient(conn *websocket.Conn) *Client {
	ctx, cancel := context.WithCancel(context.Background())
	return &Client{
		PlayerID: uuid.New().String(), // IDは自動採番
		Conn:     conn,
		ctx:      ctx,
		cancel:   cancel,
	}
}

func (c *Client) Context() context.Context {
	return c.ctx
}

// 1件送信。coder/websocket は同時書き込み不可なのでロックして書く。
func (c *Client) Send(msg *model.OutgoingMessage) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	c.writeMu.Lock()
	defer c.writeMu.Unlock()
	return c.Conn.Write(c.ctx, websocket.MessageText, data)
}

// 接続を閉じて context をキャンセル（受信ループも抜ける）
func (c *Client) Close(reason string) {
	c.Conn.Close(websocket.StatusNormalClosure, reason)
	c.cancel()
}

// ルームの状態: waiting → playing → finished
const (
	StateWaiting  = "waiting"
	StatePlaying  = "playing"
	StateFinished = "finished"
)

// 1ルームの最大人数（ホスト含む）
const maxRoomPlayers = 5

// 1ルーム内の全クライアントを管理する
type Hub struct {
	RoomID       string
	state        string
	clients      map[string]*Client // プレイヤーID → Client
	hostID       string
	destroyed    bool // 二重破棄防止
	mu           sync.Mutex
	game         GameLogic // 未開始なら nil
	gameMode     string    // 開始時に確定するゲームモード
	joinCounter  int       // JoinSeq 採番用カウンタ
	questionRepo QuestionRepo
	roomManager  *RoomManager
	ctx          context.Context // 破棄で cancel される
	cancel       context.CancelFunc
}

func NewHub(roomID string, rm *RoomManager, qr QuestionRepo) *Hub {
	ctx, cancel := context.WithCancel(context.Background())
	return &Hub{
		RoomID:       roomID,
		state:        StateWaiting,
		clients:      make(map[string]*Client),
		roomManager:  rm,
		questionRepo: qr,
		ctx:          ctx,
		cancel:       cancel,
	}
}

// ルームのライフサイクル context。ゲーム進行の待機がこれを監視し破棄時に中断する。
func (h *Hub) Context() context.Context {
	return h.ctx
}

// クライアントを登録。破棄済み・ゲーム進行中のルームには入れない（拒否理由を error で返す）。
func (h *Hub) Register(client *Client, isHost bool) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.destroyed { // 取得〜登録の間に破棄された
		return errors.New("room not found")
	}
	if h.state != StateWaiting { // 途中参加は不可（作成者は新規ルームなので常に waiting）
		return errors.New("game already in progress")
	}
	if len(h.clients) >= maxRoomPlayers { // 満室（入室処理前の接続も枠として数える）
		return errors.New("room is full")
	}
	client.IsHost = isHost
	client.JoinSeq = h.joinCounter // 参加順を採番
	h.joinCounter++
	if isHost {
		h.hostID = client.PlayerID
	}
	h.clients[client.PlayerID] = client
	return nil
}

// クライアントを除去しルームを破棄する。
// 仕様: 誰か1人でも切断したら、残り全員に room:destroyed を送って接続を閉じる。
func (h *Hub) Unregister(client *Client) {
	h.mu.Lock()
	if h.destroyed { // 既に破棄済みなら何もしない
		h.mu.Unlock()
		return
	}
	if _, exists := h.clients[client.PlayerID]; !exists {
		h.mu.Unlock()
		return
	}
	delete(h.clients, client.PlayerID)
	h.destroyed = true
	remaining := h.copyClients()
	h.mu.Unlock()

	// 残り全員に破棄を通知して切断
	if len(remaining) > 0 {
		msg := &model.OutgoingMessage{
			Type: model.MsgRoomDestroyed,
			Payload: model.RoomDestroyedPayload{
				Reason:               "player_disconnected",
				DisconnectedPlayerID: client.PlayerID,
			},
		}
		for _, c := range remaining {
			c.Send(msg)
			c.Close("room destroyed")
		}
	}

	h.roomManager.DeleteRoom(h.RoomID)
	h.cancel() // 進行中ゲームも中断
}

// 全員に同じメッセージを送る
func (h *Hub) Broadcast(msg *model.OutgoingMessage) {
	h.mu.Lock()
	clients := h.copyClients() // ロックを長く持たないようコピーしてから送る
	h.mu.Unlock()
	for _, c := range clients {
		c.Send(msg)
	}
}

// 特定の1人に送る
func (h *Hub) SendTo(playerID string, msg *model.OutgoingMessage) {
	h.mu.Lock()
	client, ok := h.clients[playerID]
	h.mu.Unlock()
	if ok {
		client.Send(msg)
	}
}

// エラーを1人に送る
func (h *Hub) SendError(playerID, message string) {
	h.SendTo(playerID, &model.OutgoingMessage{
		Type:    model.MsgError,
		Payload: model.ErrorPayload{Message: message},
	})
}

func (h *Hub) PlayerCount() int {
	h.mu.Lock()
	defer h.mu.Unlock()
	return len(h.clients)
}

func (h *Hub) GetPlayerIDs() []string {
	h.mu.Lock()
	defer h.mu.Unlock()
	ids := make([]string, 0, len(h.clients))
	for id := range h.clients {
		ids = append(ids, id)
	}
	return ids
}

func (h *Hub) PlayerList() []model.PlayerInfo {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.buildPlayerList()
}

// 参加順（JoinSeq 昇順）でプレイヤー一覧を返す。ターン制ゲームの進行順に使う。
func (h *Hub) OrderedPlayers() []model.PlayerInfo {
	h.mu.Lock()
	defer h.mu.Unlock()
	clients := h.copyClients()
	sort.Slice(clients, func(i, j int) bool {
		return clients[i].JoinSeq < clients[j].JoinSeq
	})
	list := make([]model.PlayerInfo, 0, len(clients))
	for _, c := range clients {
		list = append(list, model.PlayerInfo{
			PlayerID: c.PlayerID,
			Nickname: c.Nickname,
			IsHost:   c.IsHost,
			JoinSeq:  c.JoinSeq,
		})
	}
	return list
}

func (h *Hub) SetState(state string) {
	h.mu.Lock()
	h.state = state
	h.mu.Unlock()
}

// 参加者一覧を作る（呼び出し側で mu ロック済み前提）
func (h *Hub) buildPlayerList() []model.PlayerInfo {
	list := make([]model.PlayerInfo, 0, len(h.clients))
	for _, c := range h.clients {
		list = append(list, model.PlayerInfo{
			PlayerID: c.PlayerID,
			Nickname: c.Nickname,
			IsHost:   c.IsHost,
			JoinSeq:  c.JoinSeq,
		})
	}
	return list
}

// clients を slice にコピー（呼び出し側で mu ロック済み前提）
func (h *Hub) copyClients() []*Client {
	list := make([]*Client, 0, len(h.clients))
	for _, c := range h.clients {
		list = append(list, c)
	}
	return list
}

// 受信データを解析し type ごとに振り分ける
func (h *Hub) HandleMessage(client *Client, data []byte) {
	var msg model.IncomingMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		h.SendError(client.PlayerID, "invalid message format")
		return
	}

	switch msg.Type {
	case model.MsgRoomJoin:
		h.handleRoomJoin(client, msg.Payload)
	case model.MsgGameStart:
		h.handleGameStart(client, msg.Payload)
	default:
		// それ以外（game:vote / game:answer など）は進行中ゲームへ委譲する
		h.handleGameMessage(client, msg.Type, msg.Payload)
	}
}

// ゲーム固有メッセージをゲームロジックへ委譲する。type の解釈は各ゲームに任せる。
func (h *Hub) handleGameMessage(client *Client, msgType string, payload json.RawMessage) {
	h.mu.Lock()
	game := h.game
	h.mu.Unlock()

	if game == nil {
		h.SendError(client.PlayerID, "game has not started")
		return
	}
	if err := game.HandleMessage(h, client.PlayerID, msgType, payload); err != nil {
		h.SendError(client.PlayerID, err.Error())
	}
}

// 入室処理。本人へ room:joined、全員へ room:player_joined を送る。
func (h *Hub) handleRoomJoin(client *Client, payload json.RawMessage) {
	var p model.RoomJoinPayload
	if err := json.Unmarshal(payload, &p); err != nil || p.Nickname == "" {
		h.SendError(client.PlayerID, "invalid room:join payload")
		return
	}

	h.mu.Lock()
	// 登録後〜入室前にゲームが始まった場合、初回入室（名前未設定）は弾く。
	// リネーム用の room:join 再送（名前設定済み）は通す。
	// ルームは壊さず本人だけ外して切断する（切断時の Unregister は未登録なら何もしない）。
	if h.state != StateWaiting && client.Nickname == "" {
		delete(h.clients, client.PlayerID)
		h.mu.Unlock()
		client.Send(&model.OutgoingMessage{
			Type:    model.MsgError,
			Payload: model.ErrorPayload{Message: "game already in progress"},
		})
		client.Close("game already in progress")
		return
	}
	client.Nickname = p.Nickname
	players := h.buildPlayerList()
	h.mu.Unlock()

	client.Send(&model.OutgoingMessage{
		Type: model.MsgRoomJoined,
		Payload: model.RoomJoinedPayload{
			PlayerID: client.PlayerID,
			IsHost:   client.IsHost,
			Players:  players,
		},
	})

	h.Broadcast(&model.OutgoingMessage{
		Type: model.MsgRoomPlayerJoined,
		Payload: model.RoomPlayerJoinedPayload{
			PlayerID: client.PlayerID,
			Nickname: client.Nickname,
			Players:  players,
		},
	})
}

// ゲーム開始。ホスト以外・進行中は弾く。payload の game_mode で進行ロジックを選ぶ。
func (h *Hub) handleGameStart(client *Client, payload json.RawMessage) {
	var p model.GameStartPayload
	if err := json.Unmarshal(payload, &p); err != nil || p.GameMode == "" {
		h.SendError(client.PlayerID, "invalid game:start payload")
		return
	}

	h.mu.Lock()
	if client.PlayerID != h.hostID {
		h.mu.Unlock()
		h.SendError(client.PlayerID, "only the host can start the game")
		return
	}
	if h.state == StatePlaying { // 二重開始防止（finished からの再スタートは許可）
		h.mu.Unlock()
		h.SendError(client.PlayerID, "game already in progress")
		return
	}
	// 接続直後でまだ room:join が届いていない参加者がいる間は開始しない
	// （名前なしのプレイヤーがゲームに巻き込まれるのを防ぐ）
	for _, c := range h.clients {
		if c.Nickname == "" {
			h.mu.Unlock()
			h.SendError(client.PlayerID, "a player is still joining")
			return
		}
	}
	game, err := newGame(p.GameMode, h.questionRepo)
	if err != nil {
		h.mu.Unlock()
		h.SendError(client.PlayerID, err.Error())
		return
	}
	h.state = StatePlaying
	h.gameMode = p.GameMode
	h.game = game
	h.mu.Unlock()

	// 進行は時間がかかるので別 goroutine で回す。失敗したら waiting に戻す。
	go func() {
		if err := game.Start(h); err != nil {
			h.SendError(client.PlayerID, err.Error())
			h.SetState(StateWaiting)
		}
	}()
}
