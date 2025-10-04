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
	resp, err := http.Post(responseURL, "application/json", bytes.NewBuffer(payload))
	if err != nil {
		fmt.Printf("[ERROR] Failed to send ephemeral message: %v\n", err)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		fmt.Printf("[WARN] Slack responded with status %d: %s\n", resp.StatusCode, string(body))
	}
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
		fmt.Printf("[INFO] Fetching user from Airtable with Slack ID: %s\n", cmd.UserID)
		table := utils.GetTable(server.AirtableClient, os.Getenv("AIRTABLE_BASE_ID"), "Users")
		recordsResp, err := table.GetRecords().WithFilterFormula(fmt.Sprintf("{slack id}='%s'", cmd.UserID)).Do()
		if err != nil {
			fmt.Printf("[ERROR] Airtable request failed: %v\n", err)
			sendEphemeralSlackMessage(cmd.ResponseURL, fmt.Sprintf("Error fetching user: %v", err))
			return
		}

		if len(recordsResp.Records) == 0 {
			fmt.Printf("[WARN] No user found with Slack ID %s\n", cmd.UserID)
			sendEphemeralSlackMessage(cmd.ResponseURL, "No Airtable user found for your Slack ID")
			return
		}

		userRecord := recordsResp.Records[0]
		linkedGames, ok := userRecord.Fields["Games"].([]any)
		if !ok || len(linkedGames) == 0 {
			fmt.Printf("[INFO] User %s has no linked games\n", cmd.UserID)
			sendEphemeralSlackMessage(cmd.ResponseURL, "You have no linked games in Airtable")
			return
		}
		fmt.Printf("[INFO] User %s has %d linked games\n", cmd.UserID, len(linkedGames))

		gamesTable := utils.GetTable(server.AirtableClient, os.Getenv("AIRTABLE_BASE_ID"), "Games")
		userGames := []Game{}

		for _, idAny := range linkedGames {
			gameID, ok := idAny.(string)
			if !ok {
				fmt.Printf("[WARN] Game ID is not a string: %#v\n", idAny)
				continue
			}

			gameRecord, err := gamesTable.GetRecord(gameID)
			if err != nil {
				fmt.Printf("[WARN] Failed to fetch game %s: %v\n", gameID, err)
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

		fmt.Printf("[INFO] Fetching Hackatime data for user %s\n", cmd.UserID)
		resp, err := http.Get("https://hackatime.hackclub.com/api/summary?user=" + cmd.UserID + "&interval=today")
		if err != nil {
			fmt.Printf("[ERROR] Failed to fetch Hackatime: %v\n", err)
			sendEphemeralSlackMessage(cmd.ResponseURL, "Failed to fetch Hackatime data")
			return
		}
		defer resp.Body.Close()

		data, err := io.ReadAll(resp.Body)
		if err != nil {
			fmt.Printf("[ERROR] Failed to read Hackatime response: %v\n", err)
			sendEphemeralSlackMessage(cmd.ResponseURL, "Failed to read Hackatime data")
			return
		}

		today, err := utils.ParseHackatimeSummary(data)
		if err != nil {
			fmt.Printf("[ERROR] Failed to parse Hackatime data: %v\n", err)
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
		sb.WriteString("===\n")
		sb.WriteString(fmt.Sprintf(":shiba_hey: *Here is your activity for today, <@%s>:*\n", cmd.UserID))
		sb.WriteString("===\n")
		for _, game := range userGames {
			h := game.TotalTimeToday / 3600
			m := (game.TotalTimeToday % 3600) / 60
			sb.WriteString(fmt.Sprintf("\n*%s* _%s_\ntime logged today: %02d:%02dh\n",
				game.Name, game.Description, h, m))
			if game.TotalTimeToday >= 2*3600 {
				sb.WriteString(":yay: you got 2 hours, time to send what you added in #shiba!\n")
			}
		}
		sb.WriteString("\nKeep up the great work! :shiba-sniff:\n")

		sendEphemeralSlackMessage(cmd.ResponseURL, sb.String())
	}(cmd)
}
