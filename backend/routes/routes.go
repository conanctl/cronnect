package routes

import (
	"net/http"
	"os"

	"github.com/conan-flynn/cronnect/auth"
	"github.com/conan-flynn/cronnect/controllers"
	"github.com/conan-flynn/cronnect/middleware"
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func SetupRoutes(db *gorm.DB) *gin.Engine {
	router := gin.Default()
	
	sessionSecret := os.Getenv("SESSION_SECRET")
	store := cookie.NewStore([]byte(sessionSecret))
	store.Options(sessions.Options{
		Path:     "/",
		MaxAge:   86400 * 7,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	})
	router.Use(sessions.Sessions("cronnect_session", store))
	
	router.GET("/auth/google", auth.GoogleLogin)
	router.GET("/auth/google/callback", auth.GoogleCallback)
	router.GET("/auth/github", auth.GithubLogin)
	router.GET("/auth/github/callback", auth.GithubCallback)
	router.GET("/auth/logout", auth.Logout)
	router.GET("/auth/user", auth.GetCurrentUser)
	
	jobController := controllers.NewJobController(db)
	
	protected := router.Group("/")
	protected.Use(middleware.AuthRequired())
	{
		protected.StaticFile("/", "/app/frontend/index.html")
		protected.GET("/jobs", jobController.GetJobs)
		protected.POST("/jobs", jobController.CreateJob)
		protected.DELETE("/jobs/:id", jobController.DeleteJob)
	}

	return router
}