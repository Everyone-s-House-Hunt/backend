package service

import "sync"

// 全ルーム(Hub)をメモリ上で管理する。アプリ起動時に1つだけ生成され全接続で共有される。
type RoomManager struct {
	rooms        map[string]*Hub // ルームID → Hub
	mu           sync.RWMutex
	questionRepo QuestionRepo
}

func NewRoomManager(qr QuestionRepo) *RoomManager {
	return &RoomManager{
		rooms:        make(map[string]*Hub),
		questionRepo: qr,
	}
}

// roomID のルームを新規作成する。既に存在する場合は nil, false（ID衝突。呼び出し側で弾く）。
func (rm *RoomManager) CreateRoom(roomID string) (*Hub, bool) {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	if _, ok := rm.rooms[roomID]; ok {
		return nil, false
	}

	hub := NewHub(roomID, rm, rm.questionRepo)
	rm.rooms[roomID] = hub
	return hub, true
}

// roomID のルームを返す（作成はしない）。参加は既存ルームに限る。
func (rm *RoomManager) GetRoom(roomID string) (*Hub, bool) {
	rm.mu.RLock()
	defer rm.mu.RUnlock()
	hub, ok := rm.rooms[roomID]
	return hub, ok
}

// ルームを管理対象から外す（破棄時に Hub から呼ばれる）
func (rm *RoomManager) DeleteRoom(roomID string) {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	delete(rm.rooms, roomID)
}
