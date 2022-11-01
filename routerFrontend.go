package main

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v4"
)

// Prepared pages
var (
	MAIN_CSS []byte
	PAGE_FUND []byte
	PAGE_GOAL_FUND []byte
	FUNDS    []byte
	DEFAULT_PFP []byte
)

func InitFrontend() {
	b, err := os.ReadFile("pages/main.css")
	PanicIfErr(err)
	MAIN_CSS = b

	b, err = os.ReadFile("pages/fund.html")
	PanicIfErr(err)
	PAGE_FUND = Template(b, map[string][]byte{
		"CURRENCY": []byte(CURRENCY),
		"PP_CLIENT_ID": []byte(PAYPAL_CLIENT_ID),
	})

	b, err = os.ReadFile("pages/goalFund.html")
	PanicIfErr(err)
	PAGE_GOAL_FUND = Template(b, map[string][]byte{})

	DEFAULT_PFP, err = os.ReadFile("pages/defaultPFP.png")
	PanicIfErr(err)
}

func FrontendError(w http.ResponseWriter, r *http.Request, err string) {
	http.Redirect(w, r, "/error?err="+err, http.StatusTemporaryRedirect)
}

func FrontendFund(w http.ResponseWriter, r *http.Request, fundID string) {
	q := r.URL.Query()
	dID := q.Get("id")
	if dID == "@me" {
		return
	}
	if dID == "" {
		dID = "anon"
	}

	if fundID == "" {
		FrontendError(w, r, "Fund Not Found")
		return
	}

	fund := &Fund{
		ID: fundID,
	}

	var err error
	if fundID == "default" {
		err = DBQueryRow(`SELECT id, goal, quick_name, title, description FROM funds WHERE def = 'true'`).Scan(
			&fund.ID,
			&fund.Goal,
			&fund.Name,
			&fund.Title,
			&fund.Description,
		)
	} else {
		err = DBQueryRow(`SELECT goal, quick_name, title, description FROM funds WHERE id = $1`, fundID).Scan(
			&fund.Goal,
			&fund.Name,
			&fund.Title,
			&fund.Description,
		)
	}

	if err != nil {
		if !errors.Is(err, pgx.ErrNoRows) {
			logger.Logf(LL_ERROR, `Couldn't fetch fund '%v': %v`, fundID, err)
		}
		FrontendError(w, r, "Fund not found!")
		return
	}

	discordName := "Anonymous"
	discordPFP  := "defaultPFP.png"

	if dID != "anon" {
		req, _ := http.NewRequest("GET", "https://discord.com/api/v10/users/" + dID, nil)
		req.Header.Set("Authorization", DISCORD_TOKEN)
		resp, err := http.DefaultClient.Do(req)
		if err != nil || resp.StatusCode != 200 {
			dID = "anon"
		} else {
			b, _ := io.ReadAll(resp.Body)
			if len(b) == 0 {
				dID = "anon"
			} else {
				discordUser := DiscordRespUser{}
				err = json.Unmarshal(b, &discordUser)
				if err != nil {
					dID = "anon"
				} else {
					if discordUser.PFP != "" {
						discordPFP = "https://cdn.discordapp.com/avatars/" + dID + "/" + discordUser.PFP + ".webp?size=256"
					}
					discordName = discordUser.Username
				}
			}
		}
	}	

	Respond(w, 200, Template(PAGE_FUND, map[string][]byte{
		"FUND_NAME": []byte(fund.Name),
		"FUND_DESC": []byte(fund.Description),
		"FUND_ID": 	 []byte(fund.ID),
		"D_NAME": 	 []byte(discordName),
		"D_PFP": 	 []byte(discordPFP),
		"D_ID": 	 []byte(dID),
	}))
}

func RouterBase() *chi.Mux {
	r := chi.NewRouter()

	r.Get("/main.css", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/css")
		Respond(w, 200, MAIN_CSS)
	})

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		FrontendFund(w, r, "default")
	})

	r.Get("/defaultPFP.png", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		Respond(w, 200, DEFAULT_PFP)
	})

	return r
}
