package main

import (
	"html/template"
	"os"
)

type ComponentGoal struct {
	GoalValue   float64
	XPercOffset float64
	Perc        float64
}

func NewComponentGoal(goal float64, currentFund float64) *ComponentGoal {
	if goal == 0 {
		return nil
	}
	perc := currentFund / goal

	textOffset := perc * 50 // perc * 100 / 2

	if perc > 1 || perc < 0.25 {
		textOffset = 50
	}

	return &ComponentGoal{
		GoalValue:   Round(goal, 2),
		XPercOffset: Round(textOffset, 2),
		Perc:        Round(perc*100, 0),
	}
}

type PageFund struct {
	DiscordPFP  string
	DiscordName string
	DiscordID   string

	FundID         string
	FundTitle      string
	FundShortTitle string

	Goal *ComponentGoal
}

type PageFundsFund struct {
	ID         string
	Title      string
	ShortTitle string
	Goal       *ComponentGoal
}

// Prepared pages
var (
	MAIN_CSS    []byte
	DEFAULT_PFP []byte

	PAGE_FUNDS *template.Template
	PAGE_FUND  *template.Template
	PAGE_ERROR []byte
)

func InitFrontend() {
	b, err := os.ReadFile("pages/main.css")
	PanicIfErr(err)
	MAIN_CSS = b

	DEFAULT_PFP, err = os.ReadFile("pages/defaultPFP.png")
	PanicIfErr(err)

	b, err = os.ReadFile("pages/fund.html")
	PanicIfErr(err)
	fundRaw := Template(b, map[string][]byte{
		"CURRENCY":     []byte(CURRENCY),
		"PP_CLIENT_ID": []byte(PAYPAL_CLIENT_ID),
	})

	PAGE_FUND, err = template.New("fund").Parse(string(fundRaw))
	PanicIfErr(err)

	fundsRaw, err := os.ReadFile("pages/funds.html")
	PanicIfErr(err)

	PAGE_FUNDS, err = template.New("funds").Parse(string(fundsRaw))
	PanicIfErr(err)

	PAGE_ERROR, err = os.ReadFile("pages/error.html")
	PanicIfErr(err)
}
