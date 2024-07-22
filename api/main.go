package main

import (
	"fmt"
	"log"
	"os"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/joho/godotenv"
	"github.com/snagglebrew/shorten-url/routes"
)

func setupRoutes(app *fiber.App) {
	app.Get("/:url", routes.ResolveURL)
	app.Post("/api/v1", routes.ShortenURL)
	app.Post("/api/v1/register", routes.Register)
	app.Post("/api/v1/login", routes.Login)
}

func main() {
	err := godotenv.Load()
	if err != nil {
		fmt.Println(err)
	}

	app := fiber.New()
	app.Use(logger.New())
	setupRoutes(app)
	//database.Initialize(0) // Initialize the test database
	log.Fatal(app.Listen(os.Getenv("APP_PORT")))
}
