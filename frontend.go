package main

import (
	"html/template"
	"os"

	"github.com/shadiestgoat/log"
)

type ComponentGoal struct {
	GoalValue   float64
	XPercOffset float64
	Perc        float64
	Width       float64
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

	width := perc

	if perc < 0.12 && perc != 0 {
		width = 0.12
	} else if perc > 1 {
		width = 1
	}

	return &ComponentGoal{
		GoalValue:   Round(goal, 2),
		XPercOffset: Round(textOffset, 2),
		Perc:        Round(perc*100, 0),
		Width:       Round(width * 100, 1),
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
	PAGE_THANKS []byte
)

func InitFrontend() {
	b, err := os.ReadFile("pages/main.css")
	log.FatalIfErr(err, "opening 'pages/main.css'")

	MAIN_CSS = b

	DEFAULT_PFP, err = os.ReadFile("pages/defaultPFP.png")
	log.FatalIfErr(err, "opening defaultPFP.png image")

	b, err = os.ReadFile("pages/fund.html")
	log.FatalIfErr(err, "opening 'pages/fund.html'")

	fundRaw := Template(b, map[string][]byte{
		"CURRENCY":     []byte(CURRENCY),
		"PP_CLIENT_ID": []byte(PAYPAL_CLIENT_ID),
	})

	PAGE_FUND, err = template.New("fund").Parse(string(fundRaw))
	log.FatalIfErr(err, "parsing/creating the fund template")

	fundsRaw, err := os.ReadFile("pages/funds.html")
	log.FatalIfErr(err, "opening the 'funds' image")
	
	PAGE_FUNDS, err = template.New("funds").Parse(string(fundsRaw))
	log.FatalIfErr(err, "parsing/creating the 'funds' template")
	
	PAGE_ERROR, err = os.ReadFile("pages/error.html")
	log.FatalIfErr(err, "opening the error page")
	
	PAGE_THANKS, err = os.ReadFile("pages/thanks.html")
	log.FatalIfErr(err, "opening the 'thanks' page")
}
