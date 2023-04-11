package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/shadiestgoat/log"
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
	log.FatalIfErr(err, "opening 'auths.json'")

	raws := []*App{}

	names := map[string]bool{}
	tokens := map[string]bool{}

	log.FatalIfErr(json.Unmarshal(f, &raws), "parsing 'auths.json'")

	largestName := 0
	for _, app := range raws {
		if len(app.Name) > largestName {
			largestName = len(app.Name)
		}

		if names[app.Name] {
			log.Fatal("'%s' auth name is not unique!", app.Name)
		}
		if tokens[app.Token] {
			log.Fatal("'%s' token is not unique!", app.Token)
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

	log.Success("Applications loaded\n%s\n\n%s\n%s", header, apps, header)
}
