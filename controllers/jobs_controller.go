package controllers

import (
	"net/http"

	"github.com/conan-flynn/cronnect/models"
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
	newJob.ID = uuid.NewString()
	jc.DB.Create(&newJob)
	c.IndentedJSON(http.StatusCreated, newJob)
}
