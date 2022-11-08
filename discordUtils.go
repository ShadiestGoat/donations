package main

import (
	"encoding/json"
	"io"
	"net/http"
)

type DiscordRespUser struct {
	Username string `json:"username"`
	PFP      string `json:"avatar"`
	ID       string `json:"id"`
}

const DISCORD_BASE_URL = "https://discord.com/api/v10"

func FetchDiscordUser(id string, token string) (oID string, name string, pfp string) {
	req, _ := http.NewRequest("GET", "https://discord.com/api/v10/users/"+id, nil)
	req.Header.Set("Authorization", token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil || resp.StatusCode != 200 {
		oID = "anon"
	} else {
		b, _ := io.ReadAll(resp.Body)
		if len(b) == 0 {
			oID = "anon"
		} else {
			discordUser := DiscordRespUser{}
			err = json.Unmarshal(b, &discordUser)
			if err != nil {
				oID = "anon"
			} else {
				if discordUser.PFP != "" {
					pfp = "https://cdn.discordapp.com/avatars/" + id + "/" + discordUser.PFP + ".webp?size=256"
				}
				name = discordUser.Username
			}
		}
	}
	if oID == "anon" {
		name = "Anonymous"
		pfp = "defaultPFP.png"
	}
	return
}

type OAUth2 struct {
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
	GrantType    string `json:"authorization_code"`
	Code         string `json:"code"`
	RedirectURI  string `json:"redirect_uri"`
}
