package service

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

const traceMoeBase = "https://api.trace.moe/search"

var traceMoeHTTPClient = &http.Client{
	Timeout: 10 * time.Second,
}

type TraceMoeResult struct {
	Anilist    AnilistInfo `json:"anilist"`
	Filename   string      `json:"filename"`
	Episode    any         `json:"episode"`
	Duration   float64     `json:"duration"`
	From       float64     `json:"from"`
	To         float64     `json:"to"`
	At         float64     `json:"at"`
	Similarity float64     `json:"similarity"`
	Video      string      `json:"video"`
	Image      string      `json:"image"`
}

type TraceMoeAPI struct {
	FrameCount int              `json:"frameCount"`
	Error      string           `json:"error"`
	Result     []TraceMoeResult `json:"result"`
}

type AnilistInfo struct {
	ID       int               `json:"id"`
	IDMal    int               `json:"idMal"`
	Title    AnilistInfoTitles `json:"title"`
	Synonyms []string          `json:"synonyms"`
	IsAdult  bool              `json:"isAdult"`
}

type AnilistInfoTitles struct {
	Native  string `json:"native"`
	Romaji  string `json:"romaji"`
	English any    `json:"english"`
}

func FetchTraceMoe(attURL string) ([]byte, error) {
	apiURL := fmt.Sprintf("%s?url=%s&cutBorders&anilistInfo", traceMoeBase, url.QueryEscape(attURL))
	resp, err := traceMoeHTTPClient.Get(apiURL)
	if err != nil {
		return nil, fmt.Errorf("tracemoe request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 413 {
		return nil, fmt.Errorf("tracemoe: file too large (>25MB), status %d", resp.StatusCode)
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("tracemoe unexpected status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("tracemoe read body: %w", err)
	}

	return body, nil
}

func BuildTraceMoeAPI(data []byte) (TraceMoeAPI, error) {
	var result TraceMoeAPI

	if err := json.Unmarshal(data, &result); err != nil {
		return TraceMoeAPI{}, fmt.Errorf("tracemoe decode json: %w", err)
	}

	if result.Error != "" {
		return TraceMoeAPI{}, fmt.Errorf("tracemoe api error: %s", result.Error)
	}

	return result, nil
}
