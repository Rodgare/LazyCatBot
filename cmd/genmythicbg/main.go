package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"image"
	"image/draw"
	"image/png"
	_ "image/gif"
	_ "image/jpeg"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
	xdraw "golang.org/x/image/draw"
	_ "golang.org/x/image/webp"
)

const dungeonStyleSuffix = ", epic fantasy dungeon loading screen background art in the style of World of Warcraft, dark cinematic atmosphere, rich painterly detail, muted moody color palette, wide panoramic landscape view, no characters, no people, no text, no watermark, no logos, no UI, subtle dark vignette on edges, suitable as dark backdrop behind white text"

const raidStyleSuffix = ", epic fantasy MMORPG raid background art in the style of World of Warcraft, dark cinematic atmosphere, rich painterly detail, muted moody color palette, tall vertical composition, grand imposing scale, no characters, no people, no text, no watermark, no logos, no UI, darker top area suitable for a title, subtle dark vignette on edges"

var challengeScenes = map[int]string{
	4:  "massive vrykul fortress of black stone and timber carved into a cliff above a frozen fjord in Northrend, gothic longhouse towers, wooden bridges, snow and frost, distant great dragon silhouette, northern lights in the cold night sky",
	5:  "grand demonic citadel gate of the Burning Legion rising over a red wasteland, black iron and jagged fel-green crystal spires, rivers of lava, ash clouds, imposing arched gateway, demonic runes glowing green",
	6:  "submerged naga prison sanctum, grand curved stone arches and walkways above glowing teal water pools, water spouts, crystalline rock formations, soft underwater light rays, eerie calm of a flooded temple",
	8:  "ancient troll fortress buried in a jungle mountain, moss-covered stone ziggurat with wooden gates and tribal totems, giant carved troll face entrance, dense tropical greenery, waterfalls, misty canopy, ominous atmosphere",
	9:  "colossal titan hall with impossibly tall pillars reaching into darkness, arcs of electric lightning crackling between iron pylons and hovering storm clouds, blue-white flashes, metallic architecture of an ancient storm god, dramatic vertical perspective",
	10: "fiery demonic forge citadel interior, massive blood-fed furnace with a glowing molten core, iron chains and bellows, rivers of molten blood and fire, sparks, brutal legion architecture, oppressive red-orange glow",
	11: "ethereal draenei necropolis in a desolate waste, floating teal crystals, tombs of the dead, shattered portals glowing with arcane blue energy, sand and ruins, melancholic ghostly light",
	13: "ancient draenei crypt shrine, rows of carved sarcophagi in shadowy halls, holy blue-violet spirit lights floating among pillars, ghostly spirits, solemn cathedral-like burial chamber, dust motes in beams of light",
}

var raidNames = map[int]string{
	46: "Черный храм",
	45: "Кошмар наяву",
	44: "Битва за гору Хиджал",
	43: "Бронзовое святилище ГЕР",
	42: "Ульдуар 25 ГЕР",
	41: "Бронзовое святилище ОБ",
	40: "Ульдуар 10 ГЕР",
	39: "Змеиное святилище ГЕР",
	38: "ОКО ГЕР",
	37: "Змеиное святилище ОБ",
	36: "ОКО ОБ",
	35: "Тол'Гародская тюрьма 25",
	34: "Тол'Гародская тюрьма 10",
	33: "РС 25 ГЕР",
	32: "РС 10 ГЕР",
	31: "ЦЛК 25 ГЕР",
	30: "ЦЛК 10 ГЕР",
	29: "РС 25 ОБ",
	28: "РС 10 ОБ",
	27: "Зул'Аман",
	26: "ЦЛК 25 ОБ",
	25: "ЦЛК 10 ОБ",
	24: "СА 25",
	23: "СА 10",
	22: "Логово Груула ГЕР",
	21: "Логово Магтеридона ГЕР",
	20: "Каражан ГЕР",
	19: "ИВК 25",
	18: "ИВК 10",
	17: "Логово Груула ОБ",
	16: "Логово Магтеридона ОБ",
	15: "ИК 25 ОБ",
	14: "ИК 10 ОБ",
	13: "ОНЯ 25",
	12: "ОНЯ 10",
	11: "Каражан ОБ",
	10: "Ульдуар 25 ОБ",
	9:  "Ульдуар 10 ОБ",
	8:  "Накс 25 ГЕР",
	7:  "Накс 10 ГЕР",
	6:  "Око вечности 25",
	5:  "Око вечности 10",
	4:  "Накс 25 ОБ",
	3:  "Накс 10 ОБ",
	2:  "ОС 25",
	1:  "ОС 10",
}

var raidScenes = map[int]string{
	31: "frozen citadel of the Lich King on a towering ice spire, frost-covered dark walls, bone and saronite architecture, howling blizzard, glowing blue necromantic light, frostwyrm silhouettes, oppressive northern fortress",
	42: "colossal titan city of Ulduar carved into a mountain, ancient iron and bronze halls, glowing machinery, storm clouds and lightning above the storm pinnacle, cosmic runes, monumental scale",
	33: "ruby dragon sanctum, vast cavern with crimson scales and molten veins, glowing red crystals, draconic lair of the Red Dragonflight, golden throne at the heart, warm ominous glow",
	8:  "floating necropolis of Naxxramas hovering over a cursed plague field, skeletal spires, green necrotic glow, swarms of undead, bone gates, ominous dark fortress",
	38: "shattered sky over the Nexus, blue dragonflight sanctum of Malygos, arcane storms and ley energy, floating islands, cold sapphire light, runes in the air",
	6:  "shattered sky over the Nexus, blue dragonflight sanctum of Malygos, arcane storms and ley energy, floating islands, cold sapphire light, runes in the air",
	39: "submerged serpentine temple of the naga, grand curved arches over glowing water, coral and kelp, the lair of Lady Vashj, eerie teal depths, serpent pillars",
	43: "bronze dragonflight sanctum, shimmering golden portals to time, hourglass motifs, swirling temporal sands, warm bronze glow, mysterious ancient chamber",
	22: "mountain lair of the ogre king Gruul, colossal stone cavern, broken stalactites, ogre banners, dark rocky throne, primal and brutal",
	21: "demon pit lair of Magtheridon, dark ritual chamber, fel chains and hooks, burning green fel energy, jagged obsidian walls, oppressive demonic altar",
	20: "haunted tower of Karazhan rising into a dark sky, gothic windows glowing with arcane light, floating stairs, spectral wisps, Medivh's mysterious citadel, elegant and eerie",
	27: "troll empire temple of Zul'Aman in a dense jungle, stone ziggurat with carved troll faces, war drums, forest troll banners, overgrown ruins, dark ceremonial atmosphere",
	13: "lair of Onyxia deep underground, black dragon cave, glowing lava cracks, charred bones and treasure hoard, fiery orange glow, dragon queen's brood chamber",
	24: "corrupted blood elf sanctuary of the Sunwell, golden and crimson halls, radiant Sunwell glow turning fel-green, elegant elven architecture overrun by corruption",
	35: "imposing Tol'Barad prison fortress on a rocky coast, tall grey walls, guard towers, mist and sea spray, fortified gates, grim military atmosphere",
	2:  "obsidian dragon sanctum, dark volcanic cavern, black stone and molten magma, draconic pillars, glowing embers, sinister dragonflight lair",
	46: "demonic temple-citadel of the Black Temple, jagged fel-green spires, obsidian demonic architecture, rivers of fel energy, shattered sky, Illidan's dark sanctum, oppressive and imposing",
	45: "corrupted Emerald Dream nightmare realm, twisted gnarled trees, glowing green-black spores, fog, ancient dream magic turning dark, surreal ominous forest",
	44: "ancient night elf mountain sanctuary under siege, burning trees and demonic legion forces in the distance, the World Tree glowing above, fire and embers, epic battlefield, dramatic scale",
}

// raidFamilies groups raid orders that share the same dungeon into families.
// The first ID is the canonical one used for generation; the rest are copied.
var raidFamilies = [][]int{
	{31, 30, 26, 25}, // ЦЛК (Icecrown Citadel)
	{42, 40, 10, 9},  // Ульдуар (Ulduar)
	{33, 32, 29, 28}, // РС (Ruby Sanctum)
	{8, 7, 4, 3},     // Наксрамас (Naxxramas)
	{38, 36},         // ОКО (Eye of Eternity)
	{6, 5},           // Око вечности (Eye of Eternity 10/25)
	{39, 37},         // Змеиное святилище (Serpentshrine Cavern)
	{43, 41},         // Бронзовое святилище
	{22, 17},         // Логово Груула (Gruul's Lair)
	{21, 16},         // Логово Магтеридона (Magtheridon's Lair)
	{20, 11},         // Каражан (Karazhan)
	{19, 18},         // ИВК
	{15, 14},         // ИК
	{13, 12},         // ОНЯ (Onyxia)
	{24, 23},         // СА (Sunwell Plateau)
	{35, 34},         // Тол'Гарод (Tol'Barad Prison)
	{2, 1},           // ОС
	{46},             // Черный храм (Black Temple)
	{45},             // Кошмар наяву (Nightmare)
	{44},             // Битва за гору Хиджал (Hyjal Summit)
	{27},             // Зул'Аман (Zul'Aman)
}

type challengeInfo struct {
	ChallengeID int    `json:"challengeId"`
	Name        string `json:"name"`
	Icon        string `json:"icon"`
}

type latestRunsResponse struct {
	Data       []json.RawMessage        `json:"data"`
	Challenges map[string]challengeInfo `json:"challenges"`
}

type target struct {
	ID      int
	Name    string
	Aliases []int // other raid orders that share this background image
}

type imageGenRequest struct {
	Model        string `json:"model"`
	Prompt       string `json:"prompt"`
	AspectRatio  string `json:"aspect_ratio,omitempty"`
	Quality      string `json:"quality,omitempty"`
	Size         string `json:"size,omitempty"`
	OutputFormat string `json:"output_format,omitempty"`
}

type imageGenResponse struct {
	Data []struct {
		B64JSON   string `json:"b64_json"`
		MediaType string `json:"media_type"`
	} `json:"data"`
	Usage struct {
		Cost float64 `json:"cost"`
	} `json:"usage"`
}

func main() {
	_ = godotenv.Load()

	genType := flag.String("type", "mythic", "type of backgrounds: mythic or raid")
	outDir := flag.String("out", "", "output directory (default per type)")
	realm := flag.String("realm", "x3", "sirus realm")
	baseURL := flag.String("base", "https://sirus.su", "sirus base URL")
	only := flag.String("only", "", "comma-separated IDs to generate")
	skipExisting := flag.Bool("skip-existing", false, "skip targets that already have an output file")
	aspect := flag.String("aspect", "", "aspect ratio (default per type: mythic 16:9, raid 3:4)")
	quality := flag.String("quality", "", "quality, only sent if set (e.g. high)")
	size := flag.String("size", "", "size tier, only sent if set (e.g. 2K)")
	outH := flag.Int("height", 0, "output height in px (default per type: mythic 400, raid 1200)")
	delay := flag.Duration("sleep", 3*time.Second, "delay between generations")
	flag.Parse()

	apiKey := os.Getenv("OPENROUTER_API_KEY")
	if apiKey == "" {
		log.Fatal("OPENROUTER_API_KEY is not set")
	}
	model := os.Getenv("OPENROUTER_IMAGE_MODEL")
	if model == "" {
		model = "bytedance-seed/seedream-5-0-flash"
	}

	resolvedOut := *outDir
	resolvedAspect := *aspect
	resolvedOutH := *outH
	var targets []target

	switch *genType {
	case "raid":
		if resolvedOut == "" {
			resolvedOut = "internal/discord/assets/images/raid"
		}
		if resolvedAspect == "" {
			resolvedAspect = "3:4"
		}
		if resolvedOutH == 0 {
			resolvedOutH = 1200
		}
		for _, fam := range raidFamilies {
			canonical := fam[0]
			var aliases []int
			if len(fam) > 1 {
				aliases = fam[1:]
			}
			targets = append(targets, target{ID: canonical, Name: raidNames[canonical], Aliases: aliases})
		}
	default: // mythic
		if resolvedOut == "" {
			resolvedOut = "internal/discord/assets/images/mythic"
		}
		if resolvedAspect == "" {
			resolvedAspect = "16:9"
		}
		if resolvedOutH == 0 {
			resolvedOutH = 400
		}
		challenges, err := fetchChallenges(*baseURL, *realm)
		if err != nil {
			log.Fatalf("fetch challenges: %v", err)
		}
		for _, ch := range challenges {
			targets = append(targets, target{ID: ch.ChallengeID, Name: ch.Name})
		}
	}

	onlyIDs := map[int]bool{}
	if *only != "" {
		for _, part := range strings.Split(*only, ",") {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}
			id, err := strconv.Atoi(part)
			if err != nil {
				log.Fatalf("invalid -only id %q", part)
			}
			onlyIDs[id] = true
		}
	}
	if len(onlyIDs) > 0 {
		filtered := targets[:0]
		for _, t := range targets {
			if onlyIDs[t.ID] {
				filtered = append(filtered, t)
				continue
			}
			for _, a := range t.Aliases {
				if onlyIDs[a] {
					filtered = append(filtered, t)
					break
				}
			}
		}
		targets = filtered
	}

	if len(targets) == 0 {
		log.Fatal("no targets to generate")
	}

	if err := os.MkdirAll(resolvedOut, 0o755); err != nil {
		log.Fatalf("create output dir: %v", err)
	}

	total := len(targets)
	for i, t := range targets {
		outPath := filepath.Join(resolvedOut, fmt.Sprintf("%d.png", t.ID))
		if *skipExisting {
			if _, err := os.Stat(outPath); err == nil {
				fmt.Printf("[%d/%d] SKIP %d %s (exists)\n", i+1, total, t.ID, t.Name)
				ensureAliases(t, resolvedOut, outPath)
				continue
			}
		}

		fmt.Printf("[%d/%d] %d %s ...\n", i+1, total, t.ID, t.Name)
		raw, err := generateImage(apiKey, model, buildPrompt(*genType, t.ID, t.Name), resolvedAspect, *quality, *size)
		if err != nil {
			fmt.Printf("  ERROR generate: %v\n", err)
			continue
		}

		img, _, err := image.Decode(bytes.NewReader(raw))
		if err != nil {
			fmt.Printf("  ERROR decode: %v\n", err)
			continue
		}

		bg := coverResize(img, 800, resolvedOutH)
		f, err := os.Create(outPath)
		if err != nil {
			fmt.Printf("  ERROR create: %v\n", err)
			continue
		}
		if err := png.Encode(f, bg); err != nil {
			f.Close()
			fmt.Printf("  ERROR encode: %v\n", err)
			continue
		}
		f.Close()
		fmt.Printf("  OK -> %s\n", outPath)

		for _, alias := range t.Aliases {
			aliasPath := filepath.Join(resolvedOut, fmt.Sprintf("%d.png", alias))
			if err := copyFile(outPath, aliasPath); err != nil {
				fmt.Printf("  ERROR copy %d: %v\n", alias, err)
				continue
			}
			fmt.Printf("  copy -> %d\n", alias)
		}

		if i < total-1 {
			time.Sleep(*delay)
		}
	}
}

func ensureAliases(t target, outDir, canonicalPath string) {
	for _, alias := range t.Aliases {
		aliasPath := filepath.Join(outDir, fmt.Sprintf("%d.png", alias))
		if _, err := os.Stat(aliasPath); err == nil {
			continue
		}
		if err := copyFile(canonicalPath, aliasPath); err != nil {
			fmt.Printf("  ERROR copy %d: %v\n", alias, err)
			continue
		}
		fmt.Printf("  copy -> %d\n", alias)
	}
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}

var raidPromptNames = map[int]string{
	19: "an ancient fortress citadel of the Horde and Alliance",
	15: "a sunken ancient temple citadel",
}

func buildPrompt(genType string, id int, name string) string {
	if genType == "raid" {
		scene, ok := raidScenes[id]
		if !ok {
			promptName, ok2 := raidPromptNames[id]
			if !ok2 {
				promptName = name
			}
			scene = fmt.Sprintf("epic dark fantasy raid stronghold and battlefield of %s", promptName)
		}
		return scene + raidStyleSuffix
	}

	scene, ok := challengeScenes[id]
	if !ok {
		scene = fmt.Sprintf("dark fantasy dungeon interior with ancient stone architecture of %s", name)
	}
	return scene + dungeonStyleSuffix
}

func fetchChallenges(baseURL, realm string) ([]challengeInfo, error) {
	endpoint := fmt.Sprintf("/api/base/%s/leaderboard/challenge/latest-runs", realm)
	client := &http.Client{Timeout: 60 * time.Second}

	baseURLs := []string{baseURL}
	if !strings.Contains(baseURL, "sirus.su") {
		baseURLs = append(baseURLs, "https://sirus.su")
	}

	var lastErr error
	for _, base := range baseURLs {
		url := base + endpoint
		req, err := http.NewRequest(http.MethodGet, url, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) LazyCatBot/1.0")
		req.Header.Set("Accept", "application/json")

		resp, err := client.Do(req)
		if err != nil {
			lastErr = err
			continue
		}

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			lastErr = fmt.Errorf("sirus status %d: %s", resp.StatusCode, truncate(string(body), 300))
			continue
		}

		var lr latestRunsResponse
		if err := json.NewDecoder(resp.Body).Decode(&lr); err != nil {
			resp.Body.Close()
			return nil, err
		}
		resp.Body.Close()

		ids := make([]int, 0, len(lr.Challenges))
		for k := range lr.Challenges {
			id, err := strconv.Atoi(k)
			if err != nil {
				continue
			}
			ids = append(ids, id)
		}
		sort.Ints(ids)

		challenges := make([]challengeInfo, 0, len(ids))
		for _, id := range ids {
			challenges = append(challenges, lr.Challenges[strconv.Itoa(id)])
		}
		return challenges, nil
	}
	return nil, lastErr
}

func generateImage(apiKey, model, prompt, aspect, quality, size string) ([]byte, error) {
	data, err := attemptImageGen(apiKey, model, prompt, aspect, quality, size)
	if err == nil {
		return data, nil
	}

	// Some providers reject aspect_ratio / extra params — retry with the
	// minimal body (model + prompt only).
	fmt.Printf("  WARN full params failed (%v); retrying with minimal body\n", err)
	return attemptImageGen(apiKey, model, prompt, "", "", "")
}

func attemptImageGen(apiKey, model, prompt, aspect, quality, size string) ([]byte, error) {
	reqBody := imageGenRequest{
		Model:       model,
		Prompt:      prompt,
		AspectRatio: aspect,
	}
	if quality != "" {
		reqBody.Quality = quality
	}
	if size != "" {
		reqBody.Size = size
	}

	payload, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	client := &http.Client{Timeout: 240 * time.Second}
	req, err := http.NewRequest(http.MethodPost, "https://openrouter.ai/api/v1/images", bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	var lastErr error
	for attempt := 1; attempt <= 3; attempt++ {
		resp, err := client.Do(req)
		if err != nil {
			lastErr = err
			time.Sleep(3 * time.Second)
			continue
		}

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			lastErr = err
			continue
		}

		if resp.StatusCode != http.StatusOK {
			lastErr = fmt.Errorf("openrouter status %d (attempt %d): %s", resp.StatusCode, attempt, truncate(string(body), 400))
			time.Sleep(3 * time.Second)
			continue
		}

		var gen imageGenResponse
		if err := json.Unmarshal(body, &gen); err != nil {
			return nil, err
		}
		if len(gen.Data) == 0 || gen.Data[0].B64JSON == "" {
			return nil, fmt.Errorf("no image in response")
		}

		data, err := base64.StdEncoding.DecodeString(gen.Data[0].B64JSON)
		if err != nil {
			return nil, err
		}

		fmt.Printf("  [cost $%.4f]\n", gen.Usage.Cost)
		return data, nil
	}
	return nil, lastErr
}

func coverResize(img image.Image, w, h int) image.Image {
	srcW := img.Bounds().Dx()
	srcH := img.Bounds().Dy()

	srcRatio := float64(srcW) / float64(srcH)
	targetRatio := float64(w) / float64(h)

	var cropW, cropH int
	if srcRatio > targetRatio {
		cropH = srcH
		cropW = int(float64(srcH) * targetRatio)
	} else {
		cropW = srcW
		cropH = int(float64(srcW) / targetRatio)
	}
	if cropW < 1 {
		cropW = 1
	}
	if cropH < 1 {
		cropH = 1
	}

	offsetX := (srcW - cropW) / 2
	offsetY := (srcH - cropH) / 2
	sp := img.Bounds().Min.Add(image.Pt(offsetX, offsetY))

	sub := image.NewRGBA(image.Rect(0, 0, cropW, cropH))
	draw.Draw(sub, sub.Bounds(), img, sp, draw.Src)

	dst := image.NewRGBA(image.Rect(0, 0, w, h))
	xdraw.ApproxBiLinear.Scale(dst, dst.Bounds(), sub, sub.Bounds(), xdraw.Over, nil)
	return dst
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}