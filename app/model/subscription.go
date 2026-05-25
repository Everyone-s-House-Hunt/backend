package model

import "time"

type Subscription struct {
	ID        string    `gorm:"primaryKey;type:varchar(36)" json:"id"`
	UserID    string    `gorm:"type:varchar(36);not null" json:"user_id"`
	StartedAt time.Time `gorm:"not null" json:"started_at"`
	ExpiresAt time.Time `gorm:"not null" json:"expires_at"`
	User      User      `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE;" json:"-"`
}