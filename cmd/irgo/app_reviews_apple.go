package main

import (
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strings"
	"time"
)

// ---------------------------------------------------------------------------
// Apple (iOS + Mac App Store) — official App Store Connect API.
//
// ES256 JWT auth with an App Store Connect API key (.p8). The key must have
// Customer Reviews access (App Store Connect → Users and Access → API keys).
// Both stores use the same API — only the numeric app id differs (the number
// in the app's apps.apple.com URL: apps.apple.com/app/id<THIS NUMBER>).
// Replying uses the Review Responses endpoint.
//
// Config (irgo.package.toml → [reviews]):
//
//	ios_app_id = ""       # numeric iOS App Store id
//	mac_app_id = ""       # numeric Mac App Store id
//	ios_key_id = ""       # App Store Connect API key id
//	ios_issuer_id = ""    # App Store Connect API issuer id
//	ios_private_key = ""  # path to the .p8 private key
// ---------------------------------------------------------------------------

func reviewsApple(platform string, limit int, onlyNew bool, replyTo, replyText string) error {
	if err := writeDefaultPackageConfig(); err != nil {
		return err
	}
	store := "reviews-apple-ios"
	if platform == "mac" {
		store = "reviews-apple-mac"
	}
	if err := ensureStoreConfig(store); err != nil {
		return err
	}
	cfg := parsePackageConfig()
	appID := cfg.ReviewsIOSAppID
	if appID == "" {
		appID = os.Getenv("IRGO_IOS_APP_ID")
	}
	if platform == "mac" {
		appID = cfg.ReviewsMacAppID
		if appID == "" {
			appID = os.Getenv("IRGO_MAC_APP_ID")
		}
	}
	if appID == "" {
		return fmt.Errorf("set [reviews] %s_app_id in %s (numeric App Store id from the app's apps.apple.com URL, e.g. apps.apple.com/app/id123456789)", platform, packageConfigFile)
	}
	token, err := ascToken(cfg)
	if err != nil {
		return err
	}
	if replyTo != "" {
		if replyText == "" {
			return fmt.Errorf("--reply requires --text")
		}
		return ascReplyReview(appID, replyTo, replyText, token)
	}

	fmt.Printf("Fetching %s App Store reviews (app id %s)...\n", platform, appID)
	reviews, err := fetchASReviews(appID, limit, token)
	if err != nil {
		return err
	}
	if len(reviews) == 0 {
		fmt.Println("No reviews found.")
		return nil
	}

	st := loadReviewsState()
	prev := st.IOS
	if platform == "mac" {
		prev = st.Mac
	}
	shown := 0
	for _, r := range reviews {
		if onlyNew && prev != "" && r.createdDate <= prev {
			continue
		}
		printASReview(r, platform)
		shown++
	}
	if onlyNew {
		newest := reviews[0].createdDate
		if prev == "" || newest > prev {
			if platform == "mac" {
				st.Mac = newest
			} else {
				st.IOS = newest
			}
			if err := saveReviewsState(st); err != nil {
				return err
			}
		}
		if shown == 0 {
			fmt.Println("No new reviews since last check.")
		}
	}
	fmt.Printf("%d review(s) shown.\n", shown)
	return nil
}

type asReview struct {
	id          string
	rating      int
	title       string
	body        string
	nickname    string
	territory   string
	appVersion  string
	createdDate string // ISO8601
	hasReply    bool
}

func printASReview(r asReview, platform string) {
	stars := strings.Repeat("★", r.rating) + strings.Repeat("☆", 5-r.rating)
	name := r.nickname
	if name == "" {
		name = "anonymous"
	}
	fmt.Printf("★ %s  %s — %s\n", stars, shortDate(r.createdDate), name)
	fmt.Printf("  id: %s\n", r.id)
	if r.territory != "" {
		fmt.Printf("  territory: %s\n", r.territory)
	}
	if r.appVersion != "" {
		fmt.Printf("  version %s\n", r.appVersion)
	}
	if r.title != "" {
		fmt.Printf("  %s\n", r.title)
	}
	if r.body != "" {
		fmt.Printf("  %s\n", r.body)
	}
	if r.hasReply {
		fmt.Println("  ↳ already replied")
	} else {
		fmt.Printf("  ↳ no reply yet — `irgo reviews %s --reply %s --text \"...\"`\n", platform, r.id)
	}
	fmt.Println()
}

type asReviewsResp struct {
	Data []struct {
		ID         string `json:"id"`
		Attributes struct {
			Rating         int    `json:"rating"`
			Title          string `json:"title"`
			Body           string `json:"body"`
			ReviewNickname string `json:"reviewNickname"`
			Territory      string `json:"territory"`
			CreatedDate    string `json:"createdDate"`
			AppVersion     string `json:"appVersion"`
		} `json:"attributes"`
		Relationships struct {
			CustomerReviewResponses struct {
				Data []struct {
					ID string `json:"id"`
				} `json:"data"`
			} `json:"customerReviewResponses"`
		} `json:"relationships"`
	} `json:"data"`
}

func fetchASReviews(appID string, limit int, token string) ([]asReview, error) {
	if limit > 200 {
		limit = 200
	}
	u := fmt.Sprintf("https://api.appstoreconnect.apple.com/v1/apps/%s/customerReviews?limit=%d&sort=-createdDate&include=customerReviewResponses", url.PathEscape(appID), limit)
	body, err := ascGet(u, token)
	if err != nil {
		return nil, err
	}
	var resp asReviewsResp
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("parsing App Store Connect reviews: %w", err)
	}
	var out []asReview
	for _, d := range resp.Data {
		out = append(out, asReview{
			id:          d.ID,
			rating:      d.Attributes.Rating,
			title:       d.Attributes.Title,
			body:        d.Attributes.Body,
			nickname:    d.Attributes.ReviewNickname,
			territory:   d.Attributes.Territory,
			appVersion:  d.Attributes.AppVersion,
			createdDate: d.Attributes.CreatedDate,
			hasReply:    len(d.Relationships.CustomerReviewResponses.Data) > 0,
		})
	}
	// Newest first (the API sorts by -createdDate, but be defensive).
	sort.SliceStable(out, func(i, j int) bool { return out[i].createdDate > out[j].createdDate })
	return out, nil
}

// ascReplyReview posts a response to a customer review (Review Responses API).
func ascReplyReview(appID, reviewID, text, token string) error {
	u := fmt.Sprintf("https://api.appstoreconnect.apple.com/v1/apps/%s/customerReviews/%s/responses", url.PathEscape(appID), url.PathEscape(reviewID))
	payload, _ := json.Marshal(map[string]any{
		"data": map[string]any{
			"type":       "customerReviewResponse",
			"attributes": map[string]any{"responseBody": text},
		},
	})
	req, err := http.NewRequest("POST", u, strings.NewReader(string(payload)))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode == http.StatusForbidden {
		return fmt.Errorf("App Store Connect API 403 — your API key needs Customer Reviews access (App Store Connect → Users and Access → API keys)")
	}
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return fmt.Errorf("App Store Connect response API %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	fmt.Printf("Response posted to review %s.\n", reviewID)
	return nil
}

// ascToken builds a short-lived App Store Connect API token (ES256 JWT) from
// the [reviews] config (ios_key_id / ios_issuer_id / ios_private_key .p8 path).
func ascToken(cfg packageConfig) (string, error) {
	keyID := cfg.ReviewsIOSKeyID
	if keyID == "" {
		keyID = os.Getenv("IRGO_ASC_KEY_ID")
	}
	issuer := cfg.ReviewsIOSIssuerID
	if issuer == "" {
		issuer = os.Getenv("IRGO_ASC_ISSUER_ID")
	}
	p8 := cfg.ReviewsIOSPrivateKey
	if p8 == "" {
		p8 = os.Getenv("IRGO_ASC_PRIVATE_KEY")
	}
	if keyID == "" || issuer == "" || p8 == "" {
		return "", fmt.Errorf("set [reviews] ios_key_id, ios_issuer_id and ios_private_key in %s (App Store Connect API key — see `irgo package setup`; the key needs Customer Reviews access)", packageConfigFile)
	}
	key, err := parseECP8Key(expandHome(p8))
	if err != nil {
		return "", err
	}
	now := time.Now()
	header := map[string]string{"alg": "ES256", "kid": keyID, "typ": "JWT"}
	claims := map[string]any{
		"iss": issuer,
		"iat": now.Unix(),
		"exp": now.Add(20 * time.Minute).Unix(),
		"aud": "appstoreconnect-v1",
	}
	h, _ := json.Marshal(header)
	c, _ := json.Marshal(claims)
	signingInput := b64url(h) + "." + b64url(c)
	digest := sha256.Sum256([]byte(signingInput))
	r, s, err := ecdsa.Sign(rand.Reader, key, digest[:])
	if err != nil {
		return "", fmt.Errorf("signing App Store Connect JWT: %w", err)
	}
	// ES256 JWT signature is raw r||s (32 bytes each), not ASN.1 DER.
	sig := make([]byte, 64)
	r.FillBytes(sig[:32])
	s.FillBytes(sig[32:])
	return signingInput + "." + b64url(sig), nil
}

func parseECP8Key(path string) (*ecdsa.PrivateKey, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", path, err)
	}
	block, _ := pem.Decode(data)
	if block == nil {
		return nil, fmt.Errorf("no PEM block in %s (expected an App Store Connect .p8 EC key)", path)
	}
	k, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parsing %s (PKCS8 EC key required): %w", path, err)
	}
	ec, ok := k.(*ecdsa.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("%s is not an EC P-256 key (App Store Connect keys are P-256)", path)
	}
	return ec, nil
}

// ascGet performs an authenticated GET against the App Store Connect API,
// with a friendly error when the key lacks Customer Reviews access (403).
func ascGet(u, token string) ([]byte, error) {
	req, err := http.NewRequest("GET", u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode == http.StatusForbidden {
		return nil, fmt.Errorf("App Store Connect API 403 — your API key needs Customer Reviews access (App Store Connect → Users and Access → API keys)")
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("App Store Connect API %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	return body, nil
}
