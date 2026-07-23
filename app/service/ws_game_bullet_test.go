package service

import (
	"reflect"
	"testing"

	"house-hunt/model"
)

func TestBulletTurnIndex(t *testing.T) {
	tests := []struct {
		name        string
		playerCount int
		want        []int
	}{
		{name: "one player", playerCount: 1, want: []int{0, 0, 0, 0, 0, 0, 0, 0, 0, 0}},
		{name: "two players", playerCount: 2, want: []int{0, 1, 1, 0, 0, 1, 1, 0, 0, 1}},
		{name: "three players", playerCount: 3, want: []int{0, 1, 2, 2, 1, 0, 0, 1, 2, 2}},
		{name: "four players", playerCount: 4, want: []int{0, 1, 2, 3, 3, 2, 1, 0, 0, 1}},
		{name: "five players", playerCount: 5, want: []int{0, 1, 2, 3, 4, 4, 3, 2, 1, 0}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := make([]int, bulletTargetHits)
			for correctCount := range got {
				got[correctCount] = bulletTurnIndex(correctCount, tt.playerCount)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("bulletTurnIndex() sequence = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBulletTurnIndexHandlesInvalidPlayerCount(t *testing.T) {
	if got := bulletTurnIndex(4, 0); got != 0 {
		t.Fatalf("bulletTurnIndex(4, 0) = %d, want 0", got)
	}
}

func TestAdvanceBulletTurnMatchesPingPongSequence(t *testing.T) {
	for playerCount := 1; playerCount <= bulletMaxPlayers; playerCount++ {
		index := 0
		direction := 1
		got := make([]int, bulletTargetHits)
		for turn := range got {
			got[turn] = index
			index, direction = advanceBulletTurn(index, direction, playerCount)
		}

		want := make([]int, bulletTargetHits)
		for turn := range want {
			want[turn] = bulletTurnIndex(turn, playerCount)
		}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("playerCount=%d sequence=%v, want %v", playerCount, got, want)
		}
	}
}

func bulletPlayers(ids ...string) []model.PlayerInfo {
	players := make([]model.PlayerInfo, 0, len(ids))
	for index, id := range ids {
		players = append(players, model.PlayerInfo{
			PlayerID: id,
			Nickname: id,
			JoinSeq:  index,
		})
	}
	return players
}

func initializedBullet(ids ...string) *Bullet {
	game := NewBullet(nil)
	game.players = bulletPlayers(ids...)
	game.started = true
	game.turnDirection = 1
	game.used = map[string]bool{"回答済み": true}
	game.correctCount = 3
	return game
}

func TestBulletPlayerLeaveKeepsCurrentPlayerWhenWaitingPlayerLeaves(t *testing.T) {
	game := initializedBullet("a", "b", "c")
	game.turnIndex = 2
	game.turnDirection = -1

	payload := game.HandlePlayerLeave("a", bulletPlayers("b", "c"))

	if payload.CurrentPlayerID != "c" || game.turnIndex != 1 || game.turnDirection != -1 {
		t.Fatalf(
			"current=%s index=%d direction=%d, want c/1/-1",
			payload.CurrentPlayerID,
			game.turnIndex,
			game.turnDirection,
		)
	}
	if game.correctCount != 3 || !game.used["回答済み"] {
		t.Fatal("game progress was reset after player leave")
	}
}

func TestBulletCurrentPlayerLeaveMovesInCurrentDirection(t *testing.T) {
	tests := []struct {
		name          string
		turnIndex     int
		direction     int
		leaver        string
		remaining     []string
		wantCurrent   string
		wantIndex     int
		wantDirection int
	}{
		{
			name:          "move right",
			turnIndex:     1,
			direction:     1,
			leaver:        "b",
			remaining:     []string{"a", "c"},
			wantCurrent:   "c",
			wantIndex:     1,
			wantDirection: 1,
		},
		{
			name:          "move left",
			turnIndex:     1,
			direction:     -1,
			leaver:        "b",
			remaining:     []string{"a", "c"},
			wantCurrent:   "a",
			wantIndex:     0,
			wantDirection: -1,
		},
		{
			name:          "reverse at right edge",
			turnIndex:     2,
			direction:     1,
			leaver:        "c",
			remaining:     []string{"a", "b"},
			wantCurrent:   "b",
			wantIndex:     1,
			wantDirection: -1,
		},
		{
			name:          "reverse at left edge",
			turnIndex:     0,
			direction:     -1,
			leaver:        "a",
			remaining:     []string{"b", "c"},
			wantCurrent:   "b",
			wantIndex:     0,
			wantDirection: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			game := initializedBullet("a", "b", "c")
			game.turnIndex = tt.turnIndex
			game.turnDirection = tt.direction

			payload := game.HandlePlayerLeave(tt.leaver, bulletPlayers(tt.remaining...))

			if payload.CurrentPlayerID != tt.wantCurrent ||
				game.turnIndex != tt.wantIndex ||
				game.turnDirection != tt.wantDirection {
				t.Fatalf(
					"current=%s index=%d direction=%d, want %s/%d/%d",
					payload.CurrentPlayerID,
					game.turnIndex,
					game.turnDirection,
					tt.wantCurrent,
					tt.wantIndex,
					tt.wantDirection,
				)
			}
		})
	}
}

func TestBulletPlayerLeaveContinuesWithOnePlayer(t *testing.T) {
	game := initializedBullet("a", "b")
	game.turnIndex = 1

	payload := game.HandlePlayerLeave("b", bulletPlayers("a"))

	if payload.CurrentPlayerID != "a" || game.turnIndex != 0 || game.turnDirection != 0 {
		t.Fatalf(
			"current=%s index=%d direction=%d, want a/0/0",
			payload.CurrentPlayerID,
			game.turnIndex,
			game.turnDirection,
		)
	}
}
