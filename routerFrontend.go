package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v4"
)

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
		err = DBQueryRow(`SELECT id, goal, short_title, description FROM funds WHERE def = 'true'`).Scan(
			&fund.ID,
			&fund.Goal,
			&fund.ShortTitle,
			&fund.Title,
		)
	} else {
		err = DBQueryRow(`SELECT goal, short_title, description FROM funds WHERE id = $1`, fundID).Scan(
			&fund.Goal,
			&fund.ShortTitle,
			&fund.Title,
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

	var goalComp *ComponentGoal

	if fund.Goal != 0 {
		fund.PopulateAmount()
		goalComp = NewComponentGoal(fund.Goal, *fund.Amount)
	}

	FrontendRespond(w, r, PAGE_FUND, "fund", PageFund{
		DiscordPFP:     discordPFP,
		DiscordName:    discordName,
		DiscordID:      dID,
		FundID:         fund.ID,
		FundTitle:      fund.Title,
		FundShortTitle: fund.ShortTitle,
		Goal:           goalComp,
	})
}

func RouterBase() *chi.Mux {
	r := chi.NewRouter()

	r.Use(middleware.StripSlashes)
	r.Use(middleware.CleanPath)
	r.Use(middleware.Recoverer)

	r.Get("/main.css", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/css")
		Respond(w, 200, MAIN_CSS)
	})

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		FrontendFund(w, r, "default")
	})

	r.Get("/funds", func(w http.ResponseWriter, r *http.Request) {
		rows, _ := DBQuery(`SELECT id,goal,short_title,description,def FROM funds WHERE complete = 'false' order by id LIMIT 50`)
		funds := []*PageFundsFund{}
		for rows.Next() {
			goal := 0.0
			def := false
			frontendFund := &PageFundsFund{}
			rows.Scan(&frontendFund.ID, &goal, &frontendFund.ShortTitle, &frontendFund.Title, &def)
			if goal != 0 {
				fund := &Fund{
					ID: frontendFund.ID,
				}
				fund.PopulateAmount()
				frontendFund.Goal = NewComponentGoal(goal, *fund.Amount)
			}
			if def {
				funds = append([]*PageFundsFund{frontendFund}, funds...)
			} else {
				funds = append(funds, frontendFund)
			}
		}
		FrontendRespond(w, r, PAGE_FUNDS, "funds", funds)
	})

	r.Get(`/f/{quickName}`, func(w http.ResponseWriter, r *http.Request) {
		id := ""
		err := DBQueryRow(`SELECT id FROM funds WHERE alias = $1`, chi.URLParam(r, "quickName")).Scan(&id)
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

		http.Redirect(w, r, "/?id="+dID, http.StatusTemporaryRedirect)
	})

	r.Get("/error", func(w http.ResponseWriter, r *http.Request) {
		Respond(w, 200, PAGE_ERROR)
	})

	r.Get("/defaultPFP.png", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		Respond(w, 200, DEFAULT_PFP)
	})

	return r
}
