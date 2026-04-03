package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"booking-service/internal/config"
	"booking-service/internal/handler"
	"booking-service/internal/middleware"
	"booking-service/internal/repository/postgres_sql"
	"booking-service/internal/service"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		cfg.DBHost, cfg.DBPort, cfg.DBUser, cfg.DBPassword, cfg.DBName)
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(25)
	db.SetConnMaxLifetime(5 * time.Minute)

	if err := db.Ping(); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}
	log.Println("Database connection established")

	if err := runMigrations(db, "file://migrations"); err != nil {
		log.Printf("Warning: Failed to run migrations: %v", err)
		log.Println("Continuing without migrations...")
	} else {
		log.Println("Migrations applied successfully")
	}

	userRepo := postgres_sql.NewUserRepository(db)
	roomRepo := postgres_sql.NewRoomRepository(db)
	scheduleRepo := postgres_sql.NewScheduleRepository(db)
	slotRepo := postgres_sql.NewSlotRepository(db)
	bookingRepo := postgres_sql.NewBookingRepository(db)

	authService := service.NewAuthService(userRepo, cfg.JWTSecret, cfg.JWTExpiration)
	roomService := service.NewRoomService(roomRepo)

	slotService := service.NewSlotService(slotRepo, roomRepo, scheduleRepo)

	scheduleService := service.NewScheduleService(scheduleRepo, roomRepo, slotRepo, cfg.SlotDuration, slotService)

	conferenceService := service.NewMockConferenceService(false)
	bookingService := service.NewBookingService(bookingRepo, slotRepo, conferenceService, cfg.MaxPageSize, cfg.DefaultPageSize)

	authHandler := handler.NewAuthHandler(authService)
	roomHandler := handler.NewRoomHandler(roomService)
	scheduleHandler := handler.NewScheduleHandler(scheduleService)
	slotHandler := handler.NewSlotHandler(slotService)
	bookingHandler := handler.NewBookingHandler(bookingService)
	infoHandler := handler.NewInfoHandler()

	r := mux.NewRouter()

	r.HandleFunc("/_info", infoHandler.Info).Methods("GET")
	r.HandleFunc("/dummyLogin", authHandler.DummyLogin).Methods("POST")
	r.HandleFunc("/register", authHandler.Register).Methods("POST")
	r.HandleFunc("/login", authHandler.Login).Methods("POST")

	api := r.PathPrefix("/").Subrouter()
	api.Use(middleware.AuthMiddleware(authService))

	api.HandleFunc("/rooms/list", roomHandler.List).Methods("GET")
	api.Handle("/rooms/create", middleware.RoleMiddleware("admin")(http.HandlerFunc(roomHandler.Create))).Methods("POST")

	api.Handle("/rooms/{roomId}/schedule/create", middleware.RoleMiddleware("admin")(http.HandlerFunc(scheduleHandler.Create))).Methods("POST")

	api.HandleFunc("/rooms/{roomId}/slots/list", slotHandler.ListAvailable).Methods("GET")

	api.Handle("/bookings/create", middleware.RoleMiddleware("user")(http.HandlerFunc(bookingHandler.Create))).Methods("POST")
	api.Handle("/bookings/list", middleware.RoleMiddleware("admin")(http.HandlerFunc(bookingHandler.ListAll))).Methods("GET")
	api.Handle("/bookings/my", middleware.RoleMiddleware("user")(http.HandlerFunc(bookingHandler.ListMy))).Methods("GET")
	api.Handle("/bookings/{bookingId}/cancel", middleware.RoleMiddleware("user")(http.HandlerFunc(bookingHandler.Cancel))).Methods("POST")

	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Printf("Server starting on port %s", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited")
}

func runMigrations(db *sql.DB, migrationsPath string) error {
	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		return fmt.Errorf("failed to create migration driver: %w", err)
	}

	m, err := migrate.NewWithDatabaseInstance(
		migrationsPath,
		"postgres",
		driver,
	)
	if err != nil {
		return fmt.Errorf("failed to create migration instance: %w", err)
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("failed to apply migrations: %w", err)
	}

	return nil
}
