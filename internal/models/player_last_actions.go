package models

type PlayerLastActions []struct {
	Type    string `json:"type"`
	FightID int    `json:"id"`
	Date    string `json:"datetime"`
}
