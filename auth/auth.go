package auth

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sync"

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

var appLock = &sync.RWMutex{}

func FetchApp(token string) *App {
	appLock.RLock()
	defer appLock.RUnlock()

	return Apps[token]
}

// Returns true if app with 'token' has this permission
func HasPerm(token string, perm Permission) bool {
	appLock.RLock()
	defer appLock.RUnlock()

	if Apps[token] == nil {
		return false
	}

	return perm&Apps[token].Perms == perm || Apps[token].Perms == PERM_ADMIN
}

func Load() {
	appLock.Lock()
	defer appLock.Unlock()

	Apps = map[string]*App{}

	f, err := os.ReadFile("auths.json")
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			log.FatalIfErr(err, "opening 'auths.json'")
		}
		log.Warn("Couldn't find the 'auths.json' file, are you sure it exists?")
		return
	}

	raws := []*App{}

	names := map[string]bool{}
	tokens := map[string]bool{}

	err = json.Unmarshal(f, &raws)
	if log.ErrorIfErr(err, "parsing 'auths.json'") {
		log.Warn("Couldn't parse 'auths.json', are you sure it is valid")
	}

	largestName := 0

	for _, app := range raws {
		if len(app.Name) > largestName {
			largestName = len(app.Name)
		}

		if names[app.Name] {
			log.Error("'%s' auth name is not unique!", app.Name)
		}
		if tokens[app.Token] {
			log.Error("'%s' token is not unique!", app.Token)
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
