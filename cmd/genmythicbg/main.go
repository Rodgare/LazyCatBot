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

const styleSuffix = ", epic fantasy dungeon loading screen background art in the style of World of Warcraft, dark cinematic atmosphere, rich painterly detail, muted moody color palette, wide panoramic landscape view, no characters, no people, no text, no watermark, no logos, no UI, subtle dark vignette on edges, suitable as dark backdrop behind white text"

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

type challengeInfo struct {
	ChallengeID int    `json:"challengeId"`
	Name        string `json:"name"`
	Icon        string `json:"icon"`
}

type latestRunsResponse struct {
	Data       []json.RawMessage        `json:"data"`
	Challenges map[string]challengeInfo `json:"challenges"`
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

	outDir := flag.String("out", "internal/discord/assets/images/mythic", "output directory")
	realm := flag.String("realm", "x3", "sirus realm")
	baseURL := flag.String("base", "https://sirus.su", "sirus base URL")
	only := flag.String("only", "", "comma-separated challenge IDs to generate")
	skipExisting := flag.Bool("skip-existing", false, "skip challenges that already have an output file")
	aspect := flag.String("aspect", "16:9", "aspect ratio, e.g. 16:9, 2:1")
	quality := flag.String("quality", "high", "quality: auto|low|medium|high|xhigh|max")
	size := flag.String("size", "2K", "size tier or explicit pixels, e.g. 2K or 1536x864")
	delay := flag.Duration("sleep", 3*time.Second, "delay between generations")
	flag.Parse()

	apiKey := os.Getenv("OPENROUTER_API_KEY")
	if apiKey == "" {
		log.Fatal("OPENROUTER_API_KEY is not set")
	}
	model := os.Getenv("OPENROUTER_IMAGE_MODEL")
	if model == "" {
		model = "bytedance-seed/seedream-4.5"
	}

	if err := os.MkdirAll(*outDir, 0o755); err != nil {
		log.Fatalf("create output dir: %v", err)
	}

	challenges, err := fetchChallenges(*baseURL, *realm)
	if err != nil {
		log.Fatalf("fetch challenges: %v", err)
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
				log.Fatalf("invalid -only challenge id %q", part)
			}
			onlyIDs[id] = true
		}
	}

	var targets []challengeInfo
	for _, ch := range challenges {
		if len(onlyIDs) > 0 && !onlyIDs[ch.ChallengeID] {
			continue
		}
		targets = append(targets, ch)
	}

	if len(targets) == 0 {
		log.Fatal("no challenges to generate")
	}

	total := len(targets)
	for i, ch := range targets {
		outPath := filepath.Join(*outDir, fmt.Sprintf("%d.png", ch.ChallengeID))
		if *skipExisting {
			if _, err := os.Stat(outPath); err == nil {
				fmt.Printf("[%d/%d] SKIP %d %s (exists)\n", i+1, total, ch.ChallengeID, ch.Name)
				continue
			}
		}

		fmt.Printf("[%d/%d] %d %s ...\n", i+1, total, ch.ChallengeID, ch.Name)
		raw, err := generateImage(apiKey, model, buildPrompt(ch), *aspect, *quality, *size)
		if err != nil {
			fmt.Printf("  ERROR generate: %v\n", err)
			continue
		}

		img, _, err := image.Decode(bytes.NewReader(raw))
		if err != nil {
			fmt.Printf("  ERROR decode: %v\n", err)
			continue
		}

		bg := coverResize(img, 800, 400)
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

		if i < total-1 {
			time.Sleep(*delay)
		}
	}
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

func buildPrompt(ch challengeInfo) string {
	scene, ok := challengeScenes[ch.ChallengeID]
	if !ok {
		scene = fmt.Sprintf("dark fantasy dungeon interior with ancient stone architecture of %s", ch.Name)
	}
	return scene + styleSuffix
}

func generateImage(apiKey, model, prompt, aspect, quality, size string) ([]byte, error) {
	reqBody := imageGenRequest{
		Model:        model,
		Prompt:       prompt,
		AspectRatio:  aspect,
		Quality:      quality,
		Size:         size,
		OutputFormat: "png",
	}

	payload, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	client := &http.Client{Timeout: 240 * time.Second}
	req, err := http.NewRequest(http.MethodPost, "https://openrouter.ai/api/v1/images/generations", bytes.NewReader(payload))
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