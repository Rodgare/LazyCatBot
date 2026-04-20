package discord

import (
	"fmt"
	"slices"
	"strings"
	"unicode/utf8"
)

// SpecEmojis содержит маппинг ID класса -> Имя спека -> Discord эмодзи.
// Вам нужно будет заменить "ID" на реальные ID эмодзи с вашего сервера.
// Чтобы узнать ID эмодзи, напишите в чат дискорда: \:имя_эмодзи: (с обратным слешем).
var SpecEmojis = map[int]map[string]string{
	1: { // Warrior
		"Arms": "<:WarA:1495884556509778070>",
		"Fury": "<:WarF:1495884541443834067>",
		"Prot": "<:WarP:1495884526369640529>",
	},
	2: { // Paladin
		"Holy":  "<:PaladinH:1495884511798366298>",
		"Prot":  "<:PaladinP:1495884494220034100>",
		"Retri": "<:PaladinR:1495884478235545682>",
	},
	3: { // Hunter
		"BM":   "<:HunterBM:1495884454781124741>",
		"MM":   "<:HunterMM:1495884427706896424>",
		"Surv": "<:HunterS:1495884405674213436>",
	},
	4: { // Rogue
		"Assa":   "<:RogueA:1495884389769285772>",
		"Combat": "<:RogueC:1495884365408768172>",
		"Sub":    "<:RogueS:1495884349868998818>",
	},
	5: { // Priest
		"Disc":   "<:PriestD:1495884324397125662>",
		"Holy":   "<:PriestH:1495884304520183899>",
		"Shadow": "<:PriestS:1495884283787608274>",
	},
	6: { // DK
		"Blood":  "<:DKB:1495884265341059243>",
		"Frost":  "<:DKF:1495884248517837011>",
		"Unholy": "<:DKU:1495884230415220768>",
	},
	7: { // Shaman
		"Ele":   "<:ShamanEl:1495884213218705650>",
		"Enh":   "<:ShamanEnch:1495884197888393318>",
		"Resto": "<:ShamanR:1495884183355130047>",
	},
	8: { // Mage
		"Arcane": "<:MageA:1495884168020623370>",
		"Fire":   "<:MageF:1495884152371937451>",
		"Frost":  "<:MageFr:1495884137377173575>",
	},
	9: { // Warlock
		"Affli":  "<:LockA:1495884122080415865>",
		"Demo":   "<:LockDemon:1495884105936539729>",
		"Destro": "<:LockDestr:1495884089830408473>",
	},
	11: { // Druid
		"Bal":   "<:DruidB:1495884071887306953>",
		"Feral": "<:DruidF:1495884053117665351>",
		"Resto": "<:DruidR:1495884019995377904>",
	},
}

// BuildReportText строит текстовый отчёт для Discord embed.
func BuildReportText(report BossKillReport) (ddBlocks []string, healBlocks []string) {
	var dds, healers []PlayerReport
	for _, p := range report.Players {
		if p.Role == 1 {
			healers = append(healers, p)
		} else {
			dds = append(dds, p)
		}
	}

	slices.SortFunc(dds, func(a, b PlayerReport) int { return b.Dps - a.Dps })
	slices.SortFunc(healers, func(a, b PlayerReport) int { return b.Hps - a.Hps })

	ddBlocks = buildBlocks(dds, true, report.TotalDps)
	healBlocks = buildBlocks(healers, false, report.TotalHps)
	return
}

func buildBlocks(players []PlayerReport, isDD bool, total int) []string {
	if len(players) == 0 {
		return nil
	}

	var blocks []string
	var sb strings.Builder

	for i, p := range players {
		emoji := "❔" // Дефолт, если неизвестно
		if specs, ok := SpecEmojis[p.ClassID]; ok {
			if str, ok2 := specs[p.SpecName]; ok2 && str != "" {
				emoji = str
			}
		}

		val := p.Dps
		if !isDD {
			val = p.Hps
		}

		// Формат: Ранг Ник (Дпс/Хпс) Рейтинг
		// Используем \u2800 для пустого места (Braille Pattern Blank)
		line := fmt.Sprintf("**%d**\u2800\u2800%s**%s** %s `[Топ: спек #%d]`\n",
			i+1, emoji, p.Name, FormatNum(val), p.SpecRank)

		// Discord limit 1024 characters per field value. Safely cut at 1000.
		if utf8.RuneCountInString(sb.String())+utf8.RuneCountInString(line) > 1000 {
			blocks = append(blocks, sb.String())
			sb.Reset()
		}

		sb.WriteString(line)
	}

	if sb.Len() > 0 {
		blocks = append(blocks, sb.String())
	}

	return blocks
}

func FormatNum(n int) string {
	if n >= 1_000_000 {
		return fmt.Sprintf("%.1fM", float64(n)/1_000_000)
	}
	if n >= 1_000 {
		return fmt.Sprintf("%.1fk", float64(n)/1_000)
	}
	return fmt.Sprintf("%d", n)
}
