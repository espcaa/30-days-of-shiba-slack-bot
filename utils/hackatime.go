package utils

import (
	"encoding/json"
	"fmt"
)

type TodayCode struct {
	UserID    string
	From      string
	To        string
	Projects  []Project
	Languages []Language
}

type Project struct {
	Key   string
	Total int
}

type Language struct {
	Key   string
	Total int
}

func ParseHackatimeSummary(jsonData []byte) (*TodayCode, error) {
	var raw struct {
		UserID   string `json:"user_id"`
		From     string `json:"from"`
		To       string `json:"to"`
		Projects []struct {
			Key   string `json:"key"`
			Total int    `json:"total"`
		} `json:"projects"`
		Languages []struct {
			Key   string `json:"key"`
			Total int    `json:"total"`
		} `json:"languages"`
	}

	if err := json.Unmarshal(jsonData, &raw); err != nil {
		return nil, fmt.Errorf("failed to parse Hackatime JSON: %w", err)
	}

	today := &TodayCode{
		UserID: raw.UserID,
		From:   raw.From,
		To:     raw.To,
	}

	for _, p := range raw.Projects {
		today.Projects = append(today.Projects, Project{
			Key:   p.Key,
			Total: p.Total,
		})
	}

	for _, l := range raw.Languages {
		today.Languages = append(today.Languages, Language{
			Key:   l.Key,
			Total: l.Total,
		})
	}

	return today, nil
}
