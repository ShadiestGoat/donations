package main

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

type confItem struct {
	Res      *string
	Default  string
	Required bool
}

var (
	DB_URI             = ""
	PORT               = ""
	
	DEBUG_DISC_WEBHOOK = ""
	DEBUG_DISC_MENTION = ""

	PAYPAL_PATH        = ""

	PAYPAL_CLIENT_ID = ""
	DISCORD_TOKEN    = ""

	// D_CLIENT_ID     = ""
	// D_CLIENT_SECRET = ""
	// REDIRECT_URL    = ""

	CURRENCY = ""
)

func InitConfig() {
	godotenv.Load(".env")

	var confMap = map[string]confItem{
		"DB_URI": {
			Res:      &DB_URI,
			Required: true,
		},
		"PORT": {
			Res:     &PORT,
			Default: "3000",
		},
		"PAYPAL_PATH": {
			Res:     &PAYPAL_PATH,
			Required: true,
		},
		"DEBUG_DISC_WEBHOOK": {
			Res:      &DEBUG_DISC_WEBHOOK,
		},
		"DEBUG_DISC_MENTION": {
			Res:      &DEBUG_DISC_MENTION,
		},
		"CURRENCY": {
			Res:     &CURRENCY,
			Default: "EUR",
		},
		"DISCORD_TOKEN": {
			Res: &DISCORD_TOKEN,
			Required: true,
		},
		"PAYPAL_CLIENT_ID": {
			Res: &PAYPAL_CLIENT_ID,
			Required: true,
		},

		// TODO:
		// "DISCORD_CLIENT_ID": {
		// 	PanicIfNone: true,
		// 	Res:         &D_CLIENT_ID,
		// },
		// "DISCORD_CLIENT_SECRET": {
		// 	PanicIfNone: true,
		// 	Res:         &D_CLIENT_SECRET,
		// },
	}

	for name, opt := range confMap {
		item := os.Getenv(name)

		if item == "" {
			if opt.Required {
				panic(fmt.Sprintf("'%v' is a needed variable, but is not present! Please read the README.md file for more info.", name))
			}
			item = opt.Default
		}

		*opt.Res = item
	}

	if DEBUG_DISC_WEBHOOK == "" {
		logger.Logf(LL_WARN, "DEBUG_DISC_WEBHOOK is empty! no logs will be sent to a discord")
	} else if DEBUG_DISC_MENTION == "" {
		logger.Logf(LL_WARN, "DEBUG_DISC_MENTION is empty! No one will be mentioned. On debug info")
	}
}
