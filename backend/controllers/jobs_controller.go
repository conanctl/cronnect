package controllers

import (
	"net/http"

	"github.com/robfig/cron/v3"
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
	var jobs []models.Job
	jc.DB.Preload("Executions").Find(&jobs)
	c.IndentedJSON(http.StatusOK, jobs)
}

func (jc *JobController) CreateJob(c *gin.Context) {
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
	jc.DB.Create(&newJob)
	
	scheduler.ReloadJobs()
	
	c.IndentedJSON(http.StatusCreated, newJob)
}

func (jc *JobController) DeleteJob(c *gin.Context) {
	jobID := c.Param("id")
	
	var job models.Job
	if err := jc.DB.First(&job, "id = ?", jobID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "job not found"})
		return
	}
	
	if err := jc.DB.Delete(&job).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete job"})
		return
	}
	
	scheduler.ReloadJobs()
	
	c.JSON(http.StatusOK, gin.H{"message": "job deleted successfully"})
}
