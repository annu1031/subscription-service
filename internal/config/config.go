package config

import (
	"fmt"
	"os"
)


type Config struct {
	Server   ServerConfig
	DB       DBConfig
	Razorpay RazorpayConfig
}


type ServerConfig struct {
	Port string
}


type DBConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	DBName   string
	SSLMode  string
}


type RazorpayConfig struct {
	KeyID     string
	KeySecret string
	WebhookSecret string
}


func (c *DBConfig) GetDSN() string {
	return fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true", 
		c.User, c.Password, c.Host, c.Port, c.DBName)
}


func Load() *Config {
	return &Config{
		
		Server: ServerConfig{
			Port: getEnv("PORT", "8080"),
		},
		DB: DBConfig{
			Host:     getEnv("DB_HOST", "localhost"),
			Port:     getEnv("DB_PORT", "3306"),
			User:     getEnv("DB_USER", "appuser"),
			Password: getEnv("DB_PASSWORD", "apppassword"),
			DBName:   getEnv("DB_NAME", "subscription_db"),
			SSLMode:  getEnv("DB_SSL_MODE", "disable"),
		},
		Razorpay: RazorpayConfig{
			KeyID:         "rzp_test_vwhPl6Bttiko87",
			KeySecret:     "SxUsGLn9WAarMLSGrXBFB08o",
			WebhookSecret: "webhook123",
		},
	}
}


func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}