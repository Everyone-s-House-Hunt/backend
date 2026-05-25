package model

import "time"

type Question struct {
	ID            string    `gorm:"primaryKey;type:varchar(36)" json:"id"`
	CreatorUserID string    `gorm:"type:varchar(36);not null" json:"creator_user_id"`
	Body          string    `gorm:"type:text;not null" json:"body"`
	AnswerData    string    `gorm:"type:json;not null" json:"answer_data"`
	GameMode      string    `gorm:"type:varchar(30);not null" json:"game_mode"`
	Status        string    `gorm:"type:varchar(20);not null;default:'pending'" json:"status"`
	CreatedAt     time.Time `gorm:"not null;default:CURRENT_TIMESTAMP(3)" json:"created_at"`
	CreatorUser   User      `gorm:"foreignKey:CreatorUserID;constraint:OnDelete:CASCADE;" json:"-"`
}