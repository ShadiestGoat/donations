package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v4"
)

// Returns true if the request's app doesn't have the perm for this route
func NoPermHTTP(w http.ResponseWriter, r *http.Request, perm Permission) bool {
	token := r.Header.Get("Authorization")
	app, ok := Apps[token]
	if token == "" || !ok {
		RespondErr(w, ErrNotAuthorized)
		return true
	}
	if HasPerm(token, perm) {
		logger.Logf(LL_DEBUG, "[%v]: Requested '%v'", app.Name, r.URL)
		return false
	}
	RespondErr(w, ErrNotAuthorized)
	return true
}

// Returns true if app with 'token' has this permission
func HasPerm(token string, perm Permission) bool {
	return perm&Apps[token].Perms == perm || Apps[token].Perms == PERM_ADMIN
}

func RouterDonors() http.Handler {
	r := chi.NewRouter()

	r.Get(`/discord/{discordID}`, func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		out := FetchProfileByDiscord(chi.URLParam(r, `discordID`), q.Get("resolve") == "true")
		if len(out.Donors) == 0 {
			RespondErr(w, ErrNotFound)
			return
		}
		enc, _ := json.Marshal(out)
		Respond(w, 200, enc)
	})

	r.Get(`/donor/{donorID}`, func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		out := FetchProfileByDonor(chi.URLParam(r, `donorID`), q.Get("resolve") == "true")
		if len(out.Donors) == 0 {
			RespondErr(w, ErrNotFound)
			return
		}
		enc, _ := json.Marshal(out)
		Respond(w, 200, enc)
	})

	r.Get(`/paypal/{paypalID}`, func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		out := FetchProfileByPaypal(chi.URLParam(r, `paypalID`), q.Get("resolve") == "true")
		if len(out.Donors) == 0 {
			RespondErr(w, ErrNotFound)
			return
		}
		enc, _ := json.Marshal(out)
		Respond(w, 200, enc)
	})

	return r
}

func RouterAPI() http.Handler {
	r := chi.NewRouter()

	r.Get("/discordToken", func(w http.ResponseWriter, r *http.Request) {
		if NoPermHTTP(w, r, PERM_DISCORD_TOKEN) {
			return
		}
		Respond(w, 200, []byte(`{"token": "`+DISCORD_TOKEN+`"}`))
	})

	r.Mount(`/donors`, RouterDonors())
	r.Mount(`/funds`, RouterFunds())

	r.HandleFunc("/ws", socketHandler)

	r.Get(`/donations`, func(w http.ResponseWriter, r *http.Request) {
		before := r.URL.Query().Get("before")
		after := r.URL.Query().Get("after")

		q := `SELECT id,donor,amount,message,fund FROM donations`
		args := []any{}
		checks := []string{}
		if before != "" {
			checks = append(checks, "id <= ")
			args = append(args, before)
		}
		if after != "" {
			checks = append(checks, "id >= ")
			args = append(args, after)
		}
		if len(checks) != 0 {
			q += " WHERE"
		}

		for argIndex, check := range checks {
			if argIndex != 0 {
				q += " AND "
			}
			q += check + "$" + fmt.Sprint(argIndex+1)
		}

		q += ` ORDER BY id DESC LIMIT 50`

		donos := []*Donation{}
		rows, _ := DBQuery(q, args...)

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
		if err != nil {
			logger.Logf(LL_WARN, "An illegal paypal event has been received!\n-> |"+string(b)+"| <-")
			RespondErr(w, ErrBadBody)
			return
		}

		donation := rawPayPal.Parse()
		if donation == nil {
			logger.Logf(LL_WARN, "An illegal paypal event has been received! (Parse)\n-> |"+string(b)+"| <-")
			RespondErr(w, ErrBadBody)
			return
		} else {
			logger.Logf(LL_DEBUG, "New Donation parsed!\nPayerID: %v\nOrder/Capture IDs: %v %v\nAmount Donated: %v\nAmount Received: %v\nMessage: %v", donation.PayerID, donation.OrderID, donation.CaptureID, donation.AmountDonated, donation.AmountReceived, donation.Description)
		}

		if donation.DiscordID == "" {
			RespondErr(w, ErrNoUser)
			return
		}

		donorID, cycle := "", 0

		err = DBQueryRow(`SELECT id, cycle FROM donors WHERE discord_id = $1 AND paypal = $2`, donation.DiscordID, donation.PayerID).Scan(&donorID, &cycle)

		newPayer := false

		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				donorID, cycle = SnowNode.Generate().String(), NewCycle()
				newPayer = true
				_, err = DBExec(`INSERT INTO donors (id, discord_id, paypal, cycle) VALUES ($1, $2, $3, $4)`, donorID, donation.DiscordID, donation.PayerID, cycle)
				if err != nil {
					logger.Logf(LL_ERROR, "Can't insert into DB: (id: %v, discord_id: %v, paypal: %v, cycle: %v): %v", donorID, donation.DiscordID, donation.PayerID, cycle, err)
					RespondErr(w, ErrServerBad)
					return
				}
			} else {
				logger.Logf(LL_ERROR, "Bad db fetch for donors: d: %v pp: %v, err: %v", donation.DiscordID, donation.PayerID, err)
				RespondErr(w, ErrServerBad)
				return
			}
		}

		donationID := SnowNode.Generate().String()

		if !newPayer {
			cycleTime := PayCycle(cycle, time.Now())

			maxID, minID := TimeToSnow(cycleTime.AddDate(0, -1, 0)), TimeToSnow(cycleTime.AddDate(0, -2, 0))

			lastMonthDonos := 0

			DBQueryRow(`SELECT COUNT(*) FROM donations WHERE payer = $1 AND id BETWEEN $2 AND $3`, donorID, minID, maxID).Scan(&lastMonthDonos)

			if lastMonthDonos == 0 {
				DBExec(`UPDATE payers SET cycle = $1 WHERE id = $2`, NewCycle(), donorID)
			}
		}

		amount, _ := strconv.ParseFloat(donation.AmountDonated.Value, 64)
		amountReceived, _ := strconv.ParseFloat(donation.AmountReceived.Value, 64)

		_, err = DBExec("INSERT INTO donations(id, order_id, capture_id, donor, amount, amount_received, message, fund) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)", donationID, donation.OrderID, donation.CaptureID, donorID, amount, amountReceived, donation.Description, donation.FundID)

		if err != nil {
			RespondErr(w, ErrServerBad)
			return
		}

		fundGoal := 0.0

		DBQueryRow(`SELECT goal FROM funds WHERE id = $1`, donation.FundID).Scan(&fundGoal)

		if fundGoal != 0 {
			f := Fund{ID: donation.FundID}
			f.PopulateAmount()
			if *f.Amount >= fundGoal {
				DBExec(`UPDATE funds SET complete = 'true' WHERE id = $1`, donation.FundID)
			}
		}

		logger.Logf(LL_SUCCESS, "Donation parsed!\nPayerID: %v\nOrder/Capture IDs: %v %v\nAmount Donated: %v\nAmount Received: %v\nMessage: %v", donation.PayerID, donation.OrderID, donation.CaptureID, donation.AmountDonated, donation.AmountReceived, donation.Description)
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
