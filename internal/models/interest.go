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
	Id        uint      `gorm:"column:id" json:"id"`
	CreatedAt time.Time `gorm:"column:created_at" json:"created_at"`

	Flight1DepartaureTime    time.Time `gorm:"column:flight1_departaure_time" json:"flight1_departaure_time"`
	Flight1DepartaureAirport string    `gorm:"column:flight1_departaure_airport" json:"flight1_departaure_airport"`
	Flight1ArrivalTime       time.Time `gorm:"column:flight1_arrival_time" json:"flight1_arrival_time"`
	Flight1ArrivalAirport    string    `gorm:"column:flight1_arrival_airport" json:"flight1_arrival_airport"`

	Flight2DepartaureTime    *time.Time `gorm:"column:flight2_departaure_time;null" json:"flight2_departaure_time"`
	Flight2DepartaureAirport *string    `gorm:"column:flight2_departaure_airport;null" json:"flight2_departaure_airport"`
	Flight2ArrivalTime       *time.Time `gorm:"column:flight2_arrival_time;null" json:"flight2_arrival_time"`
	Flight2ArrivalAirport    *string    `gorm:"column:flight2_arrival_airport;null" json:"flight2_arrival_airport"`

	UserId int  `json:"-"`
	User   User `gorm:"foreignKey:UserId" json:"user"`
}

// Struct used to get new data for a flight
type InterestInput struct {
	Flight1DepartaureTime    time.Time  `json:"flight1_departaure_time" binding:"required"`
	Flight1DepartaureAirport string     `json:"flight1_departaure_airport" binding:"required"`
	Flight1ArrivalTime       time.Time  `json:"flight1_arrival_time" binding:"required"`
	Flight1ArrivalAirport    string     `json:"flight1_arrival_airport" binding:"required"`
	Flight2DepartaureTime    *time.Time `json:"flight2_departaure_time"`
	Flight2DepartaureAirport *string    `json:"flight2_departaure_airport"`
	Flight2ArrivalTime       *time.Time `json:"flight2_arrival_time"`
	Flight2ArrivalAirport    *string    `json:"flight2_arrival_airport"`
	UserId                   int        `json:"user_id" binding:"required"`
}

// It validates data from `in` and returns a possible error or not
func ValidateInterest(db *gorm.DB, variables map[string]interface{}) (*InterestInput, error) {
	var in *InterestInput

	for _, i := range []string{"flight1_departaure_time", "flight1_arrival_time", "flight2_departaure_time", "flight2_arrival_time"} {
		if variables[i] != nil && len(variables[i].(string)) == 10 {
			variables[i] = fmt.Sprintf("%sT00:00:00Z", variables[i])
		}
	}

	if variables["user_id"] == nil {
		variables["user_id"] = 1
	}

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

	if in.Flight1DepartaureAirport == in.Flight1ArrivalAirport {
		return nil, errors.New("`flight1`: `departaure_airport` can't be equals to `arrival_airport`")
	}

	if in.Flight1DepartaureTime.Equal(in.Flight1ArrivalTime) || in.Flight1DepartaureTime.After(in.Flight1ArrivalTime) {
		return nil, errors.New("`flight1`: `departaure_time` can't be after or the same `arrival_time`")
	}

	if in.Flight2DepartaureAirport != nil && in.Flight2DepartaureTime != nil && in.Flight2ArrivalAirport != nil && in.Flight2ArrivalTime != nil {
		if (*in.Flight2DepartaureAirport) == (*in.Flight2ArrivalAirport) {
			return nil, errors.New("`flight2`: `departaure_airport` can't be equals to `arrival_airport`")
		}

		if (*in.Flight2DepartaureTime).Equal(*in.Flight2ArrivalTime) || (*in.Flight2DepartaureTime).After(*in.Flight2ArrivalTime) {
			return nil, errors.New("`flight2`: `departaure_time` can't be after or the same `arrival_time`")
		}
	} else if !(in.Flight2DepartaureAirport == nil || in.Flight2DepartaureTime == nil || in.Flight2ArrivalAirport == nil || in.Flight2ArrivalTime == nil) {
		return nil, errors.New("`flight2`: all fields must be nil or filled")
	}

	return in, nil
}

// Returns a new Interest with the data from `in`. It should be called after
// `ValidateInterest(..., in)` method
func NewInterest(in InterestInput) Interest {
	return Interest{
		CreatedAt:                time.Now(),
		Flight1DepartaureTime:    in.Flight1DepartaureTime,
		Flight1DepartaureAirport: in.Flight1DepartaureAirport,
		Flight1ArrivalTime:       in.Flight1ArrivalTime,
		Flight1ArrivalAirport:    in.Flight1ArrivalAirport,
		Flight2DepartaureTime:    in.Flight2DepartaureTime,
		Flight2DepartaureAirport: in.Flight2DepartaureAirport,
		Flight2ArrivalTime:       in.Flight2ArrivalTime,
		Flight2ArrivalAirport:    in.Flight2ArrivalAirport,
		UserId:                   in.UserId,
	}
}
