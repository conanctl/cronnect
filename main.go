package main

import (
	"net/http"

	"github.com/conan-flynn/cronnect/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var db *gorm.DB

func main() {
	dsn := "host=localhost user=youruser password=yourpass dbname=yourdb port=5432 sslmode=disable TimeZone=UTC"
	var err error
	db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		panic("failed to connect to database")
	}

	db.AutoMigrate(&models.Job{}, &models.JobExecution{})

	router := gin.Default()
	router.GET("/jobs", getJobs)
	router.POST("/jobs", createJob)
	router.Run("localhost:8080")
}

func getJobs(c *gin.Context) {
	var jobs []models.Job
	db.Preload("Executions").Find(&jobs)
	c.IndentedJSON(http.StatusOK, jobs)
}

func createJob(c *gin.Context) {
	var newJob models.Job
	if err := c.BindJSON(&newJob); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid job data"})
		return
	}
	newJob.ID = uuid.NewString()
	db.Create(&newJob)
	c.IndentedJSON(http.StatusCreated, newJob)
}
