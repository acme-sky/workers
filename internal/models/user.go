package models

import "gorm.io/gorm"

// User model
type User struct {
	gorm.Model
	Name               string  `gorm:"column:name"`
	Username           string  `gorm:"column:username" gorm:"uniqueIndex"`
	Email              string  `gorm:"column:email" gorm:"uniqueIndex"`
	Password           string  `gorm:"column:password"`
	Address            *string `gorm:"column:address;null"`
	ProntogramUsername *string `gorm:"column:prontogram_username;null"`
}
