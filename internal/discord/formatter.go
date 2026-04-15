package discord

import (
	"LazyCatBot/internal/sirus"
	"strings"

	"github.com/olekukonko/tablewriter"
)

func FormatTotalTopGuild(nickname string, raids []sirus.RaidInfo) string {
	sb := &strings.Builder{}
	sb.WriteString("```\n")

	table := tablewriter.NewTable(sb)
	
	headers := []any{"№", "Ник"}
	
	for _, r := range raids {
		shortName, err := sirus.GetRaidName(r.MapID, r.Difficulty)
		
		if err != nil {
			headers = append(headers, r.MapName)
		} else {
			headers = append(headers, shortName)
		}
	}
	
	table.Header(headers...)

	data := [][]any{
		{"1", nickname},
	}
	
	for range raids {
		data[0] = append(data[0], " ")
	}

	table.Bulk(data)
	table.Render()

	sb.WriteString("```")
	
	return sb.String()
}