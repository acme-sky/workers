package models

import (
	"time"
)

// Invoice model
type Invoice struct {
	Id                uint      `gorm:"column:id" json:"id"`
	CreatedAt         time.Time `gorm:"column:created_at" json:"created_at"`
	RentId            string    `gorm:"column:rent_id" json:"rent_id"`
	RentCustomerName  string    `gorm:"column:rent_customer_name" json:"rent_customer_name"`
	RentPickupAddress string    `gorm:"column:rent_pickup_address" json:"rent_pickup_address"`
	RentPickupDate    string    `gorm:"column:rent_pickup_date" json:"rent_pickup_date"`
	RentAddress       string    `gorm:"column:rent_address" json:"rent_address"`
	JourneyId         int       `json:"-"`
	Journey           Journey   `gorm:"foreignKey:JourneyId" json:"journey"`
	UserId            int       `json:"-"`
	User              User      `gorm:"foreignKey:UserId" json:"user"`
}

// Struct used to get new data for an invoice
type InvoiceInput struct {
	RentId            string `json:"rent_id"`
	RentCustomerName  string `json:"rent_customer_name"`
	RentPickupAddress string `json:"rent_pickup_address"`
	RentPickupDate    string `json:"rent_pickup_date"`
	RentAddress       string `json:"rent_address"`
	JourneyId         int    `json:"journey_id" binding:"required"`
	UserId            int    `json:"user_id" binding:"required"`
}

func NewInvoice(in InvoiceInput) Invoice {
	return Invoice{
		CreatedAt:         time.Now(),
		RentId:            in.RentId,
		RentCustomerName:  in.RentCustomerName,
		RentPickupAddress: in.RentPickupAddress,
		RentPickupDate:    in.RentPickupDate,
		RentAddress:       in.RentAddress,
		JourneyId:         in.JourneyId,
		UserId:            in.UserId,
	}
}
