package structs

import "github.com/mehanizm/airtable"

type SlackBot struct {
	AirtableClient airtable.Client
}

type SlashCommand struct {
	Token       string
	TeamID      string
	TeamDomain  string
	UserID      string
	UserName    string
	ChannelID   string
	ChannelName string
	Command     string
	Text        string
	ResponseURL string
	TriggerID   string
}
