package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	Port            string
	DBHost          string
	DBPort          string
	DBUser          string
	DBPassword      string
	DBName          string
	JWTSecret       string
	JWTExpiration   time.Duration
	MaxPageSize     int
	DefaultPageSize int
	SlotDuration    time.Duration
	SlotFutureDays  int
}

func Load() (*Config, error) {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	jwtExpiration, err := time.ParseDuration(os.Getenv("JWT_EXPIRATION"))
	if err != nil {
		jwtExpiration = 24 * time.Hour
	}

	maxPageSize, _ := strconv.Atoi(os.Getenv("MAX_PAGE_SIZE"))
	if maxPageSize == 0 {
		maxPageSize = 100
	}

	defaultPageSize, _ := strconv.Atoi(os.Getenv("DEFAULT_PAGE_SIZE"))
	if defaultPageSize == 0 {
		defaultPageSize = 20
	}

	slotFutureDays, _ := strconv.Atoi(os.Getenv("SLOT_FUTURE_DAYS"))
	if slotFutureDays == 0 {
		slotFutureDays = 7
	}

	return &Config{
		Port:            port,
		DBHost:          os.Getenv("DB_HOST"),
		DBPort:          os.Getenv("DB_PORT"),
		DBUser:          os.Getenv("DB_USER"),
		DBPassword:      os.Getenv("DB_PASSWORD"),
		DBName:          os.Getenv("DB_NAME"),
		JWTSecret:       os.Getenv("JWT_SECRET"),
		JWTExpiration:   jwtExpiration,
		MaxPageSize:     maxPageSize,
		DefaultPageSize: defaultPageSize,
		SlotDuration:    30 * time.Minute,
		SlotFutureDays:  slotFutureDays,
	}, nil
}
