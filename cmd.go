package main

import (
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/chzyer/readline"
	"github.com/shadiestgoat/donations/auth"
	"github.com/shadiestgoat/log"
)

func printState(s int) {
	switch s {
	case 1:
		log.PrintSuccess("[1/5]: Should this fund be default? [Yes]/No")
	case 2:
		log.PrintSuccess("[2/5]: Should this have a goal, and if so, what? [Default = no goal]")
	case 3:
		log.PrintSuccess("[3/5]: What is the Short Title for this fund? (Format: Donate for {ShortTitle})?")
	case 4:
		log.PrintSuccess("[4/5]: What is the Title for this fund?")
	case 5:
		log.PrintSuccess("[5/5]: What is an Alias for this fund? (Will be used for short urls, /f/{Alias})")
	}
}

func processCMD(closer chan bool) {
	home, _ := os.UserHomeDir()

	if home == "" {
		home = "/tmp"
	}

	l, err := readline.NewEx(&readline.Config{
		Prompt:            "",
		HistoryFile:       home + "/.donationAPI_history",
		HistorySearchFold: true,
		InterruptPrompt:   "\n",
	})

	log.FatalIfErr(err, "making new readline reader")

	go func() {
		<-closer
		l.Close()
	}()

	var newFund *Fund

	fundState := 0

	for {
		line, err := l.Readline()
		if err != nil {
			break
		}

		line = strings.TrimSpace(line)

		if newFund != nil {
			switch fundState {
			case 1:
				v := true

				switch strings.ToLower(line) {
				case "yes", "y", "t", "true", "1":
					v = true
				case "no", "n", "f", "false", "0":
					v = false
				}

				newFund.Default = &v
				fundState++
			case 2:
				f, err := strconv.ParseFloat(line, 64)

				if (err == nil || line == "") && f >= 0 {
					fundState++
					newFund.Goal = f
				} else {
					log.PrintWarn("Can't parse this! Please input a number")
				}
			case 3:
				if len(line) < 3 {
					log.Warn("The title must be at least 3 in length!")
				} else {
					fundState++
					newFund.ShortTitle = line
				}
			case 4:
				if len(line) < 3 {
					log.Warn("The title must be at least 3 in length!")
				} else {
					fundState++
					newFund.Title = line
				}
			case 5:
				if len(line) < 3 {
					log.Warn("The alias must be at least 3 in length!")
				} else {
					fundState++
					newFund.Alias = line
				}
			}

			if fundState == 6 {
				NewFund(newFund)
				fundState = 0
				newFund = nil
			} else {
				printState(fundState)
			}
		} else {
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
				log.PrintSuccess("Alright, lets make a new fund! Remember: you can type 'cancel' any time to cancel the fund creation!")
				newFund = &Fund{}

				fundState = 1
				printState(fundState)
			default:
			}
		}

	}
}
