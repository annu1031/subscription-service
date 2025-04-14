package model

import (
	"database/sql"
	"time"
)

type RazorpayPlan struct {
	ID              string    `json:"id" db:"id"`
	PlanID          string    `json:"planId" db:"plan_id"`
	RazorpayPlanID  string    `json:"razorpayPlanId" db:"razorpay_plan_id"`
	CreatedAt       time.Time `json:"createdAt" db:"created_at"`
	UpdatedAt       time.Time `json:"updatedAt" db:"updated_at"`
}

type SubscriptionProduct struct {
	ID        string    `json:"id" db:"id"`
	Name      string    `json:"name" db:"name"`
	CreatedAt time.Time `json:"createdAt" db:"created_at"`
	UpdatedAt time.Time `json:"updatedAt" db:"updated_at"`
}

type SubscriptionProductAttribute struct {
	ID        string    `json:"id" db:"id"`
	ProductID string    `json:"productId" db:"product_id"`
	Name      string    `json:"name" db:"name"`
	Value     string    `json:"value" db:"value"`
	CreatedAt time.Time `json:"createdAt" db:"created_at"`
	UpdatedAt time.Time `json:"updatedAt" db:"updated_at"`
}

type SubscriptionPlan struct {
	ID           string    `json:"id" db:"id"`
	ProductID    string    `json:"productId" db:"product_id"`
	Name         string    `json:"name" db:"name"`
	PriceMonthly float64   `json:"priceMonthly" db:"price_monthly"`
	PriceYearly  float64   `json:"priceYearly" db:"price_yearly"`
	Attributes   []SubscriptionProductAttribute `json:"attributes" db:"-"` 
	CreatedAt    time.Time `json:"createdAt" db:"created_at"`
	UpdatedAt    time.Time `json:"updatedAt" db:"updated_at"`
}

type SubscriptionTransaction struct {
    ID                   string         `json:"id" db:"id"`
    UserID               string         `json:"userId" db:"user_id"`
    ProductID            string         `json:"productId" db:"product_id"`
    PlanID               string         `json:"planId" db:"plan_id"`
    CardID               string         `json:"cardId" db:"card_id"`
    IsRenewal            bool           `json:"isRenewal" db:"is_renewal"`
    IsActive             bool           `json:"isActive" db:"is_active"`
    PaymentType          string         `json:"paymentType" db:"payment_type"`
    Amount               float64        `json:"amount" db:"amount"`
    StartDate            time.Time      `json:"startDate" db:"start_date"`
    EndDate              time.Time      `json:"endDate" db:"end_date"`
    NextRenewalDate      time.Time      `json:"nextRenewalDate" db:"next_renewal_date"`
    CreatedAt            time.Time      `json:"createdAt" db:"created_at"`
    UpdatedAt            time.Time      `json:"updatedAt" db:"updated_at"`
    RazorpayPaymentID    sql.NullString `json:"razorpayPaymentId" db:"razorpay_payment_id"`
    RazorpayOrderID      sql.NullString `json:"razorpayOrderId" db:"razorpay_order_id"`
    RazorpaySubscriptionID sql.NullString `json:"razorpaySubscriptionId" db:"razorpay_subscription_id"`
    RazorpayKeyID        string         `json:"razorpayKeyId" db:"-"` 
    AutoRenewal          bool           `json:"autoRenewal" db:"auto_renewal"`
 
    PlanName      string    `json:"planName" db:"-"`
    ProductName   string    `json:"productName" db:"-"`
    CardLastFour  string    `json:"cardLastFour" db:"-"`
}


type SubscriptionPlanWithAttributes struct {
	Plan       SubscriptionPlan                `json:"plan"`
	Attributes []SubscriptionProductAttribute  `json:"attributes"`
}

type SubscriptionRequest struct {
    UserID      string `json:"userId"`
    ProductID   string `json:"productId"`
    PlanID      string `json:"planId"`
    CardID      string `json:"cardId"`
    PaymentType string `json:"paymentType"` 
    AutoRenewal bool   `json:"autoRenewal"` 
}

