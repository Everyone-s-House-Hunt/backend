package handler

import (
	"context"
	"time"

	"house-hunt/model"
	"house-hunt/service"

	"github.com/coder/websocket"
	"github.com/gin-gonic/gin"
)

// ハートビート設定。スマホのスリープ等の「静かな切断」を検知するため、
// サーバーから定期的に ping を打ち、応答がなければ切断扱いにする。
// 検知までの最悪時間 = pingInterval + pingTimeout（約40秒）。
const (
	pingInterval = 30 * time.Second // ping を打つ間隔
	pingTimeout  = 10 * time.Second // pong 応答の待ち時間
)

// WebSocket 接続の入口。全ルームを管理する RoomManager を1つ持つ。
type WSHandler struct {
	RoomManager *service.RoomManager
}

func NewWSHandler(rm *service.RoomManager) *WSHandler {
	return &WSHandler{RoomManager: rm}
}

// GET /ws/rooms/:roomID。1接続 = 1プレイヤーで、切断まで動き続ける。
// ?create=1 付きはルーム作成（ホスト）、無しは既存ルームへの参加のみ。
// 存在しないIDへの参加でルームが勝手に作られないよう、作成と参加を分けている。
func (h *WSHandler) Connect(c *gin.Context) {
	roomID := c.Param("roomID")
	isCreate := c.Query("create") == "1"

	// HTTP を WebSocket にアップグレード。InsecureSkipVerify は Origin チェック無効（開発用）。
	conn, err := websocket.Accept(c.Writer, c.Request, &websocket.AcceptOptions{
		InsecureSkipVerify: true,
	})
	if err != nil {
		return
	}

	client := service.NewClient(conn)

	var hub *service.Hub
	if isCreate {
		created, ok := h.RoomManager.CreateRoom(roomID)
		if !ok { // フロント生成の6桁IDが衝突。フロントは振り直して再試行する
			rejectAndClose(client, "room already exists")
			return
		}
		hub = created
	} else {
		existing, ok := h.RoomManager.GetRoom(roomID)
		if !ok {
			rejectAndClose(client, "room not found")
			return
		}
		hub = existing
	}

	// 作成者だけがホスト。破棄済み・ゲーム進行中のルームには入れない。
	if err := hub.Register(client, isCreate); err != nil {
		rejectAndClose(client, err.Error())
		return
	}
	defer hub.Unregister(client)         // 切断時にルーム破棄・通知を行う
	defer client.Close("handler closed") // context を cancel し pingLoop も確実に止める

	// ハートビート。ping が返らなくなったら Close し、下の受信ループを
	// エラーで抜けさせて通常の切断と同じ経路（Unregister）に乗せる。
	go pingLoop(client, conn)

	// 受信ループ。メッセージが来るたび Hub へ渡す。切断で err が返り抜ける。
	// pong の受信処理も conn.Read の内部で行われるため、このループが回っている限り ping に応答できる。
	for {
		_, data, err := conn.Read(client.Context())
		if err != nil {
			return
		}
		hub.HandleMessage(client, data)
	}
}

// pingInterval ごとに ping を打ち、pingTimeout 以内に pong が返らなければ切断する。
// client の context は切断（Close / 受信ループ終了）で cancel されるため、このループも一緒に止まる。
func pingLoop(client *service.Client, conn *websocket.Conn) {
	ticker := time.NewTicker(pingInterval)
	defer ticker.Stop()
	for {
		select {
		case <-client.Context().Done():
			return
		case <-ticker.C:
			ctx, cancel := context.WithTimeout(client.Context(), pingTimeout)
			err := conn.Ping(ctx)
			cancel()
			if err != nil {
				client.Close("ping timeout")
				return
			}
		}
	}
}

// 入室を断る。理由をエラーメッセージで伝えてから接続を閉じる
// （HTTPステータスと違いWSの切断理由はブラウザJSへ渡らないため、メッセージで送る）。
func rejectAndClose(client *service.Client, message string) {
	client.Send(&model.OutgoingMessage{
		Type:    model.MsgError,
		Payload: model.ErrorPayload{Message: message},
	})
	client.Close(message)
}
