package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"subscription-management/internal/model"
)

type SubscriptionRepository interface {
	GetProducts(ctx context.Context) ([]model.SubscriptionProduct, error)
	GetPlans(ctx context.Context, productID string) ([]model.SubscriptionPlan, error)
	GetPlanWithAttributes(ctx context.Context, planID string) (*model.SubscriptionPlanWithAttributes, error)
	GetActiveSubscription(ctx context.Context, userID string) (*model.SubscriptionTransaction, error)
	GetSubscriptionHistory(ctx context.Context, userID string) ([]model.SubscriptionTransaction, error)
	CreateSubscription(ctx context.Context, subscription *model.SubscriptionTransaction) error
	StopSubscription(ctx context.Context, subscriptionID string, userID string) error
	GetSubscriptionByID(ctx context.Context, subscriptionID string) (*model.SubscriptionTransaction, error)
	GetSubscriptionByRazorpayOrderID(ctx context.Context, orderID string) (*model.SubscriptionTransaction, error)
	GetSubscriptionByRazorpaySubscriptionID(ctx context.Context, subscriptionID string) (*model.SubscriptionTransaction, error)
	UpdateSubscription(ctx context.Context, subscription *model.SubscriptionTransaction) error
}


type SQLSubscriptionRepository struct {
	db *sqlx.DB
}


func NewSubscriptionRepository(db *sqlx.DB) SubscriptionRepository {
	return &SQLSubscriptionRepository{
		db: db,
	}
}

func (r *SQLSubscriptionRepository) GetProducts(ctx context.Context) ([]model.SubscriptionProduct, error) {
	var products []model.SubscriptionProduct
	
	query := `SELECT * FROM subscription_products`
	
	err := r.db.SelectContext(ctx, &products, query)
	if err != nil {
		return nil, err
	}
	
	return products, nil
}

func (r *SQLSubscriptionRepository) GetPlans(ctx context.Context, productID string) ([]model.SubscriptionPlan, error) {
	var plans []model.SubscriptionPlan
	
	var query string
	var args []interface{}
	
	if productID != "" {
		query = `SELECT * FROM subscription_plans WHERE product_id = ?`
		args = append(args, productID)
	} else {
		query = `SELECT * FROM subscription_plans`
	}
	
	err := r.db.SelectContext(ctx, &plans, query, args...)
	if err != nil {
		return nil, err
	}

	for i := range plans {
		attributes, err := r.getPlanAttributes(ctx, plans[i].ID)
		if err != nil {
			return nil, err
		}
		plans[i].Attributes = attributes
	}
	
	return plans, nil
}

func (r *SQLSubscriptionRepository) GetPlanWithAttributes(ctx context.Context, planID string) (*model.SubscriptionPlanWithAttributes, error) {
	var plan model.SubscriptionPlan
	
	planQuery := `SELECT * FROM subscription_plans WHERE id = ?`
	
	err := r.db.GetContext(ctx, &plan, planQuery, planID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil 
		}
		return nil, err
	}
	
	attributes, err := r.getPlanAttributes(ctx, planID)
	if err != nil {
		return nil, err
	}
	
	return &model.SubscriptionPlanWithAttributes{
		Plan:       plan,
		Attributes: attributes,
	}, nil
}

func (r *SQLSubscriptionRepository) getPlanAttributes(ctx context.Context, planID string) ([]model.SubscriptionProductAttribute, error) {
	var attributes []model.SubscriptionProductAttribute
	
	query := `
		SELECT a.* FROM subscription_product_attributes a
		JOIN subscription_plan_attributes pa ON a.id = pa.attribute_id
		WHERE pa.plan_id = ?
	`
	
	err := r.db.SelectContext(ctx, &attributes, query, planID)
	if err != nil {
		return nil, err
	}
	
	return attributes, nil
}

func (r *SQLSubscriptionRepository) GetActiveSubscription(ctx context.Context, userID string) (*model.SubscriptionTransaction, error) {
	var subscription model.SubscriptionTransaction
	
	query := `
		SELECT t.* FROM subscription_transactions t
		WHERE t.user_id = ? AND t.is_active = true
		LIMIT 1
	`
	
	err := r.db.GetContext(ctx, &subscription, query, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil 
		}
		return nil, err
	}
	
	if err := r.enrichSubscriptionData(ctx, &subscription); err != nil {
		return nil, err
	}
	
	return &subscription, nil
}

func (r *SQLSubscriptionRepository) GetSubscriptionHistory(ctx context.Context, userID string) ([]model.SubscriptionTransaction, error) {
	var subscriptions []model.SubscriptionTransaction
	
	query := `
		SELECT * FROM subscription_transactions
		WHERE user_id = ?
		ORDER BY created_at DESC
	`
	
	err := r.db.SelectContext(ctx, &subscriptions, query, userID)
	if err != nil {
		return nil, err
	}

	for i := range subscriptions {
		if err := r.enrichSubscriptionData(ctx, &subscriptions[i]); err != nil {
			return nil, err
		}
	}
	
	return subscriptions, nil
}

func (r *SQLSubscriptionRepository) CreateSubscription(ctx context.Context, subscription *model.SubscriptionTransaction) error {
	if subscription.ID == "" {
		subscription.ID = uuid.New().String()
	}
	
	deactivateQuery := `
		UPDATE subscription_transactions
		SET is_active = false
		WHERE user_id = ? AND is_active = true
	`
	
	_, err := r.db.ExecContext(ctx, deactivateQuery, subscription.UserID)
	if err != nil {
		return err
	}
	
	
	subscription.CreatedAt = time.Now()
	subscription.UpdatedAt = time.Now()
	
	insertQuery := `
		INSERT INTO subscription_transactions (
			id, user_id, product_id, plan_id, card_id,
			is_renewal, is_active, payment_type, amount,
			start_date, end_date, next_renewal_date,
			razorpay_order_id, razorpay_payment_id, razorpay_subscription_id,
			auto_renewal, created_at, updated_at
		) VALUES (
			:id, :user_id, :product_id, :plan_id, :card_id,
			:is_renewal, :is_active, :payment_type, :amount,
			:start_date, :end_date, :next_renewal_date,
			:razorpay_order_id, :razorpay_payment_id, :razorpay_subscription_id,
			:auto_renewal, :created_at, :updated_at
		)
	`
	
	_, err = r.db.NamedExecContext(ctx, insertQuery, subscription)
	return err
}

func (r *SQLSubscriptionRepository) StopSubscription(ctx context.Context, subscriptionID string, userID string) error {
	query := `
		UPDATE subscription_transactions
		SET is_active = false, updated_at = NOW()
		WHERE id = ? AND user_id = ? AND is_active = true
	`
	
	result, err := r.db.ExecContext(ctx, query, subscriptionID, userID)
	if err != nil {
		return err
	}
	
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	
	if rows == 0 {
		return errors.New("subscription not found or already inactive")
	}
	
	return nil
}

func (r *SQLSubscriptionRepository) GetSubscriptionByID(ctx context.Context, subscriptionID string) (*model.SubscriptionTransaction, error) {
	var subscription model.SubscriptionTransaction
	
	query := `SELECT * FROM subscription_transactions WHERE id = ?`
	
	err := r.db.GetContext(ctx, &subscription, query, subscriptionID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil 
		}
		return nil, err
	}
	
	if err := r.enrichSubscriptionData(ctx, &subscription); err != nil {
		return nil, err
	}
	
	return &subscription, nil
}

func (r *SQLSubscriptionRepository) enrichSubscriptionData(ctx context.Context, subscription *model.SubscriptionTransaction) error {

	var planName string
	planQuery := `SELECT name FROM subscription_plans WHERE id = ?`
	err := r.db.GetContext(ctx, &planName, planQuery, subscription.PlanID)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	subscription.PlanName = planName

	var productName string
	productQuery := `SELECT name FROM subscription_products WHERE id = ?`
	err = r.db.GetContext(ctx, &productName, productQuery, subscription.ProductID)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	subscription.ProductName = productName
	
	var cardLastFour string
	cardQuery := `SELECT last_four_digits FROM cards WHERE id = ?`
	err = r.db.GetContext(ctx, &cardLastFour, cardQuery, subscription.CardID)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	subscription.CardLastFour = cardLastFour
	
	return nil
}

func (r *SQLSubscriptionRepository) GetSubscriptionByRazorpayOrderID(ctx context.Context, orderID string) (*model.SubscriptionTransaction, error) {
	var subscription model.SubscriptionTransaction
	
	query := `SELECT * FROM subscription_transactions WHERE razorpay_order_id = ?`
	
	err := r.db.GetContext(ctx, &subscription, query, orderID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil 
		}
		return nil, err
	}
	
	
	if err := r.enrichSubscriptionData(ctx, &subscription); err != nil {
		return nil, err
	}
	
	return &subscription, nil
}

func (r *SQLSubscriptionRepository) GetSubscriptionByRazorpaySubscriptionID(ctx context.Context, subscriptionID string) (*model.SubscriptionTransaction, error) {
	var subscription model.SubscriptionTransaction
	
	query := `SELECT * FROM subscription_transactions WHERE razorpay_subscription_id = ?`
	
	err := r.db.GetContext(ctx, &subscription, query, subscriptionID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil 
		}
		return nil, err
	}
	
	
	if err := r.enrichSubscriptionData(ctx, &subscription); err != nil {
		return nil, err
	}
	
	return &subscription, nil
}

func (r *SQLSubscriptionRepository) UpdateSubscription(ctx context.Context, subscription *model.SubscriptionTransaction) error {
	subscription.UpdatedAt = time.Now()
	
	query := `
		UPDATE subscription_transactions SET
			is_active = :is_active,
			razorpay_payment_id = :razorpay_payment_id,
			razorpay_order_id = :razorpay_order_id,
			razorpay_subscription_id = :razorpay_subscription_id,
			next_renewal_date = :next_renewal_date,
			updated_at = :updated_at
		WHERE id = :id
	`
	
	_, err := r.db.NamedExecContext(ctx, query, subscription)
	return err
}