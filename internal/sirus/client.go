package sirus

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

func GetActualRaids() ([]RaidInfo, error) {
	client := &http.Client{}
	req, err := http.NewRequest("GET", "https://sirus.su/api/base/22/character/678938", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
	
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("сервер вернул плохой статус: %d %s", resp.StatusCode, resp.Status)
	}

	bodyBytes, _ := io.ReadAll(resp.Body)

	var res CharacterResponse
	if err := json.Unmarshal(bodyBytes, &res); err != nil {
		return nil, err
	}

	var actual []RaidInfo

	for _, raid := range res.Pve {
		if raid.Actual {
			actual = append(actual, raid)
		}
	}

	return actual, nil
}
