package main

import (
	"strings"

	"github.com/shadiestgoat/donations/db"
)

// the name of the allowed pp event
const PP_EVENT_FILTER = "CHECKOUT.ORDER.APPROVED"

type PPEventRaw struct {
	EventType string    `json:"event_type"`
	Resource  *Resource `json:"resource"`
}

type Resource struct {
	ID            string          `json:"id"`
	PurchaseUnits []*PurchaseUnit `json:"purchase_units"`
	Payer         *Payer          `json:"payer"`
}

type PurchaseUnit struct {
	Description string   `json:"description"`
	Payments    *Payment `json:"payments"`
	Items       []*Item  `json:"items"`
}

type Item struct {
	Name string `json:"name"`
}

type Payment struct {
	Captures []*Capture `json:"captures"`
}

type Capture struct {
	ID             string           `json:"id"`
	MoneyBreakdown *SellerBreakdown `json:"seller_receivable_breakdown"`
}

type SellerBreakdown struct {
	// The amount donated
	GrossAmount *SellerBreakdownVal `json:"gross_amount"`
	// The amount you receive
	NetAmount *SellerBreakdownVal `json:"net_amount"`
}

type SellerBreakdownVal struct {
	Currency string `json:"currency_code"`
	Value    string `json:"value"`
}

func (info *SellerBreakdownVal) String() string {
	return info.Value + " " + info.Currency
}

type Payer struct {
	PayerID string `json:"payer_id"`
}

type PPDonation struct {
	PayerID   string
	OrderID   string
	CaptureID string
	DiscordID string

	Description string
	FundID      string

	AmountDonated  *SellerBreakdownVal
	AmountReceived *SellerBreakdownVal
}

// Returns a PPDonation. If the received event is invalid, it will return nil.
func (e PPEventRaw) Parse() *PPDonation {
	if e.EventType != PP_EVENT_FILTER {
		return nil
	}
	if e.Resource == nil {
		return nil
	}
	if len(e.Resource.PurchaseUnits) == 0 {
		return nil
	}

	unit := e.Resource.PurchaseUnits[0]
	if len(unit.Items) == 0 {
		return nil
	}
	if unit.Payments == nil {
		return nil
	}
	if len(unit.Payments.Captures) == 0 {
		return nil
	}
	
	capture := unit.Payments.Captures[0]

	spl := strings.Split(unit.Items[0].Name, "-")

	if !db.Exists(`funds`, `id = $1`, spl[1]) {
		return nil
	}

	return &PPDonation{
		PayerID:        e.Resource.Payer.PayerID,
		OrderID:        e.Resource.ID,
		CaptureID:      capture.ID,
		DiscordID:      spl[0],
		FundID:         spl[1],
		Description:    unit.Description,
		AmountDonated:  capture.MoneyBreakdown.GrossAmount,
		AmountReceived: capture.MoneyBreakdown.NetAmount,
	}
}
