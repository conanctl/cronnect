package main

import (
	"fmt"
	"log"
	"os"

	"github.com/conan-flynn/cronnect/database"
	"github.com/conan-flynn/cronnect/routes"
	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables")
	}

	db := database.Connect()
	router := routes.SetupRoutes(db)

	router.Run(fmt.Sprintf("%s:%s", os.Getenv("APP_HOST"), os.Getenv("APP_PORT")))
}
