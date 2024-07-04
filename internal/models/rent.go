package models

import (
	"time"
)

// Rent model
type Rent struct {
	Id        uint      `gorm:"column:id" json:"id"`
	CreatedAt time.Time `gorm:"column:created_at" json:"created_at"`
	Name      string    `gorm:"column:name" json:"name"`
	Latitude  float32   `gorm:"column:latitude" json:"latitude"`
	Longitude float32   `gorm:"column:longitude" json:"longitude"`
	Endpoint  string    `gorm:"column:endpoint" json:"endpoint"`
}
