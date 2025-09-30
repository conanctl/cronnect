package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"os"

	"github.com/conan-flynn/cronnect/database"
	"github.com/conan-flynn/cronnect/models"
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/github"
	"golang.org/x/oauth2/google"
)

var (
	googleOauthConfig *oauth2.Config
	githubOauthConfig *oauth2.Config
)

func InitOAuth() {
	googleOauthConfig = &oauth2.Config{
		ClientID:     os.Getenv("GOOGLE_CLIENT_ID"),
		ClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
		RedirectURL:  os.Getenv("GOOGLE_REDIRECT_URL"),
		Scopes:       []string{"https://www.googleapis.com/auth/userinfo.email", "https://www.googleapis.com/auth/userinfo.profile"},
		Endpoint:     google.Endpoint,
	}

	githubOauthConfig = &oauth2.Config{
		ClientID:     os.Getenv("GITHUB_CLIENT_ID"),
		ClientSecret: os.Getenv("GITHUB_CLIENT_SECRET"),
		RedirectURL:  os.Getenv("GITHUB_REDIRECT_URL"),
		Scopes:       []string{"user:email"},
		Endpoint:     github.Endpoint,
	}
}

func generateStateToken() string {
	b := make([]byte, 32)
	rand.Read(b)
	return base64.URLEncoding.EncodeToString(b)
}

func GoogleLogin(c *gin.Context) {
	state := generateStateToken()
	session := sessions.Default(c)
	session.Set("oauth_state", state)
	session.Save()

	url := googleOauthConfig.AuthCodeURL(state, oauth2.AccessTypeOffline)
	c.Redirect(http.StatusTemporaryRedirect, url)
}

func GoogleCallback(c *gin.Context) {
	session := sessions.Default(c)
	savedState := session.Get("oauth_state")
	
	if savedState == nil || savedState != c.Query("state") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid state parameter"})
		return
	}

	code := c.Query("code")
	token, err := googleOauthConfig.Exchange(context.Background(), code)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to exchange token"})
		return
	}

	client := googleOauthConfig.Client(context.Background(), token)
	resp, err := client.Get("https://www.googleapis.com/oauth2/v2/userinfo")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get user info"})
		return
	}
	defer resp.Body.Close()

	data, _ := io.ReadAll(resp.Body)
	var userInfo map[string]interface{}
	json.Unmarshal(data, &userInfo)

	email := userInfo["email"].(string)
	name := userInfo["name"].(string)

	user := saveOrUpdateUser(email, name, "google")
	
	session.Set("user_id", user.ID)
	session.Set("user_email", user.Email)
	session.Set("user_name", user.Name)
	session.Save()

	c.Redirect(http.StatusTemporaryRedirect, "/")
}

func GithubLogin(c *gin.Context) {
	state := generateStateToken()
	session := sessions.Default(c)
	session.Set("oauth_state", state)
	session.Save()

	url := githubOauthConfig.AuthCodeURL(state)
	c.Redirect(http.StatusTemporaryRedirect, url)
}

func GithubCallback(c *gin.Context) {
	session := sessions.Default(c)
	savedState := session.Get("oauth_state")
	
	if savedState == nil || savedState != c.Query("state") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid state parameter"})
		return
	}

	code := c.Query("code")
	token, err := githubOauthConfig.Exchange(context.Background(), code)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to exchange token"})
		return
	}

	client := githubOauthConfig.Client(context.Background(), token)
	
	resp, err := client.Get("https://api.github.com/user")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get user info"})
		return
	}
	defer resp.Body.Close()

	data, _ := io.ReadAll(resp.Body)
	var userInfo map[string]interface{}
	json.Unmarshal(data, &userInfo)

	emailResp, _ := client.Get("https://api.github.com/user/emails")
	emailData, _ := io.ReadAll(emailResp.Body)
	emailResp.Body.Close()
	
	var emails []map[string]interface{}
	json.Unmarshal(emailData, &emails)
	
	var email string
	for _, e := range emails {
		if primary, ok := e["primary"].(bool); ok && primary {
			email = e["email"].(string)
			break
		}
	}
	
	if email == "" && len(emails) > 0 {
		email = emails[0]["email"].(string)
	}

	name := ""
	if userInfo["name"] != nil {
		name = userInfo["name"].(string)
	} else if userInfo["login"] != nil {
		name = userInfo["login"].(string)
	}

	user := saveOrUpdateUser(email, name, "github")
	
	session.Set("user_id", user.ID)
	session.Set("user_email", user.Email)
	session.Set("user_name", user.Name)
	session.Save()

	c.Redirect(http.StatusTemporaryRedirect, "/")
}

func saveOrUpdateUser(email, name, provider string) models.User {
	var user models.User
	
	result := database.DB.Where("email = ?", email).First(&user)
	
	if result.Error != nil {
		user = models.User{
			ID:       uuid.NewString(),
			Email:    email,
			Name:     name,
			Provider: provider,
		}
		database.DB.Create(&user)
	} else {
		user.Name = name
		user.Provider = provider
		database.DB.Save(&user)
	}
	
	return user
}

func Logout(c *gin.Context) {
	session := sessions.Default(c)
	session.Clear()
	session.Save()
	
	c.Redirect(http.StatusTemporaryRedirect, "/")
}

func GetCurrentUser(c *gin.Context) {
	session := sessions.Default(c)
	userID := session.Get("user_id")
	
	if userID == nil {
		c.JSON(http.StatusOK, gin.H{"authenticated": false})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"authenticated": true,
		"user_id":       session.Get("user_id"),
		"email":         session.Get("user_email"),
		"name":          session.Get("user_name"),
	})
}
