package main

import (
	"encoding/json"
	"log"
	"os"
	"strings"
	"time"

	"goscraper/src/globals"
	"goscraper/src/handlers"
	"goscraper/src/utils"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/compress"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/etag"
	"github.com/gofiber/fiber/v2/middleware/limiter"
	recoverMiddleware "github.com/gofiber/fiber/v2/middleware/recover" // ✅ Renamed import
	"github.com/joho/godotenv"
)

func main() {
	// Properly using defer with recover to catch panics
	defer func() {
		if r := recover(); r != nil {
			log.Fatalf("🚨 Application crashed due to panic: %v", r)
		}
	}()

	// Load environment variables in development mode
	if globals.DevMode {
		godotenv.Load()
	}

	// Log environment variables to check if anything is missing
	log.Println("🔍 ENVIRONMENT VARIABLES:")
	for _, e := range os.Environ() {
		log.Println(e)
	}

	// Log current working directory
	cwd, _ := os.Getwd()
	log.Println("🔍 Current Working Directory:", cwd)

	// Ensure correct port configuration
	port := os.Getenv("PORT")
	if port == "" {
		port = "8000" // Default to Koyeb's expected port
	}
	log.Printf("🚀 Starting server on port %s...\n", port)

	// Ensure Prefork is disabled (fixing previous issues)
	prefork := os.Getenv("PREFORK")
	usePrefork := false
	if prefork == "true" || prefork == "1" {
		usePrefork = true
	}

	// Initialize Fiber
	app := fiber.New(fiber.Config{
		Prefork:      usePrefork,
		ServerHeader: "GoScraper",
		AppName:      "GoScraper v3.0",
		JSONEncoder:  json.Marshal,
		JSONDecoder:  json.Unmarshal,
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			return utils.HandleError(c, err)
		},
	})

	// ✅ Use the renamed recover middleware
	app.Use(recoverMiddleware.New())
	app.Use(compress.New(compress.Config{Level: compress.LevelBestSpeed}))
	app.Use(etag.New())

	// Health check endpoint (prevents Koyeb from stopping the app)
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})

	// CORS Configuration
	urls := os.Getenv("URL")
	allowedOrigins := "http://localhost:243"
	if urls != "" {
		allowedOrigins += "," + urls
	}

	app.Use(cors.New(cors.Config{
		AllowOrigins:     allowedOrigins,
		AllowMethods:     "GET,POST,PUT,DELETE,OPTIONS",
		AllowHeaders:     "Origin,Content-Type,Accept,X-CSRF-Token,Authorization",
		ExposeHeaders:    "Content-Length",
		AllowCredentials: true,
	}))

	// Rate Limiting
	app.Use(limiter.New(limiter.Config{
		Max:        25,
		Expiration: 1 * time.Minute,
		KeyGenerator: func(c *fiber.Ctx) string {
			token := c.Get("X-CSRF-Token")
			if token != "" {
				return utils.Encode(token)
			}
			return c.IP()
		},
		LimitReached: func(c *fiber.Ctx) error {
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"error": "🔨 Rate limit exceeded. Please try again later.",
			})
		},
		SkipFailedRequests: false,
		LimiterMiddleware:  limiter.SlidingWindow{},
	}))

	// Authentication Middleware
	app.Use(func(c *fiber.Ctx) error {
		token := c.Get("Authorization")
		if token == "" || (!strings.HasPrefix(token, "Bearer ") && !strings.HasPrefix(token, "Token ")) {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Missing Authorization header",
			})
		}
		return c.Next()
	})

	// Error Handling Middleware
	app.Use(func(c *fiber.Ctx) error {
		err := c.Next()
		if err != nil {
			log.Printf("⚠️ Fiber Error: %v", err)
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": err.Error(),
			})
		}
		return nil
	})

	// Routes -----------------------------------------
	app.Get("/", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"message": "GoScraper is running!"})
	})

	app.Get("/user", func(c *fiber.Ctx) error {
		user, err := handlers.GetUser(c.Get("X-CSRF-Token"))
		if err != nil {
			return err
		}
		return c.JSON(user)
	})

	// Start the server and log if it crashes
	if err := app.Listen("0.0.0.0:" + port); err != nil {
		log.Fatalf("🚨 Server crashed with error: %v", err)
	}
}
