package models

import (
	"fmt"
	"math/rand"
	"os"
	"strconv"
	"time"

	"gorm.io/gorm"
)

// Offer model
type Offer struct {
	Id        uint      `gorm:"column:id" json:"id"`
	CreatedAt time.Time `gorm:"column:created_at" json:"created_at"`
	Message   string    `gorm:"column:message" json:"message"`
	Expired   string    `gorm:"column:expired" json:"expired"`
	Token     string    `gorm:"column:token" json:"token"`
	UserId    int       `json:"-"`
	User      User      `gorm:"foreignKey:UserId" json:"user"`
}

type OfferInputFields struct {
	Name              string `binding:"required"`
	DepartaureAirport string `binding:"required"`
	ArrivalAirport    string `binding:"required"`
	DepartaureTime    string `binding:"required"`
	ArrivalTime       string `binding:"required"`
}

// Struct used to get new data for an offer
type OfferInput struct {
	Fields OfferInputFields `json:"fields" binding:"required"`
	UserId int              `json:"user_id" binding:"required"`
}

// It validates data from `in` and returns a possible error or not
func ValidateOffer(db *gorm.DB, variables map[string]interface{}) (*OfferInput, error) {
	var in *OfferInput

	return in, nil
}

func randSeq(n int) string {
	var letters = []rune("ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")
	b := make([]rune, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

// Returns a new Offer with the data from `in`. It should be called after
// `ValidateOffer(..., in)` method
func NewOffer(in OfferInput) Offer {
	offerValidationTime, _ := strconv.Atoi(os.Getenv("OFFER_VALIDATION_TIME"))

	token := randSeq(6)

	message := fmt.Sprintf(
		"Hello %s, this is the offer token for your flight from <b>%s</b> to <b>%s</b> in date %s - %s.<br><a href=\"#\" target=\"_blank\">%s</a>",
		in.Fields.Name,
		in.Fields.DepartaureAirport,
		in.Fields.ArrivalAirport,
		in.Fields.DepartaureTime,
		in.Fields.ArrivalTime,
		token,
	)
	return Offer{
		CreatedAt: time.Now(),
		Message:   message,
		Expired:   time.Now().Add(time.Hour * time.Duration(offerValidationTime)).Format("20060102150405"),
		Token:     token,
		UserId:    in.UserId,
	}
}
