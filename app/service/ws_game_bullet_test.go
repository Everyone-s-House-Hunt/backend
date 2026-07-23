package service

import (
	"reflect"
	"testing"
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
