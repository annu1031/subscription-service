package model

import (
	"time"
)

type UserInfo struct {
	ID                 string    `json:"id" db:"id"`
	Name               string    `json:"name" db:"name"`
	Email              string    `json:"email" db:"email"`
	Phone              string    `json:"phone" db:"phone"`
	RazorpayCustomerID string    `json:"razorpayCustomerId" db:"razorpay_customer_id"`
	CreatedAt          time.Time `json:"createdAt" db:"created_at"`
	UpdatedAt          time.Time `json:"updatedAt" db:"updated_at"`
}