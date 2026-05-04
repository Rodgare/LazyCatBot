package models

type PlayerProfile struct {
	Player struct {
		ID int `json:"guid"`
	} `json:"character"`
}
