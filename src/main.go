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
	recoverMiddleware "github.com/gofiber/fiber/v2/middleware/recover" // ✅ Renamed to prevent conflicts
	"github.com/joho/godotenv"
)

func main() {
	// ✅ Properly handling panics with recover()
	defer func() {
		if r := recover(); r != nil {
			log.Fatalf("🚨 Application crashed due to panic: %v", r)
		}
	}()

	// Load environment variables in development mode
	if globals.DevMode {
		godotenv.Load()
	}

	// Log environment variables
	log.Println("🔍 ENVIRONMENT VARIABLES:")
	for _, e := range os.Environ() {
		log.Println(e)
	}

	// Log current working directory
	cwd, _ := os.Getwd()
	log.Println("🔍 Current Working Directory:", cwd)

	// ✅ Ensure correct port configuration
	port := os.Getenv("PORT")
	if port == "" {
		port = "10000" // ✅ Match Render's port
	}
	log.Printf("🚀 Starting server on port %s...\n", port)

	// ✅ Force Prefork Mode
	usePrefork := true
	if os.Getenv("DISABLE_PREFORK") == "true" {
		usePrefork = false
	}
	log.Printf("⚡ Prefork mode enabled: %v\n", usePrefork)

	// ✅ Initialize Fiber with Prefork
	app := fiber.New(fiber.Config{
		Prefork: usePrefork,
		ServerHeader: "GoScraper",
		AppName: "GoScraper v3.0",
		JSONEncoder: json.Marshal,
		JSONDecoder: json.Unmarshal,
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			return utils.HandleError(c, err)
		},
	})

	// ✅ Ensure Child Processes Load Environment Variables
	if fiber.IsChild() {
		log.Println("🔹 Prefork Child Process: Initializing resources")
		if globals.DevMode {
			godotenv.Load() // ✅ Reload environment variables for child processes
		}
		time.Sleep(2 * time.Second) // ✅ Prevents race conditions
		log.Println("🔹 Prefork Child Process Started Successfully")
	}

	// ✅ Force 8 Prefork processes
	os.Setenv("GOMAXPROCS", "8")

	// ✅ Use the renamed recover middleware
	app.Use(recoverMiddleware.New())
	app.Use(compress.New(compress.Config{Level: compress.LevelBestSpeed}))
	app.Use(etag.New())

	// Health check endpoint
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

	// Start the server and log if it crashes
	if err := app.Listen("0.0.0.0:" + port); err != nil {
		log.Fatalf("🚨 Server crashed with error: %v", err)
	}
}
