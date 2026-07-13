package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"house-hunt/model"
	"house-hunt/service"

	"github.com/coder/websocket"
	"github.com/gin-gonic/gin"
)

type bulletQuestionRepo struct{}

func (bulletQuestionRepo) GetRandomByGameMode(gameMode string, _ int) ([]model.Question, error) {
	if gameMode != "bullet" {
		return nil, fmt.Errorf("unexpected game mode %s", gameMode)
	}
	answers := make([]string, 10)
	aliases := make(map[string][]string)
	for index := range answers {
		answers[index] = fmt.Sprintf("answer-%d", index+1)
	}
	aliases[answers[0]] = []string{"first-alias"}
	answerData, _ := json.Marshal(map[string]interface{}{
		"question": "10 answers",
		"answers":  answers,
		"aliases":  aliases,
	})
	return []model.Question{{Body: "fallback", AnswerData: string(answerData)}}, nil
}

type wsEnvelope struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

func TestBulletWebSocketRelayWithTwoPlayers(t *testing.T) {
	gin.SetMode(gin.TestMode)
	manager := service.NewRoomManager(bulletQuestionRepo{})
	handler := NewWSHandler(manager)
	router := gin.New()
	router.GET("/ws/rooms/:roomID", handler.Connect)
	server := httptest.NewServer(router)
	defer server.Close()

	baseURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws/rooms/123456"
	host := dialWS(t, baseURL+"?create=1")
	defer host.Close(websocket.StatusNormalClosure, "test complete")
	sendWS(t, host, "room:join", map[string]string{"nickname": "host"})
	hostJoined := readWS(t, host, "room:joined")
	var hostPayload model.RoomJoinedPayload
	decodePayload(t, hostJoined, &hostPayload)

	guest := dialWS(t, baseURL)
	defer guest.Close(websocket.StatusNormalClosure, "test complete")
	sendWS(t, guest, "room:join", map[string]string{"nickname": "guest"})
	guestJoined := readWS(t, guest, "room:joined")
	var guestPayload model.RoomJoinedPayload
	decodePayload(t, guestJoined, &guestPayload)

	sendWS(t, host, "game:start", map[string]string{"game_mode": "bullet"})
	hostStart := readWS(t, host, "game:bullet_start")
	guestStart := readWS(t, guest, "game:bullet_start")
	var start model.BulletStartPayload
	decodePayload(t, hostStart, &start)
	if start.CurrentPlayerID != hostPayload.PlayerID || start.TargetHits != 10 {
		t.Fatalf("unexpected start payload: %#v", start)
	}
	var guestStartPayload model.BulletStartPayload
	decodePayload(t, guestStart, &guestStartPayload)

	answers := []string{"first-alias", "answer-2", "answer-3", "answer-4", "answer-5", "answer-6", "answer-7", "answer-8", "answer-9", "answer-10"}
	currentConn := host
	currentPlayerID := hostPayload.PlayerID
	otherPlayerID := guestPayload.PlayerID

	for index, answer := range answers {
		if index == 1 {
			sendWS(t, currentConn, "game:bullet_submit", map[string]string{"answer": "wrong"})
			missForHost := readWS(t, host, "game:bullet_miss")
			readWS(t, guest, "game:bullet_miss")
			var miss model.BulletMissPayload
			decodePayload(t, missForHost, &miss)
			if miss.CurrentPlayerID != currentPlayerID || miss.Reason != "wrong" {
				t.Fatalf("miss changed the turn: %#v", miss)
			}
		}

		sendWS(t, currentConn, "game:bullet_submit", map[string]string{"answer": answer})
		hitForHost := readWS(t, host, "game:bullet_hit")
		readWS(t, guest, "game:bullet_hit")
		var hit model.BulletHitPayload
		decodePayload(t, hitForHost, &hit)
		if hit.CorrectCount != index+1 || len(hit.Used) != index+1 {
			t.Fatalf("unexpected hit progress: %#v", hit)
		}
		if index == 0 && (hit.Answer != "answer-1" || hit.Used[0] != "answer-1") {
			t.Fatalf("alias did not resolve to the canonical label: %#v", hit)
		}

		currentPlayerID, otherPlayerID = otherPlayerID, currentPlayerID
		if currentConn == host {
			currentConn = guest
		} else {
			currentConn = host
		}
		if hit.CurrentPlayerID != currentPlayerID {
			t.Fatalf("turn did not rotate: %#v", hit)
		}
	}

	readWS(t, host, "game:clear")
	readWS(t, guest, "game:clear")
}

func dialWS(t *testing.T, url string) *websocket.Conn {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	conn, _, err := websocket.Dial(ctx, url, nil)
	if err != nil {
		t.Fatalf("dial %s: %v", url, err)
	}
	return conn
}

func sendWS(t *testing.T, conn *websocket.Conn, messageType string, payload interface{}) {
	t.Helper()
	data, err := json.Marshal(map[string]interface{}{"type": messageType, "payload": payload})
	if err != nil {
		t.Fatalf("marshal message: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := conn.Write(ctx, websocket.MessageText, data); err != nil {
		t.Fatalf("write %s: %v", messageType, err)
	}
}

func readWS(t *testing.T, conn *websocket.Conn, wantedType string) wsEnvelope {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	for {
		_, data, err := conn.Read(ctx)
		if err != nil {
			t.Fatalf("read %s: %v", wantedType, err)
		}
		var message wsEnvelope
		if err := json.Unmarshal(data, &message); err != nil {
			t.Fatalf("decode message: %v", err)
		}
		if message.Type == wantedType {
			return message
		}
	}
}

func decodePayload(t *testing.T, message wsEnvelope, target interface{}) {
	t.Helper()
	if err := json.Unmarshal(message.Payload, target); err != nil {
		t.Fatalf("decode %s payload: %v", message.Type, err)
	}
}
