package utils

import (
	"os"

	"github.com/mehanizm/airtable"
)

// everything airtable related

func CreateAirtableClient() airtable.Client {
	var token = os.Getenv("AIRTABLE_API_KEY")
	client := airtable.NewClient(token)
	return *client
}

func GetTable(client airtable.Client, baseID string, tableName string) *airtable.Table {
	table := client.GetTable(baseID, tableName)
	return table
}
