package models

type MythicRuns struct {
	Data []struct {
		ID      int `json:"id"`
		Members []struct {
			MemberGUID int    `json:"memberGuid"`
			Name       string `json:"name"`
		} `json:"members"`
		Position int `json:"position"`
	} `json:"data"`
}
