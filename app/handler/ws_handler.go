package handler

import (
	"house-hunt/service"

	"github.com/coder/websocket"
	"github.com/gin-gonic/gin"
)

// WebSocket 接続の入口。全ルームを管理する RoomManager を1つ持つ。
type WSHandler struct {
	RoomManager *service.RoomManager
}

func NewWSHandler(rm *service.RoomManager) *WSHandler {
	return &WSHandler{RoomManager: rm}
}

// GET /ws/rooms/:roomID。1接続 = 1プレイヤーで、切断まで動き続ける。
func (h *WSHandler) Connect(c *gin.Context) {
	roomID := c.Param("roomID")

	// HTTP を WebSocket にアップグレード。InsecureSkipVerify は Origin チェック無効（開発用）。
	conn, err := websocket.Accept(c.Writer, c.Request, &websocket.AcceptOptions{
		InsecureSkipVerify: true,
	})
	if err != nil {
		return
	}

	// ルーム取得（無ければ作成）。isHost = 最初の接続者か。
	hub, isHost := h.RoomManager.GetOrCreateRoom(roomID)
	client := service.NewClient(conn)

	hub.Register(client, isHost)
	defer hub.Unregister(client) // 切断時にルーム破棄・通知を行う

	// 受信ループ。メッセージが来るたび Hub へ渡す。切断で err が返り抜ける。
	for {
		_, data, err := conn.Read(client.Context())
		if err != nil {
			return
		}
		hub.HandleMessage(client, data)
	}
}
