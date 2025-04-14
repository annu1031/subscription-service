package razorpay

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"log"

	"github.com/razorpay/razorpay-go"
)

var (
	ErrPaymentCreationFailed = errors.New("failed to create payment")
	ErrSubscriptionCreationFailed = errors.New("failed to create subscription")
	ErrCustomerCreationFailed = errors.New("failed to create customer")
	ErrPlanCreationFailed = errors.New("failed to create plan")
)


type Client struct {
	client *razorpay.Client
	keyID string
	keySecret string
}

type Config struct {
	KeyID     string
	KeySecret string
}

func NewClient(config Config) *Client {
	
	if config.KeyID == "" {
		log.Println("WARNING: Razorpay Key ID is empty!")
	} else {
		log.Printf("Initializing Razorpay with Key ID: %s (length: %d)", 
			config.KeyID, len(config.KeyID))
	}
	
	if config.KeySecret == "" {
		log.Println("WARNING: Razorpay Key Secret is empty!")
	} else {
		log.Printf("Razorpay Key Secret length: %d", len(config.KeySecret))
	}
	
	client := razorpay.NewClient(config.KeyID, config.KeySecret)
	
	if config.KeyID != "" && config.KeySecret != "" {
		log.Println("Testing Razorpay connection...")
		_, err := client.Payment.All(map[string]interface{}{
			"count": 1,
		}, nil)
		if err != nil {
			log.Printf("Razorpay connection test failed: %v", err)
		} else {
			log.Println("Razorpay connection successful!")
		}
	}
	
	return &Client{
		client: client,
		keyID: config.KeyID,
		keySecret: config.KeySecret,
	}
}


func (c *Client) CreateOrder(ctx context.Context, amount int, currency string, receiptID string) (map[string]interface{}, error) {
	log.Printf("Creating Razorpay order: Amount %d %s, Receipt ID: %s", 
		amount, currency, receiptID)
	
	data := map[string]interface{}{
		"amount":   amount, // amount in smallest currency unit (paise)
		"currency": currency,
		"receipt":  receiptID,
	}

	order, err := c.client.Order.Create(data, nil)
	if err != nil {
		log.Printf("Failed to create Razorpay order: %v", err)
		return nil, fmt.Errorf("%w: %v", ErrPaymentCreationFailed, err)
	}

	log.Printf("Successfully created Razorpay order: ID %s", order["id"])
	return order, nil
}

func (c *Client) CreateCustomer(ctx context.Context, name, email, contact string) (map[string]interface{}, error) {
	log.Printf("Creating Razorpay customer: %s (%s)", name, email)
	
	data := map[string]interface{}{
		"name":    name,
		"email":   email,
		"contact": contact,
	}

	customer, err := c.client.Customer.Create(data, nil)
	if err != nil {
		log.Printf("Failed to create Razorpay customer: %v", err)
		return nil, fmt.Errorf("%w: %v", ErrCustomerCreationFailed, err)
	}

	log.Printf("Successfully created Razorpay customer: ID %s", customer["id"])
	return customer, nil
}

func (c *Client) GetOrCreateCustomer(ctx context.Context, customerID, name, email, contact string) (map[string]interface{}, error) {
	if customerID != "" {
		options := map[string]interface{}{}
		customer, err := c.client.Customer.Fetch(customerID, options, nil)
		if err == nil {
			log.Printf("Found existing Razorpay customer: ID %s", customer["id"])
			return customer, nil
		}
		log.Printf("Customer not found, creating new: %v", err)
	}
	
	return c.CreateCustomer(ctx, name, email, contact)
}

func (c *Client) CreatePlan(ctx context.Context, planName string, amount int, interval string) (map[string]interface{}, error) {
	log.Printf("Creating Razorpay plan: %s, %d per %s", planName, amount, interval)
	
	data := map[string]interface{}{
		"period": interval,
		"interval": 1,
		"item": map[string]interface{}{
			"name": planName,
			"amount": amount,
			"currency": "INR",
			"description": planName + " subscription plan",
		},
	}

	plan, err := c.client.Plan.Create(data, nil)
	if err != nil {
		log.Printf("Failed to create Razorpay plan: %v", err)
		return nil, fmt.Errorf("%w: %v", ErrPlanCreationFailed, err)
	}

	log.Printf("Successfully created Razorpay plan: ID %s", plan["id"])
	return plan, nil
}

func (c *Client) CreateSubscription(ctx context.Context, planID string, customerID string, totalCount int, customerNotify bool) (map[string]interface{}, error) {
	log.Printf("Creating Razorpay subscription: Plan ID %s, Customer ID %s", planID, customerID)
	
	var notifyValue int
	if customerNotify {
		notifyValue = 1
	} else {
		notifyValue = 0
	}
	
	data := map[string]interface{}{
		"plan_id":         planID,
		"customer_id":     customerID,
		"total_count":     totalCount,
		"customer_notify": notifyValue,
	}

	subscription, err := c.client.Subscription.Create(data, nil)
	if err != nil {
		log.Printf("Failed to create Razorpay subscription: %v", err)
		return nil, fmt.Errorf("%w: %v", ErrSubscriptionCreationFailed, err)
	}

	log.Printf("Successfully created Razorpay subscription: ID %s", subscription["id"])
	return subscription, nil
}


func (c *Client) CancelSubscription(ctx context.Context, subscriptionID string, cancelAtCycleEnd bool) (map[string]interface{}, error) {
	log.Printf("Cancelling Razorpay subscription: ID %s, At cycle end: %v", 
		subscriptionID, cancelAtCycleEnd)
	
	data := map[string]interface{}{
		"cancel_at_cycle_end": cancelAtCycleEnd,
	}

	subscription, err := c.client.Subscription.Cancel(subscriptionID, data, nil)
	if err != nil {
		log.Printf("Failed to cancel Razorpay subscription: %v", err)
		return nil, fmt.Errorf("failed to cancel subscription: %v", err)
	}

	log.Printf("Successfully cancelled Razorpay subscription: ID %s", subscription["id"])
	return subscription, nil
}

func (c *Client) VerifyPaymentSignature(attributes map[string]interface{}, signature string) bool {

	if c.keySecret == "" {
		log.Println("Warning: Webhook signature verification skipped (no key secret)")
		return true
	}
	
	payload, ok := attributes["payload"].(string)
	if !ok {
		log.Println("Warning: Invalid payload format for signature verification")
		return false
	}
	
	mac := hmac.New(sha256.New, []byte(c.keySecret))
	mac.Write([]byte(payload))
	expectedSignature := hex.EncodeToString(mac.Sum(nil))
	
	result := expectedSignature == signature
	if !result {
		log.Println("Warning: Webhook signature verification failed")
	} else {
		log.Println("Webhook signature verified successfully")
	}
	
	return result
}

func (c *Client) TestConnection() error {
	log.Println("Testing Razorpay connection...")
	_, err := c.client.Payment.All(map[string]interface{}{
		"count": 1,
	}, nil)
	if err != nil {
		log.Printf("Razorpay connection test failed: %v", err)
		return err
	}
	log.Println("Razorpay connection successful!")
	return nil
}