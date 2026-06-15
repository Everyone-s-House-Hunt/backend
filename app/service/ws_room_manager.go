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

// roomID のルームを返す（無ければ作成）。bool は「作成者＝ホストか」。
func (rm *RoomManager) GetOrCreateRoom(roomID string) (*Hub, bool) {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	if hub, ok := rm.rooms[roomID]; ok {
		return hub, false // 既存ルーム → 参加者
	}

	hub := NewHub(roomID, rm, rm.questionRepo)
	rm.rooms[roomID] = hub
	return hub, true // 新規作成 → ホスト
}

// ルームを管理対象から外す（破棄時に Hub から呼ばれる）
func (rm *RoomManager) DeleteRoom(roomID string) {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	delete(rm.rooms, roomID)
}
