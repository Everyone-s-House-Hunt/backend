package model

import "time"

type User struct {
	ID           string    `gorm:"primaryKey;type:varchar(36)" json:"id"`
	Username     string    `gorm:"type:varchar(50);not null;unique" json:"username"`
	Email        string    `gorm:"type:varchar(255);not null;unique" json:"email"`
	PasswordHash string    `gorm:"type:varchar(255);not null" json:"-"`
	IsPremium    bool      `gorm:"not null;default:false" json:"is_premium"`
	CreatedAt    time.Time `gorm:"not null;default:CURRENT_TIMESTAMP" json:"created_at"`
}