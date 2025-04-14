package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	_ "github.com/go-sql-driver/mysql"

	"subscription-management/internal/config"
	"subscription-management/internal/controller"
	"subscription-management/internal/razorpay"
	"subscription-management/internal/repository"
	"subscription-management/internal/service"
)

func main() {
	cfg := config.Load()

	
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.Println("Starting subscription management service...")
	
	
	if cfg.Razorpay.KeyID == "" {
		log.Println("WARNING: Razorpay Key ID is empty")
	} else {
		log.Printf("Razorpay Key ID: %s (length: %d)", 
			maskString(cfg.Razorpay.KeyID), len(cfg.Razorpay.KeyID))
	}
	
	if cfg.Razorpay.KeySecret == "" {
		log.Println("WARNING: Razorpay Key Secret is empty")
	} else {
		log.Printf("Razorpay Key Secret length: %d", len(cfg.Razorpay.KeySecret))
	}

	
	db, err := setupDatabase(cfg)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	
	cardRepo := repository.NewCardRepository(db)
	subscriptionRepo := repository.NewSubscriptionRepository(db)

	
	razorpayClient := razorpay.NewClient(razorpay.Config{
		KeyID:     cfg.Razorpay.KeyID,
		KeySecret: cfg.Razorpay.KeySecret,
	})

	
	cardService := service.NewCardService(cardRepo)
	razorpayService := service.NewRazorpayService(
		razorpayClient,
		subscriptionRepo,
		cfg.Razorpay.WebhookSecret,
	)
	subscriptionService := service.NewSubscriptionService(
		subscriptionRepo,
		cardRepo,
		razorpayService,
		cfg, 
	)

	
	cardController := controller.NewCardController(cardService)
	subscriptionController := controller.NewSubscriptionController(
		subscriptionService,
		razorpayService, 
	)
	webhookController := controller.NewWebhookController(razorpayService)

	
	e := echo.New()
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.CORS())

	
	cardController.RegisterRoutes(e)
	subscriptionController.RegisterRoutes(e)
	webhookController.RegisterRoutes(e)

	
	e.GET("/health", func(c echo.Context) error {
		return c.JSON(200, map[string]string{"status": "healthy"})
	})

	
	go func() {
		if err := e.Start(":" + cfg.Server.Port); err != nil {
			log.Printf("Server shutdown: %v", err)
		}
	}()

	
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := e.Shutdown(ctx); err != nil {
		log.Fatal(err)
	}
}


func setupDatabase(cfg *config.Config) (*sqlx.DB, error) {
	db, err := sqlx.Connect("mysql", cfg.DB.GetDSN())
	if err != nil {
		return nil, err
	}

	
	db.SetMaxIdleConns(5)
	db.SetMaxOpenConns(20)
	db.SetConnMaxLifetime(time.Hour)

	return db, nil
}


func maskString(s string) string {
	if len(s) <= 2 {
		return "**"
	}
	return s[:2] + "..." + s[len(s)-2:]
}