package model

import (
	"time"
)

type Card struct {
	ID             string    `json:"id" db:"id"`
	UserID         string    `json:"userId" db:"user_id"`
	CardNumber     string    `json:"cardNumber" db:"card_number"`
	CardHolderName string    `json:"cardHolderName" db:"card_holder_name"`
	ExpiryMonth    int       `json:"expiryMonth" db:"expiry_month"`
	ExpiryYear     int       `json:"expiryYear" db:"expiry_year"`
	CardType       string    `json:"cardType" db:"card_type"`
	LastFourDigits string    `json:"lastFourDigits" db:"last_four_digits"`
	IsDefault      bool      `json:"isDefault" db:"is_default"`
	CreatedAt      time.Time `json:"createdAt" db:"created_at"`
	UpdatedAt      time.Time `json:"updatedAt" db:"updated_at"`
}

type Subscription struct {
	ID          string    `json:"id" db:"id"`
	UserID      string    `json:"userId" db:"user_id"`
	PlanID      string    `json:"planId" db:"plan_id"`
	Status      string    `json:"status" db:"status"`
	StartDate   time.Time `json:"startDate" db:"start_date"`
	EndDate     time.Time `json:"endDate" db:"end_date"`
	RenewalDate time.Time `json:"renewalDate" db:"renewal_date"`
	AutoRenewal bool      `json:"autoRenewal" db:"auto_renewal"`
	CreatedAt   time.Time `json:"createdAt" db:"created_at"`
	UpdatedAt   time.Time `json:"updatedAt" db:"updated_at"`
}

type Plan struct {
	ID             string    `json:"id" db:"id"`
	Name           string    `json:"name" db:"name"`
	Description    string    `json:"description" db:"description"`
	Price          float64   `json:"price" db:"price"`
	DurationMonths int       `json:"durationMonths" db:"duration_months"`
	CreatedAt      time.Time `json:"createdAt" db:"created_at"`
	UpdatedAt      time.Time `json:"updatedAt" db:"updated_at"`
}

type Payment struct {
	ID               string    `json:"id" db:"id"`
	SubscriptionID   string    `json:"subscriptionId" db:"subscription_id"`
	PaymentMethod    string    `json:"paymentMethod" db:"payment_method"`
	CardID           string    `json:"cardId" db:"card_id"`
	Amount           float64   `json:"amount" db:"amount"`
	Currency         string    `json:"currency" db:"currency"`
	Status           string    `json:"status" db:"status"`
	RazorpayPaymentID string   `json:"razorpayPaymentId" db:"razorpay_payment_id"`
	RazorpayOrderID   string   `json:"razorpayOrderId" db:"razorpay_order_id"`
	TransactionDate  time.Time `json:"transactionDate" db:"transaction_date"`
	CreatedAt        time.Time `json:"createdAt" db:"created_at"`
	UpdatedAt        time.Time `json:"updatedAt" db:"updated_at"`
}