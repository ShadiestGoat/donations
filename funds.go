package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v4"
)

type Fund struct {
	ID         string   `json:"id,omitempty"`
	Default    *bool    `json:"default,omitempty"`
	Goal       float64  `json:"goal,omitempty"`
	Alias      string   `json:"alias"`
	ShortTitle string   `json:"shortTitle"`
	Title      string   `json:"title"`
	Amount     *float64 `json:"amount,omitempty"`
}

type ContextKeys int

const (
	CTX_FUND ContextKeys = iota
)

func (f *Fund) PopulateAmount() {
	amt := 0.0
	DBQueryRow(`SELECT SUM(amount_received) FROM donations WHERE fund=$1`, f.ID).Scan(&amt)
	f.Amount = &amt
}

func FundMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fundID := chi.URLParam(r, "fundID")
		if fundID == "" {
			RespondErr(w, ErrNotFound)
			return
		}
		fund := &Fund{
			ID: fundID,
		}
		var err error
		if fundID == "default" {
			err = DBQueryRow(`SELECT id, goal, alias, short_title, description FROM funds WHERE def = 'true'`).Scan(
				&fund.ID,
				&fund.Goal,
				&fund.Alias,
				&fund.ShortTitle,
				&fund.Title,
			)
			def := true
			fund.Default = &def
		} else {
			err = DBQueryRow(`SELECT def, goal, alias, short_title, description FROM funds WHERE id = $1`, fundID).Scan(
				&fund.Default,
				&fund.Goal,
				&fund.Alias,
				&fund.ShortTitle,
				&fund.Title,
			)
		}

		if err != nil {
			if !errors.Is(err, pgx.ErrNoRows) {
				logger.Logf(LL_ERROR, `Couldn't fetch fund '%v': %v`, fundID, err)
			}
			RespondErr(w, ErrNotFound)
			return
		}

		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), CTX_FUND, fund)))
	})
}

func FetchFunds(before, after string, complete *bool, fetchAmounts bool) []*Fund {
	q := `SELECT id,def,goal,alias,short_title,description FROM funds`
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
	if complete != nil {
		checks = append(checks, "complete = ")
		args = append(args, *complete)
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

	q += ` ORDER BY id DESC LIMIT 50`

	funds := []*Fund{}
	rows, _ := DBQuery(q, args...)

	for rows.Next() {
		fund := &Fund{}
		rows.Scan(&fund.ID, &fund.Default, &fund.Goal, &fund.Alias, &fund.ShortTitle, &fund.Title)
		if fetchAmounts {
			fund.PopulateAmount()
		}
		funds = append(funds, fund)
	}

	return funds
}

func RouterFunds() http.Handler {
	r := chi.NewRouter()

	// Get all funds
	// pagination
	r.Get(`/`, func(w http.ResponseWriter, r *http.Request) {
		before := r.URL.Query().Get("before")
		after := r.URL.Query().Get("after")
		completeRaw := r.URL.Query().Get("complete")
		amounts := !(r.URL.Query().Get("amount") == "f" || r.URL.Query().Get("amount") == "false")

		var complete *bool

		if completeRaw != "" {
			val := completeRaw == "t" || completeRaw == "true"
			complete = &val
		}

		RespondJSON(w, 200, FetchFunds(before, after, complete, amounts))
	})

	// Create a new fund
	r.Post(`/`, func(w http.ResponseWriter, r *http.Request) {
		if NoPermHTTP(w, r, PERM_FUND_CONTROL) {
			return
		}
		fund := &Fund{}
		if ParseJSON(w, r, fund) {
			return
		}
		if fund.Default == nil {
			def := true
			fund.Default = &def
		}
		fund.ID = SnowNode.Generate().String()

		if fund.Alias == "" || fund.ShortTitle == "" || fund.Title == "" {
			RespondErr(w, ErrBadBody)
			return
		}

		aliasCount := 0

		DBQueryRow(`SELECT COUNT(*) FROM funds WHERE alias = $1`, fund.Alias).Scan(&aliasCount)

		if aliasCount != 0 {
			RespondErr(w, ErrNotUniqueAlias)
			return
		}

		if *fund.Default {
			DBExec(`UPDATE funds SET def = 'false' WHERE def = 'true'`)
		}

		DBExec(`INSERT INTO funds (id, def, goal, alias, short_title, description) VALUES ($1, $2, $3, $4, $5, $6)`,
			fund.ID,
			fund.Default,
			fund.Goal,
			fund.Alias,
			fund.ShortTitle,
			fund.Title,
		)
		RespondJSON(w, 200, fund)
		WSMgr.SendEvent(WSR_NewFund{
			Fund: fund,
		}.WSEvent())
	})

	r.Mount(`/{fundID}`, RouterFundsID())

	return r
}

func RouterFundsID() http.Handler {
	r := chi.NewRouter()

	r.Use(FundMiddleware)

	// Fetch a fund by id
	r.Get(`/`, func(w http.ResponseWriter, r *http.Request) {
		fund := r.Context().Value(CTX_FUND).(*Fund)
		fund.PopulateAmount()
		RespondJSON(w, 200, fund)
	})

	// Replace a fund's settings (except default)
	r.Put(`/`, func(w http.ResponseWriter, r *http.Request) {
		if NoPermHTTP(w, r, PERM_FUND_CONTROL) {
			return
		}
		fund := &Fund{}

		if ParseJSON(w, r, fund) {
			return
		}

		DBExec(`UPDATE funds SET goal = $1, alias = $2, short_title = $3, description = $4 WHERE id = $5`,
			fund.Goal,
			fund.Alias,
			fund.ShortTitle,
			fund.Title,
			fund.ID,
		)

		RespondSuccess(w)
	})

	r.Post(`/default`, func(w http.ResponseWriter, r *http.Request) {
		if NoPermHTTP(w, r, PERM_FUND_CONTROL) {
			return
		}
		fund := r.Context().Value(CTX_FUND).(*Fund)
		if *fund.Default {
			RespondJSON(w, 200, fund)
		} else {
			DBExec(`UPDATE funds SET def = 'false' WHERE def = 'true'`)
			DBExec(`UPDATE funds SET def = 'true' WHERE id = $1`, fund.ID)
			def := true
			fund.Default = &def
			RespondJSON(w, 200, fund)
		}
	})

	return r
}
