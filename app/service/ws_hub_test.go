package service

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"

	"house-hunt/model"
)

type messageRecorder struct {
	mu       sync.Mutex
	messages []model.OutgoingMessage
}

func (r *messageRecorder) record(msg *model.OutgoingMessage) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.messages = append(r.messages, *msg)
	return nil
}

func (r *messageRecorder) types() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	types := make([]string, len(r.messages))
	for i, msg := range r.messages {
		types[i] = msg.Type
	}
	return types
}

func newHubTestClient(id, nickname string) (*Client, *messageRecorder) {
	ctx, cancel := context.WithCancel(context.Background())
	recorder := &messageRecorder{}
	return &Client{
		PlayerID: id,
		Nickname: nickname,
		ctx:      ctx,
		cancel:   cancel,
		sendFunc: recorder.record,
	}, recorder
}

func assertContextActive(t *testing.T, ctx context.Context) {
	t.Helper()
	select {
	case <-ctx.Done():
		t.Fatal("context was unexpectedly cancelled")
	default:
	}
}

func TestRegisterLimitsRoomToFiveConcurrently(t *testing.T) {
	hub := NewHub("room", nil, nil)
	host, _ := newHubTestClient("host", "ホスト")
	if err := hub.Register(host, true); err != nil {
		t.Fatalf("register host: %v", err)
	}

	var successCount atomic.Int32
	var fullCount atomic.Int32
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			client, _ := newHubTestClient(fmt.Sprintf("guest-%d", index), "ゲスト")
			switch err := hub.Register(client, false); {
			case err == nil:
				successCount.Add(1)
			case errors.Is(err, ErrRoomFull):
				fullCount.Add(1)
			default:
				t.Errorf("unexpected register error: %v", err)
			}
		}(i)
	}
	wg.Wait()

	if got := hub.PlayerCount(); got != maxRoomPlayers {
		t.Fatalf("player count = %d, want %d", got, maxRoomPlayers)
	}
	if got := successCount.Load(); got != maxRoomPlayers-1 {
		t.Fatalf("successful guests = %d, want %d", got, maxRoomPlayers-1)
	}
	if got := fullCount.Load(); got != 10-(maxRoomPlayers-1) {
		t.Fatalf("full errors = %d, want %d", got, 10-(maxRoomPlayers-1))
	}
	assertContextActive(t, host.Context())
}

func TestRegisterRejectsJoinWhileGameIsRunning(t *testing.T) {
	hub := NewHub("room", nil, nil)
	hub.state = StatePlaying
	client, _ := newHubTestClient("guest", "ゲスト")

	if err := hub.Register(client, false); !errors.Is(err, ErrGameInProgress) {
		t.Fatalf("Register() error = %v, want %v", err, ErrGameInProgress)
	}
	if got := hub.PlayerCount(); got != 0 {
		t.Fatalf("player count = %d, want 0", got)
	}
}

func TestGuestLeaveKeepsRoomAndRemainingConnections(t *testing.T) {
	rm := NewRoomManager(nil)
	host, hostMessages := newHubTestClient("host", "ホスト")
	hub, ok := rm.CreateRoomWithHost("room", host)
	if !ok {
		t.Fatal("failed to create room")
	}
	leaver, _ := newHubTestClient("guest-1", "ゲスト1")
	remaining, remainingMessages := newHubTestClient("guest-2", "ゲスト2")
	if err := hub.Register(leaver, false); err != nil {
		t.Fatalf("register leaver: %v", err)
	}
	if err := hub.Register(remaining, false); err != nil {
		t.Fatalf("register remaining: %v", err)
	}

	hub.Unregister(leaver)

	if got := hub.PlayerCount(); got != 2 {
		t.Fatalf("player count = %d, want 2", got)
	}
	if _, exists := rm.GetRoom("room"); !exists {
		t.Fatal("room was deleted after a guest left")
	}
	assertContextActive(t, host.Context())
	assertContextActive(t, remaining.Context())
	if got := hostMessages.types(); len(got) != 1 || got[0] != model.MsgRoomPlayerLeft {
		t.Fatalf("host messages = %v, want [%s]", got, model.MsgRoomPlayerLeft)
	}
	if got := remainingMessages.types(); len(got) != 1 || got[0] != model.MsgRoomPlayerLeft {
		t.Fatalf("remaining messages = %v, want [%s]", got, model.MsgRoomPlayerLeft)
	}
}

func TestHostLeaveTransfersHostToEarliestRemainingPlayer(t *testing.T) {
	rm := NewRoomManager(nil)
	host, _ := newHubTestClient("host", "ホスト")
	hub, ok := rm.CreateRoomWithHost("room", host)
	if !ok {
		t.Fatal("failed to create room")
	}
	nextHost, nextHostMessages := newHubTestClient("guest-1", "ゲスト1")
	other, otherMessages := newHubTestClient("guest-2", "ゲスト2")
	if err := hub.Register(nextHost, false); err != nil {
		t.Fatalf("register next host: %v", err)
	}
	if err := hub.Register(other, false); err != nil {
		t.Fatalf("register other: %v", err)
	}

	hub.Unregister(host)

	if hub.hostID != nextHost.PlayerID {
		t.Fatalf("host ID = %s, want %s", hub.hostID, nextHost.PlayerID)
	}
	if !nextHost.IsHost {
		t.Fatal("earliest remaining player was not promoted")
	}
	if other.IsHost {
		t.Fatal("later player was also marked as host")
	}
	if _, exists := rm.GetRoom("room"); !exists {
		t.Fatal("room was deleted after host transfer")
	}
	assertContextActive(t, nextHost.Context())
	assertContextActive(t, other.Context())
	if got := nextHostMessages.types(); len(got) != 1 || got[0] != model.MsgRoomPlayerLeft {
		t.Fatalf("next host messages = %v", got)
	}
	if got := otherMessages.types(); len(got) != 1 || got[0] != model.MsgRoomPlayerLeft {
		t.Fatalf("other messages = %v", got)
	}
}

func TestHostLeaveDuringGameTransfersHostAndCancelsGame(t *testing.T) {
	rm := NewRoomManager(nil)
	host, _ := newHubTestClient("host", "ホスト")
	hub, ok := rm.CreateRoomWithHost("room", host)
	if !ok {
		t.Fatal("failed to create room")
	}
	nextHost, recorder := newHubTestClient("guest", "ゲスト")
	if err := hub.Register(nextHost, false); err != nil {
		t.Fatalf("register guest: %v", err)
	}

	gameCtx, gameCancel := context.WithCancel(hub.Context())
	hub.mu.Lock()
	hub.state = StatePlaying
	hub.gameCtx = gameCtx
	hub.gameCancel = gameCancel
	hub.gameRunID = 11
	hub.mu.Unlock()

	hub.Unregister(host)

	if hub.hostID != nextHost.PlayerID || !nextHost.IsHost {
		t.Fatal("remaining player was not promoted to host")
	}
	assertContextActive(t, nextHost.Context())
	if got := recorder.types(); len(got) != 2 ||
		got[0] != model.MsgRoomPlayerLeft || got[1] != model.MsgGameCancelled {
		t.Fatalf("messages = %v, want [%s %s]", got, model.MsgRoomPlayerLeft, model.MsgGameCancelled)
	}
	hub.mu.Lock()
	state := hub.state
	hub.mu.Unlock()
	if state != StateWaiting {
		t.Fatalf("state = %s, want %s", state, StateWaiting)
	}
}

func TestLastPlayerLeaveDeletesRoom(t *testing.T) {
	rm := NewRoomManager(nil)
	host, _ := newHubTestClient("host", "ホスト")
	hub, ok := rm.CreateRoomWithHost("room", host)
	if !ok {
		t.Fatal("failed to create room")
	}

	hub.Unregister(host)

	if _, exists := rm.GetRoom("room"); exists {
		t.Fatal("empty room was not deleted")
	}
	select {
	case <-hub.Context().Done():
	default:
		t.Fatal("room context was not cancelled")
	}
}

func TestLeaveDuringGameCancelsRunAndReturnsRoomToWaiting(t *testing.T) {
	rm := NewRoomManager(nil)
	host, hostMessages := newHubTestClient("host", "ホスト")
	hub, ok := rm.CreateRoomWithHost("room", host)
	if !ok {
		t.Fatal("failed to create room")
	}
	leaver, _ := newHubTestClient("guest", "ゲスト")
	if err := hub.Register(leaver, false); err != nil {
		t.Fatalf("register guest: %v", err)
	}

	gameCtx, gameCancel := context.WithCancel(hub.Context())
	hub.mu.Lock()
	hub.state = StatePlaying
	hub.gameCtx = gameCtx
	hub.gameCancel = gameCancel
	hub.gameRunID = 7
	hub.mu.Unlock()

	hub.Unregister(leaver)

	select {
	case <-gameCtx.Done():
	default:
		t.Fatal("game context was not cancelled")
	}
	hub.mu.Lock()
	state := hub.state
	runID := hub.gameRunID
	hub.mu.Unlock()
	if state != StateWaiting {
		t.Fatalf("state = %s, want %s", state, StateWaiting)
	}
	if runID != 0 {
		t.Fatalf("run ID = %d, want 0", runID)
	}
	if got := hostMessages.types(); len(got) != 2 ||
		got[0] != model.MsgRoomPlayerLeft || got[1] != model.MsgGameCancelled {
		t.Fatalf("messages = %v, want [%s %s]", got, model.MsgRoomPlayerLeft, model.MsgGameCancelled)
	}

	before := len(hostMessages.types())
	if sent := hub.BroadcastGame(7, &model.OutgoingMessage{
		Type:    model.MsgGameOver,
		Payload: model.GameOverPayload{Reason: "timeout"},
	}); sent {
		t.Fatal("stale game event was sent")
	}
	if after := len(hostMessages.types()); after != before {
		t.Fatalf("message count after stale event = %d, want %d", after, before)
	}

	replacement, _ := newHubTestClient("replacement", "補充")
	if err := hub.Register(replacement, false); err != nil {
		t.Fatalf("register replacement after cancellation: %v", err)
	}
}

func TestRoomPlayerLeftPayloadContainsUpdatedHostAndPlayers(t *testing.T) {
	rm := NewRoomManager(nil)
	host, _ := newHubTestClient("host", "ホスト")
	hub, ok := rm.CreateRoomWithHost("room", host)
	if !ok {
		t.Fatal("failed to create room")
	}
	guest, recorder := newHubTestClient("guest", "ゲスト")
	if err := hub.Register(guest, false); err != nil {
		t.Fatalf("register guest: %v", err)
	}

	hub.Unregister(host)

	recorder.mu.Lock()
	defer recorder.mu.Unlock()
	if len(recorder.messages) != 1 {
		t.Fatalf("message count = %d, want 1", len(recorder.messages))
	}
	if recorder.messages[0].Type != model.MsgRoomPlayerLeft {
		t.Fatalf("message type = %s, want %s", recorder.messages[0].Type, model.MsgRoomPlayerLeft)
	}
	payload, ok := recorder.messages[0].Payload.(model.RoomPlayerLeftPayload)
	if !ok {
		t.Fatalf("payload type = %T", recorder.messages[0].Payload)
	}
	if payload.PlayerID != host.PlayerID || payload.Nickname != host.Nickname {
		t.Fatalf("left player = (%s, %s)", payload.PlayerID, payload.Nickname)
	}
	if len(payload.Players) != 1 || payload.Players[0].PlayerID != guest.PlayerID ||
		!payload.Players[0].IsHost {
		t.Fatalf("updated players = %+v", payload.Players)
	}
}
