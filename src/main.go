package main

import (
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"

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
	r          *Redis  = &Redis{}
	conf       *Config = DefaultConfig
	instanceID uint16  = 0
)

func init() {
	var err error

	if err = conf.ReadFile("config.yml"); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			log.Printf("config.yml does not exist, writing default config\n")

			if err = conf.WriteFile("config.yml"); err != nil {
				log.Fatalf("Failed to write config file: %v", err)
			}
		} else {
			log.Printf("Failed to read config file: %v", err)
		}
	}

	if err = GetBlockedServerList(); err != nil {
		log.Fatalf("Failed to retrieve EULA blocked servers: %v", err)
	}

	log.Println("Successfully retrieved EULA blocked servers")

	if conf.Redis != nil {
		if err = r.Connect(); err != nil {
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

	if instanceID, err = GetInstanceID(); err != nil {
		panic(err)
	}
}

func main() {
	if v := os.Getenv("PROFILE"); len(v) > 0 {
		port, err := strconv.ParseUint(v, 10, 16)

		if err != nil {
			panic(err)
		}

		go Profile(uint16(port))

		log.Printf("Profiler is listening on :%d\n", port)
	}

	defer r.Close()

	go ListenAndServe(conf.Host, conf.Port+instanceID)

	defer app.Shutdown()

	s := make(chan os.Signal, 1)
	signal.Notify(s, os.Interrupt, syscall.SIGTERM)
	<-s
}
