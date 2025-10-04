package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"slack-shiba-bot/structs"
	"slack-shiba-bot/utils"
	"strings"
)

type SlackResponse struct {
	ResponseType string `json:"response_type"`
	Text         string `json:"text,omitempty"`
}

func sendEphemeralSlackMessage(responseURL string, text string) {
	msg := SlackResponse{
		ResponseType: "ephemeral",
		Text:         text,
	}
	payload, _ := json.Marshal(msg)
	http.Post(responseURL, "application/json", bytes.NewBuffer(payload))
}

func SplitAndTrimMulti(s string) []string {
	s = strings.ReplaceAll(s, ",", " ")
	return strings.Fields(s)
}

type Game struct {
	Name              string
	Description       string
	HackatimeProjects []string
	TotalTimeToday    int
}

func HandleTodayCommand(w http.ResponseWriter, r *http.Request, server structs.SlackBot) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Failed to parse form", http.StatusBadRequest)
		return
	}

	cmd := structs.SlashCommand{
		Token:       r.PostFormValue("token"),
		TeamID:      r.PostFormValue("team_id"),
		TeamDomain:  r.PostFormValue("team_domain"),
		UserID:      r.PostFormValue("user_id"),
		UserName:    r.PostFormValue("user_name"),
		ChannelID:   r.PostFormValue("channel_id"),
		ChannelName: r.PostFormValue("channel_name"),
		Command:     r.PostFormValue("command"),
		Text:        r.PostFormValue("text"),
		ResponseURL: r.PostFormValue("response_url"),
		TriggerID:   r.PostFormValue("trigger_id"),
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Fetching your activity..."))

	go func(cmd structs.SlashCommand) {
		table := utils.GetTable(server.AirtableClient, os.Getenv("AIRTABLE_BASE_ID"), "Users")
		recordsResp, err := table.GetRecords().WithFilterFormula(fmt.Sprintf("{slack id}='%s'", cmd.UserID)).Do()
		if err != nil || len(recordsResp.Records) == 0 {
			sendEphemeralSlackMessage(cmd.ResponseURL, "Failed to get user from Airtable or user not found")
			return
		}

		userRecord := recordsResp.Records[0]
		linkedGames, ok := userRecord.Fields["Games"].([]any)
		if !ok || len(linkedGames) == 0 {
			sendEphemeralSlackMessage(cmd.ResponseURL, "User has no linked games")
			return
		}

		gamesTable := utils.GetTable(server.AirtableClient, os.Getenv("AIRTABLE_BASE_ID"), "Games")
		userGames := []Game{}

		for _, idAny := range linkedGames {
			gameID, ok := idAny.(string)
			if !ok {
				continue
			}
			gameRecord, err := gamesTable.GetRecord(gameID)
			if err != nil {
				continue
			}
			game := Game{}
			if name, ok := gameRecord.Fields["Name"].(string); ok {
				game.Name = name
			}
			if desc, ok := gameRecord.Fields["Description"].(string); ok {
				game.Description = desc
			}
			if hackatime, ok := gameRecord.Fields["Hackatime Projects"].(string); ok {
				game.HackatimeProjects = SplitAndTrimMulti(hackatime)
			}
			userGames = append(userGames, game)
		}

		resp, err := http.Get("https://hackatime.hackclub.com/api/summary?user=" + cmd.UserID + "&interval=today")
		if err != nil {
			sendEphemeralSlackMessage(cmd.ResponseURL, "Failed to fetch Hackatime data")
			return
		}
		defer resp.Body.Close()

		data, err := io.ReadAll(resp.Body)
		if err != nil {
			sendEphemeralSlackMessage(cmd.ResponseURL, "Failed to read Hackatime response")
			return
		}

		today, err := utils.ParseHackatimeSummary(data)
		if err != nil {
			sendEphemeralSlackMessage(cmd.ResponseURL, "Failed to parse Hackatime data")
			return
		}

		for i, game := range userGames {
			total := 0
			for _, p := range today.Projects {
				for _, gp := range game.HackatimeProjects {
					if strings.EqualFold(p.Key, gp) {
						total += p.Total
					}
				}
			}
			userGames[i].TotalTimeToday = total
		}

		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("*Here is your activity for today, <@%s>:*\n", cmd.UserID))
		for _, game := range userGames {
			h := game.TotalTimeToday / 3600
			m := (game.TotalTimeToday % 3600) / 60
			s := game.TotalTimeToday % 60
			sb.WriteString(fmt.Sprintf("\n*%s*\n%s\nTime spent today: %02d:%02d:%02d\n",
				game.Name, game.Description, h, m, s))
		}

		sendEphemeralSlackMessage(cmd.ResponseURL, sb.String())
	}(cmd)
}
