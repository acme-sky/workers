package models

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"
)

// Interest model
type Interest struct {
	Id                uint      `gorm:"column:id" json:"id"`
	CreatedAt         time.Time `gorm:"column:created_at" json:"crated_at"`
	DepartaureTime    time.Time `gorm:"column:departaure_time" json:"departaure_time"`
	DepartaureAirport string    `gorm:"column:departaure_airport" json:"departaure_airport"`
	ArrivalTime       time.Time `gorm:"column:arrival_time" json:"arrival_time"`
	ArrivalAirport    string    `gorm:"column:arrival_airport" json:"arrival_airport"`
	UserId            int       `json:"-"`
	User              User      `gorm:"foreignKey:UserId" json:"user"`
}

// Struct used to get new data for a flight
type InterestInput struct {
	DepartaureTime    time.Time `json:"departaure_time" binding:"required"`
	DepartaureAirport string    `json:"departaure_airport" binding:"required"`
	ArrivalTime       time.Time `json:"arrival_time" binding:"required"`
	ArrivalAirport    string    `json:"arrival_airport" binding:"required"`
	UserId            int       `json:"user_id" binding:"required"`
}

// It validates data from `in` and returns a possible error or not
func ValidateInterest(db *gorm.DB, variables map[string]interface{}) (*InterestInput, error) {
	var in *InterestInput

	jsonData, err := json.Marshal(variables)

	if err != nil {
		return nil, errors.New(fmt.Sprintf("Error reading variables `%s`", err.Error()))
	}

	if err := json.Unmarshal(jsonData, &in); err != nil {
		return nil, errors.New(fmt.Sprintf("Error converting json to input `%s`", err.Error()))
	}

	var user User

	if err := db.Where("id = ?", in.UserId).First(&user).Error; err != nil {
		return nil, errors.New("`user_id` does not exist.")
	}

	if in.DepartaureAirport == in.ArrivalAirport {
		return nil, errors.New("`departaure_airport` can't be equals to `arrival_airport`")
	}

	if in.DepartaureTime.Equal(in.ArrivalTime) || in.DepartaureTime.After(in.ArrivalTime) {
		return nil, errors.New("`departaure_time` can't be after or the same `arrival_time`")
	}

	return in, nil
}

// Returns a new Interest with the data from `in`. It should be called after
// `ValidateInterest(..., in)` method
func NewInterest(in InterestInput) Interest {
	return Interest{
		CreatedAt:         time.Now(),
		DepartaureTime:    in.DepartaureTime,
		DepartaureAirport: in.DepartaureAirport,
		ArrivalTime:       in.ArrivalTime,
		ArrivalAirport:    in.ArrivalAirport,
		UserId:            in.UserId,
	}
}
