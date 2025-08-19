package main

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type Job struct {
	ID         string         `gorm:"primaryKey" json:"id"`
	Name       string         `gorm:"size:100;not null" json:"name"`
	URL        string         `gorm:"not null" json:"url"`
	Method     string         `gorm:"size:10;default:GET" json:"method"`
	Schedule   string         `gorm:"size:100;not null" json:"schedule"`
	Status     string         `gorm:"size:20;default:active" json:"status"`
	Executions []JobExecution `gorm:"foreignKey:JobID;constraint:OnDelete:CASCADE" json:"executions,omitempty"`
}

type JobExecution struct {
	ID           string     `gorm:"primaryKey" json:"id"`
	JobID        string     `gorm:"index;not null" json:"job_id"`
	StartedAt    time.Time  `gorm:"autoCreateTime" json:"started_at"`
	FinishedAt   *time.Time `json:"finished_at,omitempty"`
	Status       string     `gorm:"size:20;not null" json:"status"`
	ResponseCode int        `json:"response_code"`
}

var db *gorm.DB

func main() {
	dsn := "host=localhost user=youruser password=yourpass dbname=yourdb port=5432 sslmode=disable TimeZone=UTC"
	var err error
	db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		panic("failed to connect to database")
	}

	db.AutoMigrate(&Job{}, &JobExecution{})

	router := gin.Default()
	router.GET("/jobs", getJobs)
	router.POST("/jobs", createJob)
	router.Run("localhost:8080")
}

func getJobs(c *gin.Context) {
	var jobs []Job
	db.Preload("Executions").Find(&jobs)
	c.IndentedJSON(http.StatusOK, jobs)
}

func createJob(c *gin.Context) {
	var newJob Job
	if err := c.BindJSON(&newJob); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid job data"})
		return
	}
	newJob.ID = uuid.NewString()
	db.Create(&newJob)
	c.IndentedJSON(http.StatusCreated, newJob)
}
