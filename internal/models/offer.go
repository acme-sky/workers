package models

import (
	"fmt"
	"math/rand"
	"strconv"
	"time"

	"github.com/acme-sky/workers/internal/config"
	"github.com/charmbracelet/log"
	"gorm.io/gorm"
)

// Offer model
type Offer struct {
	Id           uint      `gorm:"column:id" json:"id"`
	CreatedAt    time.Time `gorm:"column:created_at" json:"created_at"`
	Message      string    `gorm:"column:message" json:"message"`
	Expired      string    `gorm:"column:expired" json:"expired"`
	Token        string    `gorm:"column:token" json:"token"`
	IsUsed       bool      `gorm:"column:is_used" json:"is_used"`
	PaymentLink  string    `gorm:"column:payment_link" json:"payment_link"`
	PaymentPaid  bool      `gorm:"column:payment_paid" json:"payment_paid"`
	RentEndpoint string    `gorm:"column:rent_endpoint" json:"rent_endpoint"`
	RentId       string    `gorm:"column:rent_id" json:"rent_id"`
	JourneyId    int       `json:"-"`
	Journey      Journey   `gorm:"foreignKey:JourneyId" json:"journey"`
	UserId       int       `json:"-"`
	User         User      `gorm:"foreignKey:UserId" json:"user"`
}

type OfferInputFields struct {
	DepartureAirport string  `binding:"required"`
	ArrivalAirport   string  `binding:"required"`
	DepartureTime    string  `binding:"required"`
	ArrivalTime      string  `binding:"required"`
	Cost             float64 `binding:"required"`
}

// Struct used to get new data for an offer
type OfferInput struct {
	Name      string            `json:"name"`
	Flight1   OfferInputFields  `json:"flight1" binding:"required"`
	Flight2   *OfferInputFields `json:"flight2"`
	JourneyId int               `json:"journey_id" binding:"required"`
	UserId    int               `json:"user_id" binding:"required"`
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
	var offerValidationTime int

	conf, err := config.GetConfig()
	if err != nil {
		log.Warnf("Can't load config for OFFER_VALIDATION_TIME, so use '24' by default %s", err.Error())
		offerValidationTime = 24
	} else {
		offerValidationTime = conf.Int("offer.validation.time")
	}

	token := randSeq(6)

	message := fmt.Sprintf(
		"Hello %s, this is the offer token for your flight from <b>%s</b> to <b>%s</b> in date %s - %s for %.2f€.",
		in.Name,
		in.Flight1.DepartureAirport,
		in.Flight1.ArrivalAirport,
		in.Flight1.DepartureTime,
		in.Flight1.ArrivalTime,
		in.Flight1.Cost,
	)

	cost := in.Flight1.Cost

	if in.Flight2 != nil {
		message = fmt.Sprintf("%s <br>You also have a return flight  from <b>%s</b> to <b>%s</b> in date %s - %s for %.2f€.",
			message,
			in.Flight2.DepartureAirport,
			in.Flight2.ArrivalAirport,
			in.Flight2.DepartureTime,
			in.Flight2.ArrivalTime,
			in.Flight2.Cost,
		)

		cost += in.Flight2.Cost
	}

	message = fmt.Sprintf("%s <br>The total for your journey is %.2f€. <br><a href=\"#\" target=\"_blank\">%s</a>",
		message,
		cost,
		token,
	)

	return Offer{
		CreatedAt:    time.Now(),
		Message:      message,
		Expired:      strconv.FormatInt(time.Now().Add(time.Hour*time.Duration(offerValidationTime)).Unix(), 10),
		Token:        token,
		IsUsed:       false,
		PaymentLink:  "",
		PaymentPaid:  false,
		RentEndpoint: "",
		RentId:       "",
		JourneyId:    in.JourneyId,
		UserId:       in.UserId,
	}
}
