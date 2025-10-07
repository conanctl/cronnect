package controllers

import (
	"net/http"

	"github.com/robfig/cron/v3"
	"github.com/conan-flynn/cronnect/middleware"
	"github.com/conan-flynn/cronnect/models"
	"github.com/conan-flynn/cronnect/scheduler"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type JobController struct {
	DB *gorm.DB
}

func NewJobController(db *gorm.DB) *JobController {
	return &JobController{DB: db}
}

func (jc *JobController) GetJobs(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not authenticated"})
		return
	}

	var jobs []models.Job
	jc.DB.Where("user_id = ?", userID).Preload("Executions").Find(&jobs)
	c.IndentedJSON(http.StatusOK, jobs)
}

func (jc *JobController) GetJob(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not authenticated"})
		return
	}

	jobID := c.Param("id")
	
	var job models.Job
	if err := jc.DB.Where("id = ? AND user_id = ?", jobID, userID).Preload("Executions").First(&job).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "job not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve job"})
		}
		return
	}
	
	c.IndentedJSON(http.StatusOK, job)
}

func (jc *JobController) CreateJob(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not authenticated"})
		return
	}

	var newJob models.Job
	if err := c.BindJSON(&newJob); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid job data"})
		return
	}

	if _, err := cron.ParseStandard(newJob.Schedule); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid cron schedule"})
		return
	}
	newJob.ID = uuid.NewString()
	newJob.UserID = userID.(string)

	if err := jc.DB.Create(&newJob).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create job"})
		return
	}
	
	scheduler.ReloadJobs()
	
	c.IndentedJSON(http.StatusCreated, newJob)
}

func (jc *JobController) UpdateJob(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not authenticated"})
		return
	}

	jobID := c.Param("id")
	
	var existingJob models.Job
	if err := jc.DB.Where("id = ? AND user_id = ?", jobID, userID).First(&existingJob).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "job not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve job"})
		}
		return
	}

	var updateData models.Job
	if err := c.BindJSON(&updateData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid job data"})
		return
	}

	if updateData.Schedule != "" {
		if _, err := cron.ParseStandard(updateData.Schedule); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid cron schedule"})
			return
		}
	}

	updateData.UserID = existingJob.UserID
	updateData.ID = existingJob.ID

	if err := jc.DB.Model(&existingJob).Updates(updateData).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update job"})
		return
	}
	
	scheduler.ReloadJobs()
	
	c.IndentedJSON(http.StatusOK, existingJob)
}

func (jc *JobController) DeleteJob(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not authenticated"})
		return
	}

	jobID := c.Param("id")
	
	var job models.Job
	if err := jc.DB.Where("id = ? AND user_id = ?", jobID, userID).First(&job).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "job not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve job"})
		}
		return
	}
	
	if err := jc.DB.Delete(&job).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete job"})
		return
	}
	
	scheduler.ReloadJobs()
	
	c.JSON(http.StatusOK, gin.H{"message": "job deleted successfully"})
}

func (jc *JobController) GetRateLimit(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not authenticated"})
		return
	}

	rateLimiter := middleware.NewRateLimiter()
	used, remaining, limit, resetAt, err := rateLimiter.GetRateLimitStatus(userID.(string))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get rate limit status"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"used":      used,
		"remaining": remaining,
		"limit":     limit,
		"reset_at":  resetAt.Format("2006-01-02T15:04:05Z07:00"),
		"window":    "1 hour",
	})
}
