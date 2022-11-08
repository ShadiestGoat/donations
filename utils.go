package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"math"
	"math/rand"
	"net/http"
	"time"
)

type HTTPError struct {
	Status    int
	Err       string
	CachedMsg []byte
}

func (h HTTPError) Error() string {
	return h.Err
}

var (
	ErrNotFound      = &HTTPError{Status: 404, Err: "notFound"}
	ErrBadBody       = &HTTPError{Status: 400, Err: "badBody"}
	ErrDoubleAccess  = &HTTPError{Status: 400, Err: "doubleAccess"}
	ErrNoUser        = &HTTPError{Status: 400, Err: "UserNotFound"}
	ErrNotAuthorized = &HTTPError{Status: 401, Err: "notAuthorized"}
	ErrServerBad     = &HTTPError{Status: 500, Err: "serverBad"}
	// ErrNoPollLeft = &HTTPError{Status: 200, Err: "noPollLeft"}
	// ErrBodyMissing = &HTTPError{Status: 400, Err: "bodyMissing"}
	// ErrRateLimit = &HTTPError{Status: 429, Err: "rateLimit"}
	// ErrOAuth2Code = &HTTPError{Status: 400, Err: "noCode"}
	// ErrBadEmail = &HTTPError{Status: 401, Err: "badEmailDomain"}
	// ErrBadLimit = &HTTPError{Status: 400, Err: "badLimit"}
	// ErrBanned = &HTTPError{Status: 401, Err: "banned"}
)

func init() {
	allErrors := []*HTTPError{
		// ErrNotFound,
		ErrBadBody,
		// ErrBadLength,
		// ErrProfanity,
		// ErrBodyMissing,
		// ErrRateLimit,
		// ErrOAuth2Code,
		// ErrBadEmail,
		// ErrBadLimit,
		ErrNotAuthorized,
		// ErrBanned,
	}
	for _, err := range allErrors {
		err.CachedMsg = []byte(fmt.Sprintf(`{"error":"%v"}`, err.Err))
	}
}

var msgSucc = []byte(`{"status":"success"}`)

func RespondSuccess(w http.ResponseWriter) {
	Respond(w, 200, msgSucc)
}

// Panics if err != nil. Should only be used pre-server setup, or w/ debug
func PanicIfErr(err error) {
	if err != nil {
		panic(err)
	}
}

// util function to write a unified error message
func RespondErr(w http.ResponseWriter, err *HTTPError) {
	Respond(w, err.Status, err.CachedMsg)
}

// util function for responding w/ a string
func RespondString(w http.ResponseWriter, status int, msg string) {
	Respond(w, status, []byte(msg))
}

// util function to respond w/ a status. Just puts the things in the same place
func Respond(w http.ResponseWriter, status int, msg []byte) {
	w.WriteHeader(status)
	w.Write(msg)
}

func FrontendRespond(w http.ResponseWriter, r *http.Request, Page *template.Template, templateName string, data any) {
	w.WriteHeader(200)
	err := Page.ExecuteTemplate(w, templateName, data)
	if err != nil {
		FrontendError(w, r, "Can't load page")
	}
}

func RespondJSON(w http.ResponseWriter, status int, val any) {
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(val)
}

// returns true if there was an error
func ParseJSON(w http.ResponseWriter, r *http.Request, v any) bool {
	if r.Body == nil {
		RespondErr(w, ErrBadBody)
		return true
	}
	err := json.NewDecoder(r.Body).Decode(v)
	if err != nil {
		RespondErr(w, ErrBadBody)
		return true
	}
	return false
}

func RandInt(min, max int) int {
	rand.Seed(time.Now().UnixNano())
	return rand.Intn(max-min) + min
}

// Return the current paycycle
func PayCycle(day int, now time.Time) time.Time {
	curMonth := now.Month()
	if now.Day() > day {
		if curMonth == time.January {
			curMonth = time.December
		} else {
			curMonth -= 1
		}

		return time.Date(now.Year(), curMonth, day, 0, 0, 0, 0, time.UTC)
	}
	return time.Date(now.Year(), curMonth, day, 0, 0, 0, 0, time.UTC)
}

func NewCycle() int {
	day := time.Now().Day()
	if day > 28 {
		day = 28
	}
	return day
}

func Template(inp []byte, variables map[string][]byte) []byte {
	for varName, val := range variables {
		inp = bytes.ReplaceAll(inp, []byte("{{%"+varName+"}}"), val)
	}
	return inp
}

func Round(val float64, decimals int) float64 {
	vDiv := math.Pow10(decimals)
	return math.Round(val*vDiv)/vDiv
}
