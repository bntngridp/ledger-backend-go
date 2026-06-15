package main

import (
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/bntngridp/ledger-backend-go/internal/delivery"
	repo "github.com/bntngridp/ledger-backend-go/internal/repository"
	"github.com/bntngridp/ledger-backend-go/internal/usecase"
	"github.com/bntngridp/ledger-backend-go/pkg/database"
	"github.com/bntngridp/ledger-backend-go/pkg/middleware"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func getEnv(key, fallback string) string {
	if val, ok := os.LookupEnv(key); ok {
		return val
	}
	return fallback
}

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("no .env file found, using system env")
	}

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		log.Fatal("JWT_SECRET is required (set in .env or system env)")
	}
	expiryHoursStr := getEnv("JWT_EXPIRY_HOURS", "24")
	expiryHours, err := strconv.Atoi(expiryHoursStr)
	if err != nil {
		expiryHours = 24
	}
	port := getEnv("PORT", "8080")

	dbCfg := database.Config{
		Host:     getEnv("DB_HOST", "localhost"),
		Port:     getEnv("DB_PORT", "5432"),
		User:     getEnv("DB_USER", "postgres"),
		Password: getEnv("DB_PASSWORD", ""),
		DBName:   getEnv("DB_NAME", "ledger_db"),
		SSLMode:  getEnv("DB_SSLMODE", "disable"),
		LogLevel: getEnv("DB_LOG_LEVEL", "warn"),
	}

	db, err := database.InitDB(dbCfg)
	if err != nil {
		log.Fatalf("database init failed: %v", err)
	}

	if err := database.RunMigrations(db); err != nil {
		log.Fatalf("migration failed: %v", err)
	}

	userRepo := repo.NewUserRepository(db)
	walletRepo := repo.NewWalletRepository(db)
	txRepo := repo.NewTransactionRepository(db)

	authUC := usecase.NewAuthUsecase(userRepo, walletRepo)
	transferUC := usecase.NewTransferUsecase(walletRepo, txRepo)
	walletUC := usecase.NewWalletUsecase(walletRepo, txRepo)

	authHandler := delivery.NewAuthHandler(authUC, jwtSecret, expiryHours)
	transferHandler := delivery.NewTransferHandler(transferUC)
	walletHandler := delivery.NewWalletHandler(walletUC)

	r := gin.Default()

	api := r.Group("/api/v1")
	{
		auth := api.Group("/auth")
		{
			auth.POST("/register", authHandler.Register)
			auth.POST("/login", authHandler.Login)
		}

		api.Use(middleware.JWTAuth(jwtSecret))
		{
			api.POST("/transfer", transferHandler.Transfer)
			api.POST("/topup", walletHandler.TopUp)
			api.GET("/transactions", walletHandler.GetTransactionHistory)
		}
	}

	r.GET("/ping", middleware.JWTAuth(jwtSecret), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "pong",
		})
	})

	log.Printf("server starting on port %s", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}
