package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/shadiestgoat/donations/auth"
	"github.com/shadiestgoat/donations/db"
	"github.com/shadiestgoat/log"
)

func main() {
	InitConfig()
	os.Setenv("TZ", "UTC")

	cbs := []log.LogCB{
		log.NewLoggerPrint(), log.NewLoggerFile("logs/log"),
	}

	if DEBUG_DISC_WEBHOOK != "" {
		mention := DEBUG_DISC_MENTION
		if mention != "" {
			mention += ", "
		}

		cbs = append(cbs, log.NewLoggerDiscordWebhook(mention, DEBUG_DISC_WEBHOOK))
	}

	log.Init(cbs...)

	log.Success("Config & Log loaded!")

	auth.Load()

	log.Success("Snowflake loaded!")

	InitFrontend()

	log.Success("Frontend loaded!")

	// db.Init(DB_URI)
	log.Success("Database connected!")

	defer db.Close()

	r := RouterBase()
	r.Mount(`/api`, RouterAPI())

	log.Success("Server started!")

	server := &http.Server{
		Addr:    ":" + PORT,
		Handler: r,
	}

	go func() {
		err := server.ListenAndServe()
		if !errors.Is(err, http.ErrServerClosed) {
			log.Fatal("HTTP server shut down: %s", err.Error())
		}
	}()

	stopper := make(chan os.Signal, 2)
	signal.Notify(stopper, os.Interrupt)

	log.Success("Everything is ready! You can now press Ctrl+C to shut down <3")
	log.PrintSuccess("Also - you can now do 'reload' to reload the 'auths.json' file!")

	readLineStopper := make(chan bool)
	go processCMD(readLineStopper)

	<-stopper

	readLineStopper <- true

	log.Warn("Shutting down :(")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)

	server.Shutdown(ctx)

	cancel()
}
