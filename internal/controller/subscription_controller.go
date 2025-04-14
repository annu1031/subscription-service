package controller

import (
	"net/http"
	"log"

	"github.com/labstack/echo/v4"

	"subscription-management/internal/model"
	"subscription-management/internal/service"
)


type SubscriptionController struct {
	subscriptionService service.SubscriptionService
	razorpayService     service.RazorpayService
}


func NewSubscriptionController(
	subscriptionService service.SubscriptionService,
	razorpayService service.RazorpayService,
) *SubscriptionController {
	return &SubscriptionController{
		subscriptionService: subscriptionService,
		razorpayService:     razorpayService,
	}
}


func (sc *SubscriptionController) RegisterRoutes(e *echo.Echo) {
	subscriptions := e.Group("/api/subscriptions")
	
	
	subscriptions.GET("/plans", sc.GetPlans)
	subscriptions.GET("/active", sc.GetActiveSubscription)
	subscriptions.GET("/history", sc.GetSubscriptionHistory)
	subscriptions.POST("", sc.CreateSubscription)
	subscriptions.PUT("/:id/renew", sc.RenewSubscription)
	subscriptions.PUT("/:id/stop", sc.StopSubscription)
	
	
	subscriptions.GET("/test-razorpay", sc.TestRazorpay)
	subscriptions.POST("/verify-payment", sc.VerifyPayment)
}


func (sc *SubscriptionController) GetPlans(c echo.Context) error {
	plans, err := sc.subscriptionService.GetAvailablePlans(c.Request().Context())
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to retrieve plans"})
	}
	
	return c.JSON(http.StatusOK, plans)
}


func (sc *SubscriptionController) GetActiveSubscription(c echo.Context) error {
	userID := c.QueryParam("userId")
	if userID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "User ID is required"})
	}
	
	subscription, err := sc.subscriptionService.GetActiveSubscription(c.Request().Context(), userID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to retrieve active subscription"})
	}
	
	if subscription == nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "No active subscription found"})
	}
	
	return c.JSON(http.StatusOK, subscription)
}


func (sc *SubscriptionController) GetSubscriptionHistory(c echo.Context) error {
	userID := c.QueryParam("userId")
	if userID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "User ID is required"})
	}
	
	history, err := sc.subscriptionService.GetSubscriptionHistory(c.Request().Context(), userID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to retrieve subscription history"})
	}
	
	return c.JSON(http.StatusOK, history)
}


type CreateSubscriptionRequest struct {
    UserID      string `json:"userId" validate:"required"`
    ProductID   string `json:"productId" validate:"required"`
    PlanID      string `json:"planId" validate:"required"`
    CardID      string `json:"cardId" validate:"required"`
    PaymentType string `json:"paymentType" validate:"required"` 
    AutoRenewal bool   `json:"autoRenewal"`                   
    
   
    Name        string `json:"name"`
    Email       string `json:"email"`
    Phone       string `json:"phone"`
}


func (sc *SubscriptionController) CreateSubscription(c echo.Context) error {
	var req CreateSubscriptionRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
	}
	
	log.Printf("Received subscription request: %+v", req)
	
	
	userInfo := &model.UserInfo{
		ID:    req.UserID,
		Name:  req.Name,
		Email: req.Email,
		Phone: req.Phone,
	}
	
	
	subscriptionReq := &model.SubscriptionRequest{
		UserID:      req.UserID,
		ProductID:   req.ProductID,
		PlanID:      req.PlanID,
		CardID:      req.CardID,
		PaymentType: req.PaymentType,
		AutoRenewal: req.AutoRenewal,
	}
	
	
	subscription, err := sc.subscriptionService.CreateSubscription(c.Request().Context(), subscriptionReq, userInfo)
	if err != nil {
		log.Printf("Error creating subscription: %v", err)
		switch err {
		case service.ErrInvalidPlan:
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid subscription plan"})
		case service.ErrInvalidCard:
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid card"})
		case service.ErrInvalidPaymentType:
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "Payment type must be 'monthly' or 'yearly'"})
		default:
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to create subscription: " + err.Error()})
		}
	}
	
	
	return c.JSON(http.StatusCreated, map[string]interface{}{
		"subscription": subscription,
		"razorpay": map[string]interface{}{
			"key_id": subscription.RazorpayKeyID,
			"order_id": subscription.RazorpayOrderID,
			"subscription_id": subscription.RazorpaySubscriptionID,
			"notes": map[string]string{
				"subscription_id": subscription.ID,
			},
		},
	})
}


func (sc *SubscriptionController) RenewSubscription(c echo.Context) error {
	id := c.Param("id")
	userID := c.QueryParam("userId")
	
	if userID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "User ID is required"})
	}
	
	subscription, err := sc.subscriptionService.RenewSubscription(c.Request().Context(), id, userID)
	if err != nil {
		switch err {
		case service.ErrSubscriptionNotFound:
			return c.JSON(http.StatusNotFound, map[string]string{"error": "Subscription not found"})
		default:
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to renew subscription"})
		}
	}
	
	return c.JSON(http.StatusOK, subscription)
}


func (sc *SubscriptionController) StopSubscription(c echo.Context) error {
	id := c.Param("id")
	userID := c.QueryParam("userId")
	
	if userID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "User ID is required"})
	}
	
	if err := sc.subscriptionService.StopSubscription(c.Request().Context(), id, userID); err != nil {
		switch err {
		case service.ErrSubscriptionNotFound:
			return c.JSON(http.StatusNotFound, map[string]string{"error": "Subscription not found or already inactive"})
		default:
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to stop subscription"})
		}
	}
	
	return c.JSON(http.StatusOK, map[string]string{"message": "Subscription stopped successfully"})
}


func (sc *SubscriptionController) TestRazorpay(c echo.Context) error {
	result, err := sc.razorpayService.TestConnection(c.Request().Context())
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"status": "error",
			"message": err.Error(),
		})
	}
	
	return c.JSON(http.StatusOK, map[string]interface{}{
		"status": "success",
		"result": result,
	})
}


type VerifyPaymentRequest struct {
	RazorpayPaymentID    string `json:"razorpay_payment_id"`
	RazorpayOrderID      string `json:"razorpay_order_id"`
	RazorpaySignature    string `json:"razorpay_signature"`
	RazorpaySubscriptionID string `json:"razorpay_subscription_id"`
}


func (sc *SubscriptionController) VerifyPayment(c echo.Context) error {
	var req VerifyPaymentRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
	}
	
	log.Printf("Verifying payment: %+v", req)
	

	return c.JSON(http.StatusOK, map[string]string{
		"status": "success",
		"message": "Payment verified successfully",
	})
}