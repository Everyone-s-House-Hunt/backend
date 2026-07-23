package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/coder/websocket"
	"github.com/google/uuid"
	"house-hunt/model"
)

// ゲームモードを差し替え可能にするインターフェース。他モードもこれを実装すれば追加できる。
type GameLogic interface {
	Start(hub *Hub, ctx context.Context, runID uint64) error
	// ゲーム中のクライアントメッセージを処理する。type の解釈は各ゲームに委ねる。
	HandleMessage(hub *Hub, runID uint64, playerID, msgType string, payload json.RawMessage) error
}

// 途中退出後も続行できるゲームだけが実装する任意インターフェース。
// Hubの配信ロック中に呼ばれるため、このメソッド内ではBroadcastしない。
type GamePlayerLeaveHandler interface {
	HandlePlayerLeave(playerID string, players []model.PlayerInfo) model.GamePlayerLeftPayload
}

// game_mode 文字列から対応するゲームを生成する。新モードはここに足す。
func newGame(mode string, qr QuestionRepo) (GameLogic, error) {
	switch mode {
	case inoshishiGameMode:
		return NewInoshishiPanic(qr), nil
	case orderGameMode:
		return NewMojiOrder(qr), nil
	case piaceGameMode:
		return NewPiace(qr), nil
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
	sendFunc func(*model.OutgoingMessage) error
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
	c.writeMu.Lock()
	defer c.writeMu.Unlock()
	if c.sendFunc != nil {
		return c.sendFunc(msg)
	}
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	return c.Conn.Write(c.ctx, websocket.MessageText, data)
}

// 接続を閉じて context をキャンセル（受信ループも抜ける）
func (c *Client) Close(reason string) {
	if c.Conn != nil {
		c.Conn.Close(websocket.StatusNormalClosure, reason)
	}
	if c.cancel != nil {
		c.cancel()
	}
}

func (c *Client) KeepAlive(interval, timeout time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-c.ctx.Done():
			return
		case <-ticker.C:
			ctx, cancel := context.WithTimeout(c.ctx, timeout)
			err := c.Conn.Ping(ctx)
			cancel()
			if err != nil {
				c.Close("ping timeout")
				return
			}
		}
	}
}

// ルームの状態: waiting → playing → finished
const (
	StateWaiting    = "waiting"
	StatePlaying    = "playing"
	StateCancelling = "cancelling"
	StateFinished   = "finished"
	maxRoomPlayers  = 5
)

var (
	ErrRoomUnavailable = errors.New("room not found")
	ErrRoomFull        = errors.New("room is full")
	ErrGameInProgress  = errors.New("game already in progress")
)

// 1ルーム内の全クライアントを管理する
type Hub struct {
	RoomID        string
	state         string
	clients       map[string]*Client // プレイヤーID → Client
	hostID        string
	destroyed     bool // 二重破棄防止
	mu            sync.Mutex
	broadcastMu   sync.Mutex // 全クライアントへのイベント順を直列化
	game          GameLogic  // 未開始なら nil
	gameMode      string     // 開始時に確定するゲームモード
	gameCtx       context.Context
	gameCancel    context.CancelFunc
	gameRunID     uint64 // 実行中ゲームを識別し、古いgoroutineの送信を抑止する
	nextGameRunID uint64
	joinCounter   int // JoinSeq 採番用カウンタ
	questionRepo  QuestionRepo
	roomManager   *RoomManager
	ctx           context.Context // 破棄で cancel される
	cancel        context.CancelFunc
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

// クライアントを登録。破棄済みルームと満室ルームには入れない。
// 人数判定と登録を同じロック内で行い、同時接続でも5人を超えないようにする。
func (h *Hub) Register(client *Client, isHost bool) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.destroyed {
		return ErrRoomUnavailable
	}
	if h.state == StatePlaying || h.state == StateCancelling {
		return ErrGameInProgress
	}
	if len(h.clients) >= maxRoomPlayers {
		return ErrRoomFull
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

// クライアントを除去する。残りがいれば接続を維持し、ホスト退出時は参加順で引き継ぐ。
// 対応ゲームは残ったメンバーで続行し、それ以外はゲームだけを中断してルームへ戻す。
func (h *Hub) Unregister(client *Client) {
	// 一覧更新とゲーム中断の通知順を全クライアントで揃える。
	h.broadcastMu.Lock()
	defer h.broadcastMu.Unlock()

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
	if client.cancel != nil {
		client.cancel()
	}

	// 最後の1人が抜けたときだけルームそのものを削除する。
	if len(h.clients) == 0 {
		h.destroyed = true
		cancelGame := h.gameCancel
		h.game = nil
		h.gameMode = ""
		h.gameCtx = nil
		h.gameCancel = nil
		h.gameRunID = 0
		h.mu.Unlock()

		if cancelGame != nil {
			cancelGame()
		}
		if h.roomManager != nil {
			h.roomManager.DeleteRoom(h.RoomID)
		}
		h.cancel() // 進行中ゲームも中断
		return
	}

	if client.PlayerID == h.hostID {
		h.promoteNextHostLocked()
	}

	players := h.buildPlayerList()
	remaining := h.copyClients()
	wasPlaying := h.state == StatePlaying
	game := h.game
	leaveHandler, canContinue := game.(GamePlayerLeaveHandler)
	var cancelGame context.CancelFunc
	if wasPlaying && !canContinue {
		h.state = StateCancelling
		cancelGame = h.gameCancel
		// 先にrunを無効化することで、cancelと競合した旧ゲームイベントを抑止する。
		h.gameRunID = 0
		h.gameCancel = nil
		h.gameCtx = nil
		h.game = nil
		h.gameMode = ""
	}
	h.mu.Unlock()

	if cancelGame != nil {
		cancelGame()
	}

	var gamePlayerLeft *model.GamePlayerLeftPayload
	if wasPlaying && canContinue {
		payload := leaveHandler.HandlePlayerLeave(client.PlayerID, players)
		payload.DisconnectedNickname = client.Nickname
		gamePlayerLeft = &payload
	}

	sendToClients(remaining, &model.OutgoingMessage{
		Type: model.MsgRoomPlayerLeft,
		Payload: model.RoomPlayerLeftPayload{
			PlayerID: client.PlayerID,
			Nickname: client.Nickname,
			Players:  players,
		},
	})

	if gamePlayerLeft != nil {
		sendToClients(remaining, &model.OutgoingMessage{
			Type:    model.MsgGamePlayerLeft,
			Payload: *gamePlayerLeft,
		})
	} else if wasPlaying {
		sendToClients(remaining, &model.OutgoingMessage{
			Type: model.MsgGameCancelled,
			Payload: model.GameCancelledPayload{
				Reason:               "player_disconnected",
				DisconnectedPlayerID: client.PlayerID,
			},
		})

		h.mu.Lock()
		if h.state == StateCancelling {
			h.state = StateWaiting
		}
		h.mu.Unlock()
	}
}

// 全員に同じメッセージを送り、別イベントとの配信順を揃える。
func (h *Hub) Broadcast(msg *model.OutgoingMessage) {
	h.broadcastMu.Lock()
	defer h.broadcastMu.Unlock()

	h.mu.Lock()
	clients := h.copyClients() // ロックを長く持たないようコピーしてから送る
	h.mu.Unlock()
	sendToClients(clients, msg)
}

// 実行中runに属するゲームイベントだけを配信する。
// 退出でrunが無効化された後のタイマー・回答イベントはここで捨てる。
func (h *Hub) BroadcastGame(runID uint64, msg *model.OutgoingMessage) bool {
	h.broadcastMu.Lock()
	defer h.broadcastMu.Unlock()

	h.mu.Lock()
	if h.destroyed || h.state != StatePlaying || h.gameRunID != runID {
		h.mu.Unlock()
		return false
	}
	clients := h.copyClients()
	h.mu.Unlock()
	sendToClients(clients, msg)
	return true
}

func sendToClients(clients []*Client, msg *model.OutgoingMessage) {
	for _, c := range clients {
		_ = c.Send(msg)
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

// ホスト退出後、参加順が最も早いプレイヤーを新ホストにする（muロック済み前提）。
func (h *Hub) promoteNextHostLocked() {
	var next *Client
	for _, c := range h.clients {
		c.IsHost = false
		if next == nil || c.JoinSeq < next.JoinSeq {
			next = c
		}
	}
	if next == nil {
		h.hostID = ""
		return
	}
	next.IsHost = true
	h.hostID = next.PlayerID
}

// ゲームgoroutineの完了を反映する。runが既に中断・更新済みなら何もしない。
func (h *Hub) finishGame(runID uint64, starterID string, err error) {
	h.broadcastMu.Lock()
	defer h.broadcastMu.Unlock()

	h.mu.Lock()
	if h.gameRunID != runID {
		h.mu.Unlock()
		return
	}
	target := h.clients[starterID]
	if err != nil {
		h.state = StateWaiting
	} else {
		h.state = StateFinished
	}
	h.game = nil
	h.gameMode = ""
	h.gameCtx = nil
	h.gameCancel = nil
	h.gameRunID = 0
	h.mu.Unlock()

	if err != nil && target != nil {
		_ = target.Send(&model.OutgoingMessage{
			Type:    model.MsgError,
			Payload: model.ErrorPayload{Message: err.Error()},
		})
	}
}

// runが今も実行中の場合だけ、ゲーム由来のエラーを送る。
func (h *Hub) sendGameError(runID uint64, playerID, message string) {
	h.broadcastMu.Lock()
	defer h.broadcastMu.Unlock()

	h.mu.Lock()
	if h.state != StatePlaying || h.gameRunID != runID {
		h.mu.Unlock()
		return
	}
	target := h.clients[playerID]
	h.mu.Unlock()
	if target != nil {
		_ = target.Send(&model.OutgoingMessage{
			Type:    model.MsgError,
			Payload: model.ErrorPayload{Message: message},
		})
	}
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
	runID := h.gameRunID
	state := h.state
	h.mu.Unlock()

	if game == nil || state != StatePlaying {
		h.SendError(client.PlayerID, "game has not started")
		return
	}
	if err := game.HandleMessage(h, runID, client.PlayerID, msgType, payload); err != nil {
		h.sendGameError(runID, client.PlayerID, err.Error())
	}
}

// 入室処理。本人へ room:joined、全員へ room:player_joined を送る。
func (h *Hub) handleRoomJoin(client *Client, payload json.RawMessage) {
	var p model.RoomJoinPayload
	if err := json.Unmarshal(payload, &p); err != nil || p.Nickname == "" {
		h.SendError(client.PlayerID, "invalid room:join payload")
		return
	}

	// ニックネーム更新と一覧配信を退出イベントと直列化する。
	h.broadcastMu.Lock()
	defer h.broadcastMu.Unlock()

	h.mu.Lock()
	if h.destroyed {
		h.mu.Unlock()
		return
	}
	if _, exists := h.clients[client.PlayerID]; !exists {
		h.mu.Unlock()
		return
	}
	client.Nickname = p.Nickname
	players := h.buildPlayerList()
	clients := h.copyClients()
	h.mu.Unlock()

	_ = client.Send(&model.OutgoingMessage{
		Type: model.MsgRoomJoined,
		Payload: model.RoomJoinedPayload{
			PlayerID: client.PlayerID,
			IsHost:   client.IsHost,
			Players:  players,
		},
	})

	sendToClients(clients, &model.OutgoingMessage{
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
	if h.state == StatePlaying || h.state == StateCancelling { // 二重開始・中断中の再開防止
		h.mu.Unlock()
		h.SendError(client.PlayerID, "game already in progress")
		return
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
	previousCancel := h.gameCancel
	gameCtx, gameCancel := context.WithCancel(h.ctx)
	h.nextGameRunID++
	runID := h.nextGameRunID
	h.gameCtx = gameCtx
	h.gameCancel = gameCancel
	h.gameRunID = runID
	h.mu.Unlock()

	if previousCancel != nil {
		previousCancel()
	}

	// 進行は時間がかかるので別goroutineで回す。完了処理はrun ID一致時だけ反映する。
	go func() {
		err := game.Start(h, gameCtx, runID)
		if gameCtx.Err() != nil {
			err = nil // 退出・ルーム破棄によるキャンセルはユーザー向けエラーにしない
		}
		h.finishGame(runID, client.PlayerID, err)
		gameCancel()
	}()
}
