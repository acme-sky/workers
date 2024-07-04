package models

import (
	"time"
)

// Airline model
type Airline struct {
	Id            uint      `gorm:"column:id" json:"id"`
	CreatedAt     time.Time `gorm:"column:created_at" json:"created_at"`
	Name          string    `gorm:"column:name" json:"name"`
	LoginUsername string    `gorm:"column:login_username" json:"login_username"`
	LoginPassword string    `gorm:"column:login_password" json:"login_password"`
	Endpoint      string    `gorm:"column:endpoint" json:"endpoint"`
}
