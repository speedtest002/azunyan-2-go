package service

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/disgoorg/disgo/discord"
)

const vietqrAPI = "https://api.vietqr.io/v2/banks"

var vietQRHTTPClient = &http.Client{
	Timeout: 10 * time.Second,
}

type Bank struct {
	ID                int    `json:"id"`
	Name              string `json:"name"`
	Code              string `json:"code"`
	Bin               string `json:"bin"`
	ShortName         string `json:"shortName"`
	Logo              string `json:"logo"`
	TransferSupported int    `json:"transferSupported"`
	LookupSupported   int    `json:"lookupSupported"`
}

type vietQRResponse struct {
	Data []Bank `json:"data"`
}

type VietQRService struct {
	Banks   []Bank
	BankMap map[string]*Bank
}

func NewVietQRService() (*VietQRService, error) {
	svc := &VietQRService{
		BankMap: make(map[string]*Bank),
	}

	banks, err := fetchVietQR()
	if err != nil {
		return svc, err
	}

	svc.Banks = banks
	svc.BankMap = make(map[string]*Bank, len(banks)*2)
	for i := range svc.Banks {
		bank := &svc.Banks[i]
		svc.BankMap[strings.ToLower(bank.ShortName)] = bank
		svc.BankMap[strings.ToLower(bank.Code)] = bank
	}

	slog.Info("loaded banks", slog.Int("count", len(banks)))
	return svc, nil
}

func fetchVietQR() ([]Bank, error) {
	resp, err := vietQRHTTPClient.Get(vietqrAPI)
	if err != nil {
		return nil, fmt.Errorf("vietqr request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("vietqr unexpected status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("vietqr read body: %w", err)
	}

	var apiResp vietQRResponse
	err = json.Unmarshal(body, &apiResp)
	if err != nil {
		return nil, fmt.Errorf("vietqr decode json: %w", err)
	}
	return apiResp.Data, nil
}

func BuildQRImageURL(bank, accountNumber string, amount int, description string) string {
	baseURL := fmt.Sprintf("https://img.vietqr.io/image/%s-%s-print.jpg", bank, accountNumber)

	params := url.Values{}
	if amount > 0 {
		params.Set("amount", strconv.Itoa(amount))
	}
	if description != "" {
		params.Set("addInfo", description)
	}

	if encoded := params.Encode(); encoded != "" {
		return baseURL + "?" + encoded
	}
	return baseURL
}

func BuildQREmbed(bank, accountNumber string, amount int, description string, user discord.User) discord.Embed {
	imgURL := BuildQRImageURL(bank, accountNumber, amount, description)
	return discord.NewEmbed().
		WithImage(imgURL).
		WithColor(0x00FFFF).
		WithAuthor(user.EffectiveName(), "", user.EffectiveAvatarURL())
}
