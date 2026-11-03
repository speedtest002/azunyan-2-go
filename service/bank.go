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

const (
	vietQRAPI                = "https://api.vietqr.io/v2/banks"
	vietQRDeepLinkAPIAndroid = "https://api.vietqr.io/v2/android-app-deeplinks"
	vietQRDeepLinkAPIIOS     = "https://api.vietqr.io/v2/ios-app-deeplinks"
	vietQRDeepLinkBaseURL    = "https://dl.vietqr.io/pay"
	vietQRDefaultRedirectURL = "https://www.vietqr.io/deeplink"
	vietQRImageBaseURL       = "https://img.vietqr.io/image"
	imageProxyBaseURL        = "https://wsrv.nl/"
)

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

type AppDeepLink struct {
	AppId          string `json:"appId"`
	AppLogo        string `json:"appLogo"`
	AppName        string `json:"appName"`
	BankName       string `json:"bankName"`
	MonthlyInstall int    `json:"monthlyInstall"`
	Deeplink       string `json:"deeplink"`
}

type vietQRResponse struct {
	Data []Bank `json:"data"`
}

type vietQRDeepLinkResponse struct {
	Apps []AppDeepLink `json:"apps"`
}

type VietQRService struct {
	Banks            []Bank
	IOSDeepLinks     []AppDeepLink
	AndroidDeepLinks []AppDeepLink
	BankMap          map[string]*Bank
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

	androidDeepLinks, err := fetchVietQRDeepLink(vietQRDeepLinkAPIAndroid)
	if err != nil {
		return svc, err
	}
	svc.AndroidDeepLinks = androidDeepLinks
	slog.Info("loaded deeplink for android", slog.Int("count", len(androidDeepLinks)))

	iOSDeepLinks, err := fetchVietQRDeepLink(vietQRDeepLinkAPIIOS)
	if err != nil {
		return svc, err
	}
	svc.IOSDeepLinks = iOSDeepLinks
	slog.Info("loaded deeplink for ios", slog.Int("count", len(iOSDeepLinks)))

	return svc, nil
}

func fetchVietQR() ([]Bank, error) {
	resp, err := vietQRHTTPClient.Get(vietQRAPI)
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

func fetchVietQRDeepLink(endpoint string) ([]AppDeepLink, error) {
	resp, err := vietQRHTTPClient.Get(endpoint)
	if err != nil {
		return nil, fmt.Errorf("vietqr deeplink request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("vietqr deeplink unexpected status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("vietqr deeplink read body: %w", err)
	}

	var dlResponse vietQRDeepLinkResponse
	err = json.Unmarshal(body, &dlResponse)
	if err != nil {
		return nil, fmt.Errorf("vietqr decode json: %w", err)
	}

	return dlResponse.Apps, nil
}

func BuildQRImageURL(bank, accountNumber, description string, amount int) string {
	svgURL := fmt.Sprintf("%s/%s-%s-print.svg", vietQRImageBaseURL, bank, accountNumber)

	params := url.Values{}
	if amount > 0 {
		params.Set("amount", strconv.Itoa(amount))
	}
	if strings.TrimSpace(description) != "" {
		params.Set("addInfo", description)
	}

	if encoded := params.Encode(); encoded != "" {
		svgURL += "?" + encoded
	}

	return fmt.Sprintf("%s?url=%s&output=png", imageProxyBaseURL, url.QueryEscape(svgURL))
}

// BuildBaseDeepLinkURL builds the VietQR payment link containing recipient info
// doc: https://www.vietqr.io/deeplink
func BuildBaseDeepLinkURL(toBank, accountNumber, description string, amount int) string {
	params := url.Values{}
	params.Set("ba", fmt.Sprintf("%s@%s", accountNumber, strings.ToLower(toBank)))
	params.Set("url", vietQRDefaultRedirectURL)

	if amount > 0 {
		params.Set("am", strconv.Itoa(amount))
	}
	if strings.TrimSpace(description) != "" {
		params.Set("tn", description)
	}

	return fmt.Sprintf("%s?%s", vietQRDeepLinkBaseURL, params.Encode())
}

// WithDeepLinkApp attaches the sender's bank `app` parameter to a base deeplink URL.
func WithDeepLinkApp(baseDeepLink, fromBank string) (string, error) {
	u, err := url.Parse(baseDeepLink)
	if err != nil {
		return "", fmt.Errorf("service.WithDeepLinkApp: %w", err)
	}
	q := u.Query()
	q.Set("app", strings.ToLower(fromBank))
	u.RawQuery = q.Encode()
	return u.String(), nil
}

func BankAutofillSupport() {

}

func BuildQREmbed(bank, accountNumber string, amount int, description string, user discord.User) discord.Embed {
	imgURL := BuildQRImageURL(bank, accountNumber, description, amount)
	baseDeepLink := BuildBaseDeepLinkURL(bank, accountNumber, description, amount)

	return discord.NewEmbed().
		WithImage(imgURL).
		WithColor(0x00FFFF).
		WithAuthor(user.EffectiveName(), baseDeepLink, user.EffectiveAvatarURL())
}

func DeeplinkButton() discord.ButtonComponent {
	return discord.NewSuccessButton("Chuyển tiền ngay", "/bank/deeplink")
}
