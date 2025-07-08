package main

import (
	"log"
	"net/http"
	"os"

	"github.com/Jiang-hao/walletApiService/internal/api"
	"github.com/Jiang-hao/walletApiService/internal/repository"
	"github.com/Jiang-hao/walletApiService/internal/service"
	"github.com/Jiang-hao/walletApiService/package/database"
	"github.com/gin-gonic/gin"
)

func main() {
	// Initialize database
	db, err := database.NewPostgresDB(database.Config{
		Host:     getEnv("DB_HOST", "localhost"),
		Port:     getEnv("DB_PORT", "5432"),
		User:     getEnv("DB_USER", "postgres"),
		Password: getEnv("DB_PASSWORD", "920313"),
		DBName:   getEnv("DB_NAME", "walletapi"),
		SSLMode:  getEnv("DB_SSL_MODE", "disable"),
	})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Initialize repositories
	walletRepo := repository.NewWalletRepository(db)
	transactionRepo := repository.NewTransactionRepository(db)

	// Initialize services
	walletService := service.NewWalletService(
		walletRepo,
		transactionRepo,
		transactionRepo.(repository.TxManager),
	)

	// Initialize handlers
	walletHandler := api.NewWalletHandler(walletService)

	// Set up router
	router := gin.Default()

	// API routes
	apiGroup := router.Group("/api/v1")
	{
		users := apiGroup.Group("/wallet")
		{
			users.POST("/deposit", walletHandler.Deposit)
			users.POST("/withdraw", walletHandler.Withdraw)
			users.POST("/transfer", walletHandler.Transfer)
			users.GET("/balance", walletHandler.GetBalance)
			users.GET("/transactions", walletHandler.GetTransactionHistory)
		}
	}

	// Health check
	router.GET("/api/v1/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// Start server
	port := getEnv("PORT", "8080")
	log.Printf("Server starting on port %s", port)
	if err := router.Run(":" + port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}
