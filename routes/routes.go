package routes

import (
	"github.com/conan-flynn/cronnect/controllers"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func SetupRoutes(db *gorm.DB) *gin.Engine {
	router := gin.Default()
	jobController := controllers.NewJobController(db)

	router.GET("/jobs", jobController.GetJobs)
	router.POST("/jobs", jobController.CreateJob)

	return router
}
