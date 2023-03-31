package main

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
)

var (
	app *fiber.App = fiber.New(fiber.Config{
		DisableStartupMessage: true,
		ErrorHandler: func(ctx *fiber.Ctx, err error) error {
			log.Printf("Error: %v - URI: %v\n", err, ctx.Request().URI())

			return ctx.SendStatus(http.StatusInternalServerError)
		},
	})
	r    *Redis  = &Redis{}
	conf *Config = DefaultConfig
)

func init() {
	if err := conf.ReadFile("config.yml"); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			log.Printf("config.yml does not exist, writing default config\n")

			if err = conf.WriteFile("config.yml"); err != nil {
				log.Fatalf("Failed to write config file: %v", err)
			}
		} else {
			log.Printf("Failed to read config file: %v", err)
		}
	}

	if err := GetBlockedServerList(); err != nil {
		log.Fatalf("Failed to retrieve EULA blocked servers: %v", err)
	}

	log.Println("Successfully retrieved EULA blocked servers")

	if conf.Redis != nil {
		if err := r.Connect(); err != nil {
			log.Fatalf("Failed to connect to Redis: %v", err)
		}

		log.Println("Successfully connected to Redis")
	}

	app.Use(recover.New(recover.Config{
		EnableStackTrace: true,
	}))

	if conf.Environment == "development" {
		app.Use(cors.New(cors.Config{
			AllowOrigins:  "*",
			AllowMethods:  "HEAD,OPTIONS,GET",
			ExposeHeaders: "X-Cache-Hit,X-Cache-Time-Remaining",
		}))

		app.Use(logger.New(logger.Config{
			Format:     "${time} ${ip}:${port} -> ${status}: ${method} ${path} (${latency})\n",
			TimeFormat: "2006/01/02 15:04:05",
		}))
	}
}

func main() {
	defer r.Close()

	log.Printf("Listening on %s:%d\n", conf.Host, conf.Port)

	if err := app.Listen(fmt.Sprintf("%s:%d", conf.Host, conf.Port)); err != nil {
		log.Fatalf("failed to start server: %v", err)
	}
}
