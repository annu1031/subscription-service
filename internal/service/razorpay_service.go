package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"time"

	"subscription-management/internal/model"
	"subscription-management/internal/razorpay"
	"subscription-management/internal/repository"
)

var (
	ErrRazorpayOperationFailed = errors.New("razorpay operation failed")
	ErrInvalidWebhookSignature = errors.New("invalid webhook signature")
)

type RazorpayService interface {
	CreatePayment(ctx context.Context, amount float64, currency string, receiptID string) (map[string]interface{}, error)
	CreateSubscription(ctx context.Context, subscription *model.SubscriptionTransaction, userInfo *model.UserInfo) (map[string]interface{}, error)
	CancelSubscription(ctx context.Context, razorpaySubscriptionID string) error
	HandleWebhook(ctx context.Context, payload []byte, signature string) error
	TestConnection(ctx context.Context) (interface{}, error)
	GetPlanInfo(ctx context.Context, planID string, paymentType string) (map[string]interface{}, error)
}

type PlanInfo struct {
	Monthly string
	Yearly  string
}

type DefaultRazorpayService struct {
	razorpayClient     *razorpay.Client
	subscriptionRepo   repository.SubscriptionRepository
	webhookSecret      string
	planMapping        map[string]PlanInfo
}

func NewRazorpayService(
	razorpayClient *razorpay.Client,
	subscriptionRepo repository.SubscriptionRepository,
	webhookSecret string,
) RazorpayService {
	planMapping := map[string]PlanInfo{
		"plan-001": { // Budget
			Monthly: "plan_QIbEUICtejuBUQ", 
			Yearly:  "plan_QIbFBU9kxYoEhg",
		},
		"plan-002": { // Standard
			Monthly: "plan_LgWAqFqsESLnhb",
			Yearly:  "plan_LgWAtVDXJI4Nzu",
		},
		"plan-003": { // Premium
			Monthly: "plan_LgWB3F9fPBzPRV",
			Yearly:  "plan_LgWB7dBSP7iUf7",
		},
	}
	
	return &DefaultRazorpayService{
		razorpayClient:   razorpayClient,
		subscriptionRepo: subscriptionRepo,
		webhookSecret:    webhookSecret,
		planMapping:      planMapping,
	}
}

func (s *DefaultRazorpayService) TestConnection(ctx context.Context) (interface{}, error) {
	log.Println("Testing Razorpay connection")
	if s.razorpayClient == nil {
		return nil, errors.New("razorpay client is not initialized")
	}
	
	if err := s.razorpayClient.TestConnection(); err != nil {
		return nil, err
	}

	return "Razorpay connection successful", nil
}

func (s *DefaultRazorpayService) GetPlanInfo(ctx context.Context, planID string, paymentType string) (map[string]interface{}, error) {
	planInfo, exists := s.planMapping[planID]
	if !exists {
		log.Printf("No Razorpay plan mapping found for plan ID: %s", planID)
		return nil, fmt.Errorf("unknown plan ID: %s", planID)
	}

	var razorpayPlanID string
	if paymentType == "monthly" {
		razorpayPlanID = planInfo.Monthly
	} else { 
		razorpayPlanID = planInfo.Yearly
	}
	
	log.Printf("Using Razorpay plan ID: %s for local plan %s (%s)", 
		razorpayPlanID, planID, paymentType)
	
	return map[string]interface{}{
		"razorpay_plan_id": razorpayPlanID,
	}, nil
}

func (s *DefaultRazorpayService) CreatePayment(
	ctx context.Context, 
	amount float64, 
	currency string, 
	receiptID string,
) (map[string]interface{}, error) {
	amountInPaise := int(amount * 100)
	
	log.Printf("Creating Razorpay payment: Amount %.2f %s (%d paise), Receipt ID: %s", 
		amount, currency, amountInPaise, receiptID)
	
	order, err := s.razorpayClient.CreateOrder(ctx, amountInPaise, currency, receiptID)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrRazorpayOperationFailed, err)
	}
	
	return order, nil
}

func (s *DefaultRazorpayService) CreateSubscription(
	ctx context.Context, 
	subscription *model.SubscriptionTransaction,
	userInfo *model.UserInfo,
) (map[string]interface{}, error) {
	log.Printf("Creating Razorpay subscription for user %s, plan %s (%s)", 
		subscription.UserID, subscription.PlanID, subscription.PaymentType)

	if userInfo == nil {
		log.Println("Warning: No user info provided for subscription")
		userInfo = &model.UserInfo{
			ID: subscription.UserID,
			Name: "User " + subscription.UserID,
			Email: subscription.UserID + "@example.com",
			Phone: "",
		}
	}
	
	customer, err := s.razorpayClient.GetOrCreateCustomer(
		ctx,
		userInfo.RazorpayCustomerID, 
		userInfo.Name,
		userInfo.Email,
		userInfo.Phone,
	)
	if err != nil {
		log.Printf("Failed to create/get Razorpay customer: %v", err)
		return nil, fmt.Errorf("failed to create/get customer: %v", err)
	}
	
	customerID, _ := customer["id"].(string)
	log.Printf("Using Razorpay customer ID: %s", customerID)

	planInfo, err := s.GetPlanInfo(ctx, subscription.PlanID, subscription.PaymentType)
	if err != nil {
		log.Printf("Failed to get Razorpay plan: %v", err)
		return nil, fmt.Errorf("failed to get plan: %v", err)
	}
	
	razorpayPlanID := planInfo["razorpay_plan_id"].(string)
	razorpayPlanID = "plan_QIbDj6gHdpgbDl"
	log.Printf("Using Razorpay plan ID: %s", razorpayPlanID)
	
	
	totalCount := 12 // For monthly billing (12 payments in a year)
	if subscription.PaymentType == "yearly" {
		totalCount = 1 // Only one payment for yearly
	}

	razorpaySubscription, err := s.razorpayClient.CreateSubscription(
		ctx,
		razorpayPlanID,
		customerID,
		totalCount,
		true, 
	)
	
	if err != nil {
		log.Printf("Failed to create Razorpay subscription: %v", err)
		return nil, fmt.Errorf("%w: %v", ErrRazorpayOperationFailed, err)
	}
	
	log.Printf("Successfully created Razorpay subscription: %v", razorpaySubscription["id"])
	return razorpaySubscription, nil
}

func (s *DefaultRazorpayService) CancelSubscription(
	ctx context.Context, 
	razorpaySubscriptionID string,
) error {
	_, err := s.razorpayClient.CancelSubscription(ctx, razorpaySubscriptionID, false)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrRazorpayOperationFailed, err)
	}
	
	return nil
}

func (s *DefaultRazorpayService) HandleWebhook(
	ctx context.Context, 
	payload []byte, 
	signature string,
) error {
	log.Println("Received Razorpay webhook")

	log.Printf("Webhook payload: %s", string(payload))

	testMode := ctx.Value("testMode") != nil

	if !testMode {
		if !s.razorpayClient.VerifyPaymentSignature(map[string]interface{}{
			"payload": string(payload),
		}, signature) {
			log.Println("Webhook signature verification failed")
			return ErrInvalidWebhookSignature
		}
	} else {
		log.Println("TESTING MODE: Skipping webhook signature verification")
	}

	var event map[string]interface{}
	if err := json.Unmarshal(payload, &event); err != nil {
		log.Printf("Failed to parse webhook payload: %v", err)
		return fmt.Errorf("failed to parse webhook payload: %v", err)
	}

	eventType, ok := event["event"].(string)
	if !ok {
		log.Println("Missing event type in webhook payload")
		return fmt.Errorf("missing event type in webhook payload")
	}

	log.Printf("Received Razorpay webhook event: %s", eventType)

	switch eventType {
	case "payment.authorized":
		return s.handlePaymentAuthorized(ctx, event)
	case "subscription.charged":
		return s.handleSubscriptionCharged(ctx, event)
	case "subscription.cancelled":
		return s.handleSubscriptionCancelled(ctx, event)
	case "payment.failed":
		return s.handlePaymentFailed(ctx, event)
	default:
		log.Printf("Unhandled webhook event type: %s", eventType)
		return nil
	}
}

func (s *DefaultRazorpayService) handlePaymentAuthorized(ctx context.Context, event map[string]interface{}) error {
	log.Println("Processing payment.authorized webhook")

	payloadObj, _ := event["payload"].(map[string]interface{})
	if payloadObj == nil {
		return fmt.Errorf("invalid payload in webhook event")
	}
	
	paymentObj, _ := payloadObj["payment"].(map[string]interface{})
	if paymentObj == nil {
		return fmt.Errorf("invalid payment data in webhook payload")
	}

	entity, _ := paymentObj["entity"].(map[string]interface{})
	if entity == nil {
		return fmt.Errorf("invalid entity in payment data")
	}
	
	paymentID, _ := entity["id"].(string)
	orderID, _ := entity["order_id"].(string)
	
	if paymentID == "" || orderID == "" {
		return fmt.Errorf("missing payment or order ID in webhook")
	}
	
	log.Printf("Payment authorized: Payment ID %s, Order ID %s", paymentID, orderID)

	subscription, err := s.subscriptionRepo.GetSubscriptionByRazorpayOrderID(ctx, orderID)
	if err != nil {
		log.Printf("Failed to find subscription with order ID %s: %v", orderID, err)
		return fmt.Errorf("failed to find subscription with order ID %s: %v", orderID, err)
	}
	
	if subscription == nil {
		log.Printf("No subscription found with Razorpay order ID: %s", orderID)
		return fmt.Errorf("no subscription found with Razorpay order ID: %s", orderID)
	}

	subscription.RazorpayPaymentID = toNullString(paymentID)

	if err := s.subscriptionRepo.UpdateSubscription(ctx, subscription); err != nil {
		log.Printf("Failed to update subscription with payment ID: %v", err)
		return fmt.Errorf("failed to update subscription with payment ID: %v", err)
	}
	
	log.Printf("Subscription payment authorized: Order ID %s, Payment ID %s", orderID, paymentID)
	return nil
}

func (s *DefaultRazorpayService) handleSubscriptionCharged(ctx context.Context, event map[string]interface{}) error {
	log.Println("Processing subscription.charged webhook")

	payloadObj, _ := event["payload"].(map[string]interface{})
	if payloadObj == nil {
		return fmt.Errorf("invalid payload in webhook event")
	}
	
	subscriptionObj, _ := payloadObj["subscription"].(map[string]interface{})
	if subscriptionObj == nil {
		return fmt.Errorf("invalid subscription data in webhook payload")
	}

	entity, _ := subscriptionObj["entity"].(map[string]interface{})
	if entity == nil {
		return fmt.Errorf("invalid entity in subscription data")
	}
	
	subscriptionID, _ := entity["id"].(string)
	if subscriptionID == "" {
		return fmt.Errorf("missing subscription ID in webhook")
	}
	
	log.Printf("Subscription charged: Subscription ID %s", subscriptionID)

	subscription, err := s.subscriptionRepo.GetSubscriptionByRazorpaySubscriptionID(ctx, subscriptionID)
	if err != nil {
		log.Printf("Failed to find subscription with ID %s: %v", subscriptionID, err)
		return fmt.Errorf("failed to find subscription with ID %s: %v", subscriptionID, err)
	}
	
	if subscription == nil {
		log.Printf("No subscription found with Razorpay subscription ID: %s", subscriptionID)
		return fmt.Errorf("no subscription found with Razorpay subscription ID: %s", subscriptionID)
	}

	startDate := time.Now()
	var endDate time.Time
	
	if subscription.PaymentType == "monthly" {
		endDate = startDate.AddDate(0, 1, 0)
	} else { // yearly
		endDate = startDate.AddDate(1, 0, 0)
	}

	renewalSubscription := &model.SubscriptionTransaction{
		UserID:                 subscription.UserID,
		ProductID:              subscription.ProductID,
		PlanID:                 subscription.PlanID,
		CardID:                 subscription.CardID,
		IsRenewal:              true,
		IsActive:               true,
		PaymentType:            subscription.PaymentType,
		Amount:                 subscription.Amount,
		StartDate:              startDate,
		EndDate:                endDate,
		NextRenewalDate:        endDate,
		RazorpaySubscriptionID: toNullString(subscriptionID),
	}

	if err := s.subscriptionRepo.CreateSubscription(ctx, renewalSubscription); err != nil {
		log.Printf("Failed to create renewal subscription: %v", err)
		return fmt.Errorf("failed to create renewal subscription: %v", err)
	}
	
	log.Printf("Subscription charged and renewed: Subscription ID %s", subscriptionID)
	return nil
}

func (s *DefaultRazorpayService) handleSubscriptionCancelled(ctx context.Context, event map[string]interface{}) error {
	log.Println("Processing subscription.cancelled webhook")

	payloadObj, _ := event["payload"].(map[string]interface{})
	if payloadObj == nil {
		return fmt.Errorf("invalid payload in webhook event")
	}
	
	subscriptionObj, _ := payloadObj["subscription"].(map[string]interface{})
	if subscriptionObj == nil {
		return fmt.Errorf("invalid subscription data in webhook payload")
	}

	entity, _ := subscriptionObj["entity"].(map[string]interface{})
	if entity == nil {
		return fmt.Errorf("invalid entity in subscription data")
	}
	
	subscriptionID, _ := entity["id"].(string)
	if subscriptionID == "" {
		return fmt.Errorf("missing subscription ID in webhook")
	}
	
	log.Printf("Subscription cancelled: Subscription ID %s", subscriptionID)

	subscription, err := s.subscriptionRepo.GetSubscriptionByRazorpaySubscriptionID(ctx, subscriptionID)
	if err != nil {
		log.Printf("Failed to find subscription with ID %s: %v", subscriptionID, err)
		return fmt.Errorf("failed to find subscription with ID %s: %v", subscriptionID, err)
	}
	
	if subscription == nil {
		log.Printf("No subscription found with Razorpay subscription ID: %s", subscriptionID)
		return fmt.Errorf("no subscription found with Razorpay subscription ID: %s", subscriptionID)
	}

	if err := s.subscriptionRepo.StopSubscription(ctx, subscription.ID, subscription.UserID); err != nil {
		log.Printf("Failed to deactivate subscription: %v", err)
		return fmt.Errorf("failed to deactivate subscription: %v", err)
	}
	
	log.Printf("Subscription cancelled: Subscription ID %s", subscriptionID)
	return nil
}

func (s *DefaultRazorpayService) handlePaymentFailed(ctx context.Context, event map[string]interface{}) error {
	log.Println("Processing payment.failed webhook")

	payloadObj, _ := event["payload"].(map[string]interface{})
	if payloadObj == nil {
		return fmt.Errorf("invalid payload in webhook event")
	}
	
	paymentObj, _ := payloadObj["payment"].(map[string]interface{})
	if paymentObj == nil {
		return fmt.Errorf("invalid payment data in webhook payload")
	}

	entity, _ := paymentObj["entity"].(map[string]interface{})
	if entity == nil {
		return fmt.Errorf("invalid entity in payment data")
	}
	
	orderID, _ := entity["order_id"].(string)
	subscriptionID, _ := entity["subscription_id"].(string)

	log.Printf("Payment failed: Order ID %s, Subscription ID %s", orderID, subscriptionID)

	return nil
}