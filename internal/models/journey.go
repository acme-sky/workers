package models

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"
)

// Journey model
type Journey struct {
	Id        uint             `gorm:"column:id" json:"id"`
	CreatedAt time.Time        `gorm:"column:created_at" json:"created_at"`
	Flight1Id int              `json:"-"`
	Flight1   AvailableFlight  `gorm:"foreignKey:Flight1Id;null" json:"flight1"`
	Flight2Id *int             `json:"-"`
	Flight2   *AvailableFlight `gorm:"foreignKey:Flight2Id;null" json:"flight2"`
	Cost      float64          `gorm:"column:cost" json:"cost"`
	UserId    int              `json:"-"`
	User      User             `gorm:"foreignKey:UserId" json:"user"`
}

// Struct used to get new data for a flight
type JourneyInput struct {
	Flight1Id int     `json:"flight1_id" binding:"required"`
	Flight2Id *int    `json:"flight2_id"`
	Cost      float64 `json:"cost" binding:"required"`
	UserId    int     `json:"user_id" binding:"required"`
}

// It validates data from `in` and returns a possible error or not
func ValidateJourney(db *gorm.DB, variables map[string]interface{}) (*JourneyInput, error) {
	var in *JourneyInput

	jsonData, err := json.Marshal(variables)

	if err != nil {
		return nil, errors.New(fmt.Sprintf("Error reading variables `%s`", err.Error()))
	}

	if err := json.Unmarshal(jsonData, &in); err != nil {
		return nil, errors.New(fmt.Sprintf("Error converting json to input `%s`", err.Error()))
	}

	var user User
	var flight1 AvailableFlight
	var flight2 AvailableFlight

	if err := db.Where("id = ?", in.UserId).First(&user).Error; err != nil {
		return nil, errors.New("`user_id` does not exist.")
	}

	if err := db.Where("id = ?", in.Flight1Id).First(&flight1).Error; err != nil {
		return nil, errors.New("`flight1_id` does not exist.")
	}

	if in.Flight2Id != nil {
		if err := db.Where("id = ?", in.Flight2Id).First(&flight2).Error; err != nil {
			return nil, errors.New("`flight2_id` does not exist.")
		}

		if in.Flight1Id == *in.Flight2Id {
			return nil, errors.New("`flight1_id` can't be equals to `flight2_id`")
		}

		if flight1.UserId != flight2.UserId {
			return nil, errors.New("`flight1_id` must have the same user of `flight2_id`")
		}
	}

	if flight1.UserId != in.UserId {
		return nil, errors.New("`flight1_id` must be the same user of `user_id`")
	}

	return in, nil
}

// Returns a new Journey with the data from `in`. It should be called after
// `ValidateJourney(..., in)` method
func NewJourney(in JourneyInput) Journey {
	return Journey{
		CreatedAt: time.Now(),
		Flight1Id: in.Flight1Id,
		Flight2Id: in.Flight2Id,
		Cost:      in.Cost,
		UserId:    in.UserId,
	}
}
