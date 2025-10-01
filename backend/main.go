package main

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/conan-flynn/cronnect/auth"
	"github.com/conan-flynn/cronnect/database"
	"github.com/conan-flynn/cronnect/models"
	"github.com/conan-flynn/cronnect/queue"
	"github.com/conan-flynn/cronnect/routes"
	"github.com/conan-flynn/cronnect/scheduler"
	"github.com/conan-flynn/cronnect/worker"
	"github.com/joho/godotenv"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var db *gorm.DB

func main() {
	godotenv.Load()
	
	auth.InitOAuth()
	
	sessionSecret := os.Getenv("SESSION_SECRET")
	if sessionSecret == "" {
		log.Fatal("SESSION_SECRET environment variable is required")
	}
	
	dsn := fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=%s sslmode=%s TimeZone=%s",
		getEnv("DATABASE_HOST", "localhost"),
		getEnv("DATABASE_USER", "cronnect"),
		getEnv("DATABASE_PASSWORD", "password"),
		getEnv("DATABASE_NAME", "cronnect"),
		getEnv("DATABASE_PORT", "5432"),
		getEnv("DATABASE_SSLMODE", "disable"),
		getEnv("DATABASE_TIMEZONE", "UTC"),
	)
	var err error
	for attempt := 1; attempt <= 30; attempt++ {
		db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
		if err == nil {
			break
		}
		log.Printf("Database not ready (attempt %d/30): %v", attempt, err)
		time.Sleep(1 * time.Second)
	}
	if err != nil {
		panic("failed to connect to database")
	}
	database.DB = db
	db.AutoMigrate(&models.User{}, &models.Job{}, &models.JobExecution{})

	database.ConnectRedis()

	go scheduler.StartScheduler()

	queueService := queue.NewQueueService()
	go queueService.ProcessRetryQueue()

	workerCount := getWorkerCount()
	log.Printf("Starting %d workers", workerCount)
	worker.StartMultipleWorkers(workerCount)

	router := routes.SetupRoutes(db)
	
	router.Run("0.0.0.0:8080")
}

func getWorkerCount() int {
	workerCountStr := os.Getenv("WORKER_COUNT")
	if workerCountStr == "" {
		return 3
	}
	
	count, err := strconv.Atoi(workerCountStr)
	if err != nil {
		log.Printf("Invalid WORKER_COUNT value: %s, using default of 3", workerCountStr)
		return 3
	}
	
	if count < 1 {
		return 1
	}
	
	return count
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

