package handler

import (
	"house-hunt/model"
	"house-hunt/service"

	"github.com/coder/websocket"
	"github.com/gin-gonic/gin"

	"time"
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
		created, ok := h.RoomManager.CreateRoomWithHost(roomID, client)
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
		// 参加はゲーム開始前かつ5人未満の場合だけ許可する。
		if err := hub.Register(client, false); err != nil {
			rejectAndClose(client, err.Error())
			return
		}
	}
	defer hub.Unregister(client) // 切断時に退出者だけを除去し、残りの接続は維持する

	go client.KeepAlive(30*time.Second, 10*time.Second)

	// 受信ループ。メッセージが来るたび Hub へ渡す。切断で err が返り抜ける。
	for {
		_, data, err := conn.Read(client.Context())
		if err != nil {
			return
		}
		hub.HandleMessage(client, data)
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
