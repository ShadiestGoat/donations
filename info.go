package main

import (
	"errors"
	"time"

	"github.com/jackc/pgx/v4"
)

type DonorInfo struct {
	Total float64 `json:"total"`
	Month float64 `json:"month"`
}

type Donor struct {
	ID        string `json:"id"`
	DiscordID string `json:"discordID"`
	PayPal    string `json:"PayPal"`
	CycleDay  int    `json:"payCycle"`
}

type Donation struct {
	ID        string  `json:"id"`
	OrderID   string  `json:"ppOrderID"`
	CaptureID string  `json:"ppCaptureID"`
	Donor     string  `json:"donor"`
	Message   string  `json:"message"`
	Amount    float64 `json:"amount"`
	FundID    string  `json:"fundID"`
}

type ProfileResponse struct {
	Donors    []*Donor     `json:"donors"`
	Total     *DonorInfo   `json:"total"`
	Donations *[]*Donation `json:"donations,omitempty"`
}

// Return ProfileResponse based on DonorID
// if resolve is true, ProfileResponse.Donations is populated.
func FetchProfileByDonor(id string, resolve bool) *ProfileResponse {
	donor := &Donor{
		ID: id,
	}
	err := DBQueryRow(`SELECT discord_id, paypal, cycle FROM donors WHERE id = $1`, id).Scan(
		&donor.DiscordID,
		&donor.PayPal,
		&donor.CycleDay,
	)
	if err != nil {
		if !errors.Is(err, pgx.ErrNoRows) {
			logger.Logf(LL_ERROR, `Couldn't fetch payer from DB: %v`, err)
		}
		return &ProfileResponse{
			Donors:    []*Donor{},
			Total:     &DonorInfo{},
			Donations: &[]*Donation{},
		}
	}
	now := time.Now()
	cycle := PayCycle(donor.CycleDay, now)

	if resolve {
		rows, _ := DBQuery(`SELECT id, order_id, capture_id, amount, message, fund FROM donations WHERE donor = $1 ORDER BY DESC`, id)
		donations := []*Donation{}
		total := &DonorInfo{
			Total: 0,
			Month: 0,
		}
		for rows.Next() {
			donation := &Donation{
				Donor: id,
			}
			rows.Scan(&donation.ID, &donation.OrderID, &donation.CaptureID, &donation.Amount, &donation.Message, &donation.FundID)
			total.Total += donation.Amount
			donationTime := SnowToTime(donation.ID)
			if donationTime.Unix() > cycle.Unix() {
				total.Month += donation.Amount
			}
			donations = append(donations, donation)
		}
		return &ProfileResponse{
			Donors:    []*Donor{donor},
			Total:     total,
			Donations: &donations,
		}
	} else {
		total, monthly := 0.0, 0.0
		DBQueryRow(`SELECT SUM(amount) FROM donations WHERE donor = $1`, id).Scan(&total)
		DBQueryRow(`SELECT SUM(amount) FROM donations WHERE donor = $1 AND id >= $2`, id, TimeToSnow(cycle)).Scan(&monthly)
		return &ProfileResponse{
			Donors: []*Donor{
				donor,
			},
			Total: &DonorInfo{
				Total: total,
				Month: monthly,
			},
			Donations: nil,
		}
	}
}

func FetchProfileByX(columnName string, id string, resolve bool) *ProfileResponse {
	rows, _ := DBQuery(`SELECT id FROM donors WHERE `+columnName+` = $1`, id)
	resp := &ProfileResponse{
		Donors:    []*Donor{},
		Total:     &DonorInfo{},
		Donations: nil,
	}
	if resolve {
		resp.Donations = &[]*Donation{}
	}
	for rows.Next() {
		donorID := ""
		rows.Scan(&donorID)
		cResp := FetchProfileByDonor(donorID, resolve)
		resp.Donors = append(resp.Donors, cResp.Donors...)
		if resolve {
			*resp.Donations = append(*resp.Donations, *cResp.Donations...)
		}
		resp.Total.Month += cResp.Total.Month
		resp.Total.Total += cResp.Total.Total
	}
	return resp
}

func FetchProfileByDiscord(id string, resolve bool) *ProfileResponse {
	return FetchProfileByX("discord_id", id, resolve)
}

func FetchProfileByPaypal(id string, resolve bool) *ProfileResponse {
	return FetchProfileByX("paypal", id, resolve)
}
