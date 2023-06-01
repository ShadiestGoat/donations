package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v4"
	"github.com/shadiestgoat/donations/auth"
	"github.com/shadiestgoat/donations/db"
	"github.com/shadiestgoat/log"
)

// Handles authentication & responses.
// Returns true if the request's app doesn't have the perm for this route
func NoPermHTTP(w http.ResponseWriter, r *http.Request, perm auth.Permission) bool {
	token := r.Header.Get("Authorization")

	app := auth.FetchApp(token)

	if token == "" || app == nil {
		RespondErr(w, ErrNotAuthorized)
		return true
	}

	if auth.HasPerm(token, perm) {
		log.Debug(`[%s]: Requested '%s`, app.Name, r.URL)
		return false
	}

	RespondErr(w, ErrNotAuthorized)
	return true
}

func RouterDonors() http.Handler {
	r := chi.NewRouter()

	r.Get(`/discord/{discordID}`, func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		out := FetchProfileByDiscord(chi.URLParam(r, `discordID`), q.Get("resolve") == "true")
		enc, _ := json.Marshal(out)
		Respond(w, 200, enc)
	})

	r.Get(`/donor/{donorID}`, func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		out := FetchProfileByDonor(chi.URLParam(r, `donorID`), q.Get("resolve") == "true")
		enc, _ := json.Marshal(out)
		Respond(w, 200, enc)
	})

	r.Get(`/paypal/{paypalID}`, func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		out := FetchProfileByPaypal(chi.URLParam(r, `paypalID`), q.Get("resolve") == "true")
		enc, _ := json.Marshal(out)
		Respond(w, 200, enc)
	})

	return r
}

func RouterAPI() http.Handler {
	r := chi.NewRouter()

	r.Get("/discordToken", func(w http.ResponseWriter, r *http.Request) {
		if NoPermHTTP(w, r, auth.PERM_DISCORD_TOKEN) {
			return
		}
		Respond(w, 200, []byte(`{"token": "`+DISCORD_TOKEN+`"}`))
	})

	r.Mount(`/donors`, RouterDonors())
	r.Mount(`/funds`, RouterFunds())

	r.HandleFunc("/ws", socketHandler)

	r.Get(`/donations`, func(w http.ResponseWriter, r *http.Request) {
		if NoPermHTTP(w, r, auth.PERM_FETCH_DONATIONS) {
			return
		}
		before := r.URL.Query().Get("before")
		after := r.URL.Query().Get("after")

		q := `SELECT id,donor,amount,message,fund FROM donations`
		args := []any{}
		checks := []string{}

		qSplit := strings.Split(r.URL.RawQuery, "&")

		order := "DESC"

		for _, rawQ := range qSplit {
			rawV := strings.Split(rawQ, "=")
			v := rawV[0]

			if v == "before" {
				order = "DESC"
				break
			} else if v == "after" {
				order = "ASC"
			}
		}

		if before != "" {
			checks = append(checks, "id < ")
			args = append(args, before)
		}
		if after != "" {
			checks = append(checks, "id > ")
			args = append(args, after)
		}

		if len(checks) != 0 {
			q += " WHERE "
		}

		for argIndex, check := range checks {
			if argIndex != 0 {
				q += " AND "
			}
			q += check + "$" + fmt.Sprint(argIndex+1)
		}

		q += ` ORDER BY id ` + order + ` LIMIT 50`

		donos := []*Donation{}
		rows, _ := db.Query(q, args...)

		for rows.Next() {
			don := &Donation{}
			rows.Scan(&don.ID, &don.Donor, &don.Amount, &don.Message, &don.FundID)
			donos = append(donos, don)
		}

		RespondJSON(w, 200, donos)
	})

	r.Post("/"+PAYPAL_PATH, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rawPayPal := &PPEventRaw{}
		if r.Body == nil {
			RespondErr(w, ErrBadBody)
			return
		}
		b, _ := io.ReadAll(r.Body)
		if len(b) == 0 {
			RespondErr(w, ErrBadBody)
			return
		}
		err := json.Unmarshal(b, &rawPayPal)

		if log.ErrorIfErr(err, "parsing paypal event") {
			log.Error(string(b))

			RespondErr(w, ErrBadBody)
			return
		}

		donation := rawPayPal.Parse()

		if donation == nil {
			log.Error("An illegal paypal even has been received! (Parse)")
			log.Error(string(b))

			RespondErr(w, ErrBadBody)
			return
		}

		if donation.DiscordID == "" {
			RespondErr(w, ErrNoUser)
			return
		}

		donorID, cycle := "", 0

		if donation.DiscordID == "anon" {
			donation.DiscordID = ""
		}

		err = db.QueryRow(`SELECT id, cycle FROM donors WHERE discord_id = $1 AND paypal = $2`, []any{donation.DiscordID, donation.PayerID}, &donorID, &cycle)

		newPayer := false

		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				donorID, cycle = SnowNode.Generate().String(), NewCycle()
				newPayer = true

				_, err = db.Exec(`INSERT INTO donors (id, discord_id, paypal, cycle) VALUES ($1, $2, $3, $4)`, donorID, donation.DiscordID, donation.PayerID, cycle)

				if err != nil {
					RespondErr(w, ErrServerBad)
					return
				}
			} else {
				RespondErr(w, ErrServerBad)
				return
			}
		}

		donationID := SnowNode.Generate().String()

		if !newPayer {
			cycleTime := PayCycle(cycle, time.Now())

			maxID, minID := TimeToSnow(cycleTime.AddDate(0, -1, 0)), TimeToSnow(cycleTime.AddDate(0, -2, 0))

			lastMonthDonos := 0

			db.QueryRow(`SELECT COUNT(*) FROM donations WHERE payer = $1 AND id BETWEEN $2 AND $3`, []any{donorID, minID, maxID}, &lastMonthDonos)

			if lastMonthDonos == 0 {
				db.Exec(`UPDATE payers SET cycle = $1 WHERE id = $2`, NewCycle(), donorID)
			}
		}

		amount, _ := strconv.ParseFloat(donation.AmountDonated.Value, 64)
		amountReceived, _ := strconv.ParseFloat(donation.AmountReceived.Value, 64)

		_, err = db.Exec("INSERT INTO donations(id, order_id, capture_id, donor, amount, amount_received, message, fund) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)", donationID, donation.OrderID, donation.CaptureID, donorID, amount, amountReceived, donation.Description, donation.FundID)

		if err != nil {
			RespondErr(w, ErrServerBad)
			return
		}

		fundGoal := 0.0

		db.QueryRowID(`SELECT goal FROM funds WHERE id = $1`, donation.FundID, &fundGoal)

		if fundGoal != 0 {
			f := Fund{ID: donation.FundID}
			f.PopulateAmount()
			if *f.Amount >= fundGoal {
				db.Exec(`UPDATE funds SET complete = 'true' WHERE id = $1`, donation.FundID)
			}
		}

		log.Success("Donation parsed!\nPayerID: %v\nOrder/Capture IDs: %v %v\nAmount Donated: %v\nAmount Received: %v\nMessage: %v", donation.PayerID, donation.OrderID, donation.CaptureID, donation.AmountDonated, donation.AmountReceived, donation.Description)

		RespondSuccess(w)

		WSMgr.SendEvent(WSR_NewDon{
			Donation: &Donation{
				ID:        donationID,
				OrderID:   donation.OrderID,
				CaptureID: donation.CaptureID,
				Donor:     donorID,
				Message:   donation.Description,
				Amount:    amount,
				FundID:    donation.FundID,
			},
		}.WSEvent())
	}))

	return r
}
