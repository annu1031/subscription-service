package service

import (
	"context"
	"database/sql"
	"errors"
	"time"
	"log"
	"github.com/google/uuid"

	"subscription-management/internal/model"
	"subscription-management/internal/repository"
	"subscription-management/internal/config"
)

var (
    ErrInvalidPlan          = errors.New("invalid subscription plan")
    ErrInvalidCard          = errors.New("invalid card")
    ErrInvalidPaymentType   = errors.New("payment type must be 'monthly' or 'yearly'")
    ErrSubscriptionNotFound = errors.New("subscription not found")
)

type DefaultSubscriptionService struct {
	subscriptionRepo repository.SubscriptionRepository
	cardRepo         repository.CardRepository
	razorpayService  RazorpayService
	config           *config.Config
}

type SubscriptionService interface {
	GetAvailablePlans(ctx context.Context) ([]model.SubscriptionPlan, error)
	GetActiveSubscription(ctx context.Context, userID string) (*model.SubscriptionTransaction, error)
	GetSubscriptionHistory(ctx context.Context, userID string) ([]model.SubscriptionTransaction, error)
	CreateSubscription(ctx context.Context, request *model.SubscriptionRequest, userInfo *model.UserInfo) (*model.SubscriptionTransaction, error)
	RenewSubscription(ctx context.Context, subscriptionID string, userID string) (*model.SubscriptionTransaction, error)
	StopSubscription(ctx context.Context, subscriptionID string, userID string) error
}

func NewSubscriptionService(
	subscriptionRepo repository.SubscriptionRepository,
	cardRepo repository.CardRepository,
	razorpayService RazorpayService,
	config *config.Config,
) SubscriptionService {
	return &DefaultSubscriptionService{
		subscriptionRepo: subscriptionRepo,
		cardRepo:         cardRepo,
		razorpayService:  razorpayService,
		config:           config,
	}
}

func (s *DefaultSubscriptionService) GetAvailablePlans(ctx context.Context) ([]model.SubscriptionPlan, error) {
    return s.subscriptionRepo.GetPlans(ctx, "")
}

func (s *DefaultSubscriptionService) GetActiveSubscription(ctx context.Context, userID string) (*model.SubscriptionTransaction, error) {
	return s.subscriptionRepo.GetActiveSubscription(ctx, userID)
}

func (s *DefaultSubscriptionService) GetSubscriptionHistory(ctx context.Context, userID string) ([]model.SubscriptionTransaction, error) {
	return s.subscriptionRepo.GetSubscriptionHistory(ctx, userID)
}

func (s *DefaultSubscriptionService) CreateSubscription(ctx context.Context, request *model.SubscriptionRequest, userInfo *model.UserInfo) (*model.SubscriptionTransaction, error) {
    log.Println("Creating subscription:", request)

    if request.PaymentType != "monthly" && request.PaymentType != "yearly" {
        log.Println("Invalid payment type:", request.PaymentType)
        return nil, ErrInvalidPaymentType
    }

    card, err := s.cardRepo.GetByID(ctx, request.CardID)
    if err != nil {
        log.Println("Error getting card:", err)
        return nil, err
    }
    if card == nil {
        log.Println("Card not found:", request.CardID)
        return nil, ErrInvalidCard
    }
    if card.UserID != request.UserID {
        log.Println("Card doesn't belong to user. CardUserID:", card.UserID, "RequestUserID:", request.UserID)
        return nil, ErrInvalidCard
    }
    log.Println("Card validation successful")

    planWithAttrs, err := s.subscriptionRepo.GetPlanWithAttributes(ctx, request.PlanID)
    if err != nil {
        log.Println("Error getting plan:", err)
        return nil, err
    }
    if planWithAttrs == nil {
        log.Println("Plan not found:", request.PlanID)
        return nil, ErrInvalidPlan
    }
    log.Println("Plan validation successful")

    var amount float64
    var startDate, endDate, nextRenewalDate time.Time
    
    startDate = time.Now()
    
    if request.PaymentType == "monthly" {
        amount = planWithAttrs.Plan.PriceMonthly
        endDate = startDate.AddDate(0, 1, 0)
    } else { 
        amount = planWithAttrs.Plan.PriceYearly
        endDate = startDate.AddDate(1, 0, 0)
    }

    nextRenewalDate = endDate
    log.Println("Calculated dates - Start:", startDate, "End:", endDate, "Next Renewal:", nextRenewalDate)

    subscription := &model.SubscriptionTransaction{
        ID:              uuid.New().String(), 
        UserID:          request.UserID,
        ProductID:       request.ProductID,
        PlanID:          request.PlanID,
        CardID:          request.CardID,
        IsRenewal:       false,
        IsActive:        true,
        PaymentType:     request.PaymentType,
        Amount:          amount,
        StartDate:       startDate,
        EndDate:         endDate,
        NextRenewalDate: nextRenewalDate,
        AutoRenewal:     request.AutoRenewal,
        RazorpayKeyID:   s.config.Razorpay.KeyID,

        RazorpayOrderID:        sql.NullString{},
        RazorpayPaymentID:      sql.NullString{},
        RazorpaySubscriptionID: sql.NullString{},
    }
    log.Println("Created subscription object:", subscription)
    
 
    razorpayEnabled := true 
    
    if s.razorpayService == nil {
        log.Println("Warning: Razorpay service is not configured")
        razorpayEnabled = false
    }
    
    if !razorpayEnabled {
        log.Println("Razorpay integration is disabled")
    } else {
        if request.AutoRenewal {
            log.Println("Setting up auto-renewal with Razorpay")
            razorpaySub, err := s.razorpayService.CreateSubscription(ctx, subscription, userInfo)
            if err != nil {
                log.Println("Error creating Razorpay subscription:", err)
                return nil, err
            }

            if razorpaySubID, ok := razorpaySub["id"].(string); ok {
                subscription.RazorpaySubscriptionID = toNullString(razorpaySubID)
                log.Println("Razorpay subscription ID:", razorpaySubID)
            } else {
                log.Println("Warning: Could not extract Razorpay subscription ID")
            }
        } else {
            log.Println("Setting up one-time payment with Razorpay")
          
            order, err := s.razorpayService.CreatePayment(
                ctx, 
                subscription.Amount, 
                "INR", 
                subscription.ID, 
            )
            
            if err != nil {
                log.Println("Error creating Razorpay payment:", err)
                return nil, err
            }

            if razorpayOrderID, ok := order["id"].(string); ok {
                subscription.RazorpayOrderID = toNullString(razorpayOrderID)
                log.Println("Razorpay order ID:", razorpayOrderID)
            } else {
                log.Println("Warning: Could not extract Razorpay order ID")
            }
        }
    }
    
    log.Println("Saving subscription to database")
    if err := s.subscriptionRepo.CreateSubscription(ctx, subscription); err != nil {
        log.Println("Error saving subscription:", err)
        return nil, err
    }
    
    log.Println("Getting complete subscription data")
    return s.subscriptionRepo.GetSubscriptionByID(ctx, subscription.ID)
}

func (s *DefaultSubscriptionService) RenewSubscription(ctx context.Context, subscriptionID string, userID string) (*model.SubscriptionTransaction, error) {
    subscription, err := s.subscriptionRepo.GetSubscriptionByID(ctx, subscriptionID)
    if err != nil {
        return nil, err
    }
    
    if subscription == nil {
        return nil, ErrSubscriptionNotFound
    }
    
    if subscription.UserID != userID {
        return nil, ErrUnauthorized 
    }
    

    newSubscription := &model.SubscriptionTransaction{
        UserID:          userID,
        ProductID:       subscription.ProductID,
        PlanID:          subscription.PlanID,
        CardID:          subscription.CardID,
        IsRenewal:       true,
        IsActive:        true,
        PaymentType:     subscription.PaymentType,
        Amount:          subscription.Amount,
        StartDate:       time.Now(),
        EndDate:         time.Now().AddDate(0, 1, 0), 
        NextRenewalDate: time.Now().AddDate(0, 1, 0),
        
        RazorpayOrderID:        sql.NullString{},
        RazorpayPaymentID:      sql.NullString{},
        RazorpaySubscriptionID: sql.NullString{},
    }
    
    
    err = s.subscriptionRepo.CreateSubscription(ctx, newSubscription)
    if err != nil {
        return nil, err
    }
    
    return s.subscriptionRepo.GetSubscriptionByID(ctx, newSubscription.ID)
}

func (s *DefaultSubscriptionService) StopSubscription(ctx context.Context, subscriptionID string, userID string) error {
	
	subscription, err := s.subscriptionRepo.GetSubscriptionByID(ctx, subscriptionID)
	if err != nil {
		return err
	}
	
	if subscription == nil {
		return ErrSubscriptionNotFound
	}
	
	if subscription.UserID != userID {
		return ErrUnauthorized
	}
	
	if subscription.RazorpaySubscriptionID.Valid && subscription.RazorpaySubscriptionID.String != "" {
		if err := s.razorpayService.CancelSubscription(ctx, subscription.RazorpaySubscriptionID.String); err != nil {
			return err
		}
	}
	
	return s.subscriptionRepo.StopSubscription(ctx, subscriptionID, userID)
}