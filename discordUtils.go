package main

import (
	"encoding/json"
	"fmt"
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
		if err == nil && resp.Body != nil {
			b, _ := io.ReadAll(resp.Body)
			fmt.Println(string(b))	
		}
		oID = "anon"
	} else {
		b, _ := io.ReadAll(resp.Body)
		fmt.Println(string(b))
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
				oID = discordUser.ID
			}
		}
	}
	if oID == "anon" {
		name = "Anonymous"
		pfp = "defaultPFP.png"
	}
	return
}

type DiscordOAuth2 struct {
	Token string `json:"access_token"`
}
