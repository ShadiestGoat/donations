package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v4"
)

// Prepared pages
var (
	MAIN_CSS       []byte
	PAGE_FUND      []byte
	GOAL_COMPONENT []byte
	FUNDS          []byte
	DEFAULT_PFP    []byte

	GOAL_COMPONENT_WIDTH = 118.0
)

func InitFrontend() {
	b, err := os.ReadFile("pages/main.css")
	PanicIfErr(err)
	MAIN_CSS = b

	b, err = os.ReadFile("pages/fund.html")
	PanicIfErr(err)
	PAGE_FUND = Template(b, map[string][]byte{
		"CURRENCY":     []byte(CURRENCY),
		"PP_CLIENT_ID": []byte(PAYPAL_CLIENT_ID),
	})

	b, err = os.ReadFile("pages/compGoal.html")
	PanicIfErr(err)
	GOAL_COMPONENT = Template(b, map[string][]byte{
		"GOAL_WIDTH": []byte(fmt.Sprint(GOAL_COMPONENT_WIDTH)),
	})

	DEFAULT_PFP, err = os.ReadFile("pages/defaultPFP.png")
	PanicIfErr(err)
}

func FrontendError(w http.ResponseWriter, r *http.Request, err string) {
	http.Redirect(w, r, "/error?err="+url.QueryEscape(err), http.StatusTemporaryRedirect)
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
	discordPFP := "/defaultPFP.png"

	if dID != "anon" {
		dID, discordName, discordPFP = FetchDiscordUser(dID, "Bot "+DISCORD_TOKEN)
	}

	goalComp := []byte{}

	if fund.Goal != 0 {
		fund.PopulateAmount()

		perc := *fund.Amount / fund.Goal
		width := perc * GOAL_COMPONENT_WIDTH

		if perc > 1 {
			width = GOAL_COMPONENT_WIDTH
		}

		textXOffset := perc * 50 // perc * 100 / 2

		if perc > 1 || perc < 0.25 {
			textXOffset = 50
		}

		text := fmt.Sprintf(`<text x="%.2f%%" y="50%%" fill="#fff" font-family="sans-serif" font-size="9" dominant-baseline="middle" text-anchor="middle">%.0f%%</text>`, textXOffset, perc*100)

		goalComp = Template(GOAL_COMPONENT, map[string][]byte{
			"WIDTH":    []byte(fmt.Sprintf(`%.2f`, width)),
			"TEXT_SVG": []byte(text),
			"GOAL_MAX": []byte(fmt.Sprint(fund.Goal)),
		})
	}

	Respond(w, 200, Template(PAGE_FUND, map[string][]byte{
		"FUND_NAME": []byte(fund.Name),
		"FUND_DESC": []byte(fund.Description),
		"FUND_ID":   []byte(fund.ID),
		"CURRENCY":  []byte(CURRENCY),
		"D_NAME":    []byte(discordName),
		"D_PFP":     []byte(discordPFP),
		"D_ID":      []byte(dID),
		"GOAL_COMP": goalComp,
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

	r.Get("/funds", func(w http.ResponseWriter, r *http.Request) {
		// FrontendFund(w, r, "default")
		// TODO:
	})

	r.Get(`/f/{quickName}`, func(w http.ResponseWriter, r *http.Request) {
		id := ""
		err := DBQueryRow(`SELECT id FROM funds WHERE quick_name = $1`, chi.URLParam(r, "quickName")).Scan(&id)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				FrontendError(w, r, "404 Not Found")
			} else {
				logger.Logf(LL_ERROR, "Couldn't fetch: %v", err)
				FrontendError(w, r, "Unknown Error")
			}
			return
		}
		FrontendFund(w, r, id)
	})

	r.Get("/funds/{fundID}", func(w http.ResponseWriter, r *http.Request) {
		FrontendFund(w, r, chi.URLParam(r, "fundID"))
	})

	r.Get(`/login`, func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if q.Get("code") == "" {
			http.Redirect(w, r, DISCORD_OAUTH_LINK, http.StatusTemporaryRedirect)
			return
		}

		vals := url.Values{}

		vals.Set("client_id", D_CLIENT_ID)
		vals.Set("client_secret", D_CLIENT_SECRET)
		vals.Set("grant_type", "authorization_code")
		vals.Set("code", q.Get("code"))
		vals.Set("redirect_uri", DISCORD_OAUTH_REDIRECT)

		resp, err := http.PostForm(DISCORD_BASE_URL+"/oauth2/token", vals)

		if err != nil || resp.StatusCode != 200 || resp.Body == nil {
			if err == nil && resp.Body != nil {
				b, _ := io.ReadAll(resp.Body)
				fmt.Println(resp.StatusCode, string(b))
			}
			logger.Logf(LL_ERROR, "Couldn't login user!")
			FrontendError(w, r, "Unknown Error")
			return
		}

		auth := &DiscordOAuth2{}
		b, _ := io.ReadAll(resp.Body)

		err = json.Unmarshal(b, auth)

		if err != nil {
			logger.Logf(LL_ERROR, "Couldn't login user: %v\n%v", err, string(b))
			FrontendError(w, r, "Unknown Error")
			return
		}

		dID, _, _ := FetchDiscordUser("@me", "Bearer "+auth.Token)

		fund := q.Get("fund")

		red := "/"

		if fund != "" {
			red = "/funds/" + fund
		}

		red += "?id=" + dID

		http.Redirect(w, r, red, http.StatusTemporaryRedirect)
	})

	r.Get("/defaultPFP.png", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		Respond(w, 200, DEFAULT_PFP)
	})

	return r
}
