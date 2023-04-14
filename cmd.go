package main

import (
	"os"
	"strings"
	"time"

	"github.com/chzyer/readline"
	"github.com/shadiestgoat/donations/auth"
	"github.com/shadiestgoat/log"
)

func processCMD(closer chan bool) {
	home, _ := os.UserHomeDir()
	
	if home == "" {
		home = "/tmp"
	}

	l, err := readline.NewEx(&readline.Config{
		Prompt:            "",
		HistoryFile:       home + "/.donationAPI_history",
		HistorySearchFold: true,
		InterruptPrompt:   "",
	})

	log.FatalIfErr(err, "making new readline reader")

	go func ()  {
		<- closer
		l.Close()
	}()

	for {
		line, err := l.Readline()
		if err != nil {
			break
		}

		line = strings.TrimSpace(line)

		if line == "" {
			continue
		}

		words := strings.Split(line, " ")

		switch {
		case words[0] == "reload":
			log.Debug("Reloading the auth.json file...")
			time.Sleep(50 * time.Millisecond)
			auth.Load()
		case words[0] == "fund":
			// TODO:
		default:
		}
	}
}