package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/conan-flynn/cronnect/database"
	"github.com/conan-flynn/cronnect/models"
	"github.com/conan-flynn/cronnect/queue"
	"github.com/conan-flynn/cronnect/scheduler"
	"github.com/conan-flynn/cronnect/worker"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var db *gorm.DB

func main() {
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
	db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		panic("failed to connect to database")
	}
	database.DB = db
	db.AutoMigrate(&models.Job{}, &models.JobExecution{})

	database.ConnectRedis()

	go scheduler.StartScheduler()

	queueService := queue.NewQueueService()
	go queueService.ProcessRetryQueue()

	workerCount := getWorkerCount()
	log.Printf("Starting %d workers", workerCount)
	worker.StartMultipleWorkers(workerCount)

	router := gin.Default()
	router.StaticFile("/", "./index.html")
	router.GET("/jobs", getJobs)
	router.POST("/jobs", createJob)
	router.Run("localhost:8080")
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

func getJobs(c *gin.Context) {
	var jobs []models.Job
	database.DB.Preload("Executions").Find(&jobs)
	c.IndentedJSON(http.StatusOK, jobs)
}

func createJob(c *gin.Context) {
	var newJob models.Job
	if err := c.BindJSON(&newJob); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid job data"})
		return
	}
	newJob.ID = uuid.NewString()
	database.DB.Create(&newJob)
	c.IndentedJSON(http.StatusCreated, newJob)
}
