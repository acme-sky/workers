package models

import (
	"time"
)

// Rent model
type Rent struct {
	Id        uint      `gorm:"column:id" json:"id"`
	CreatedAt time.Time `gorm:"column:created_at" json:"created_at"`
	Name      string    `gorm:"name" json:"name"`
	Latitude  float32   `gorm:"latitude" json:"latitude"`
	Longitude float32   `gorm:"longitude" json:"longitude"`
	Endpoint  string    `gorm:"endpoint" json:"endpoint"`
}
