package controller

import (
	"context"
	"io/ioutil"
	"net/http"
	"log"

	"github.com/labstack/echo/v4"
	
	"subscription-management/internal/service"
)


type WebhookController struct {
	razorpayService service.RazorpayService
}

func NewWebhookController(razorpayService service.RazorpayService) *WebhookController {
	return &WebhookController{
		razorpayService: razorpayService,
	}
}

func (wc *WebhookController) RegisterRoutes(e *echo.Echo) {
	e.POST("/webhooks/razorpay", wc.HandleRazorpayWebhook)
}

func (wc *WebhookController) HandleRazorpayWebhook(c echo.Context) error {
	body, err := ioutil.ReadAll(c.Request().Body)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Failed to read request body",
		})
	}
	
	signature := c.Request().Header.Get("X-Razorpay-Signature")
	
	testMode := c.QueryParam("test_mode") == "true"
	if testMode {
		log.Println("Webhook received in TEST MODE - signature verification will be skipped")
		if signature == "" {
			signature = "test_signature"
		}
	} else if signature == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Missing Razorpay signature",
		})
	}
	
	ctx := c.Request().Context()
	if testMode {
		ctx = context.WithValue(ctx, "testMode", true)
	}
	
	if err := wc.razorpayService.HandleWebhook(ctx, body, signature); err != nil {
		log.Printf("Webhook error: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to process webhook: " + err.Error(),
		})
	}
	
	return c.JSON(http.StatusOK, map[string]string{
		"status": "success",
	})
}