package main

import (
	"encoding/json"
	"fmt"
	"os"
)

type Permission int

const (
	PERM_NILL Permission = 1 << iota
	// Fetch Discord Token
	PERM_DISCORD_TOKEN
	// Identify a payer
	PERM_PAYER_IDENTIFY
	// Full perms
	PERM_ADMIN
	// Live Notifications
	PERM_LIVE_NOTIFICATION
	// Admin fund control
	PERM_FUND_CONTROL
	// Fetch from /donations
	PERM_FETCH_DONATIONS
)

type App struct {
	Name  string     `json:"name"`
	Perms Permission `json:"permissions"`
	Token string     `json:"token"`
}

var Apps = map[string]*App{}

func InitAuths() {
	f, err := os.ReadFile("auths.json")
	PanicIfErr(err)

	raws := []*App{}

	names := map[string]bool{}
	tokens := map[string]bool{}

	PanicIfErr(json.Unmarshal(f, &raws))

	largestName := 0
	for _, app := range raws {
		if len(app.Name) > largestName {
			largestName = len(app.Name)
		}
		if names[app.Name] {
			logger.Logf(LL_PANIC, "'%v' name is not unique!", app.Name)
		}
		if tokens[app.Token] {
			logger.Logf(LL_PANIC, "'%v' token is not unique!", app.Token)
		}
		names[app.Name] = true
		names[app.Token] = true
	}

	if largestName < 9 {
		largestName = 9
	}

	largestName += 3

	header := ">"

	for i := 0; i < largestName+10; i++ {
		header += "="
	}

	header += "<"

	apps := ""

	for _, app := range raws {
		name := app.Name

		Apps[app.Token] = app

		for len(name) < largestName {
			name += " "
		}
		name += fmt.Sprint(app.Perms)
		name = "      " + name

		apps += name + "\n"
	}

	logger.Logf(LL_DEBUG, "Applications loaded\n"+header+"\n\n"+apps+"\n"+header)
}
