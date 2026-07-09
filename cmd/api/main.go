package main

import (
	"log"
	"os"

	"suratnesia/internal/api/middleware"
	"suratnesia/internal/api/router"
	"suratnesia/internal/config"
	"suratnesia/internal/repository"
)

func main() {
	cfg := config.Load()

	// Initialize real database pool
	db, err := repository.InitDB(cfg)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}

	// Setup server and router
	e := router.New()

	// Global Middlewares
	e.Use(middleware.TenantMiddleware(db))

	port := cfg.Port
	if os.Getenv("PORT") != "" {
		port = os.Getenv("PORT")
	}

	log.Printf("Server starting on port %s", port)
	if err := e.Start(":" + port); err != nil {
		log.Fatalf("server shut down unexpectedly: %v", err)
	}
}
