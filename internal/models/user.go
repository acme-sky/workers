package models

import "gorm.io/gorm"

// User model
type User struct {
	gorm.Model
	Username string  `gorm:"column:username" gorm:"uniqueIndex"`
	Password string  `gorm:"column:password"`
	Address  *string `gorm:"colum:address;null"`
}
