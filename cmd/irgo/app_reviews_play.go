package main

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

// ---------------------------------------------------------------------------
// Android — Google Play Developer API (service-account JWT, stdlib only).
// Lets a dev list reviews AND reply from the terminal.
//
// Config (irgo.package.toml → [reviews]):
//
//	android_package = ""         # e.g. "com.example.myapp"
//	android_service_account = "" # path to a Play service-account JSON
// ---------------------------------------------------------------------------

const playScope = "https://www.googleapis.com/auth/androidpublisher"

func reviewsAndroid(limit int, onlyNew bool, replyTo, replyText string) error {
	if err := writeDefaultPackageConfig(); err != nil {
		return err
	}
	if err := ensureStoreConfig("reviews-android"); err != nil {
		return err
	}
	cfg := parsePackageConfig()
	pkg := cfg.ReviewsAndroidPackage
	if pkg == "" {
		pkg = os.Getenv("IRGO_ANDROID_PACKAGE")
	}
	sa := cfg.ReviewsAndroidServiceAcc
	if sa == "" {
		sa = os.Getenv("IRGO_PLAY_SERVICE_ACCOUNT")
	}
	if pkg == "" {
		return fmt.Errorf("set [reviews] android_package in %s (your Play package name, e.g. com.example.myapp)", packageConfigFile)
	}
	if sa == "" {
		return fmt.Errorf("set [reviews] android_service_account in %s (path to a Play Developer API service-account JSON)", packageConfigFile)
	}
	if replyTo != "" {
		if replyText == "" {
			return fmt.Errorf("--reply requires --text")
		}
		return playReplyReview(pkg, sa, replyTo, replyText)
	}

	fmt.Printf("Fetching Google Play reviews (%s)...\n", pkg)
	token, err := playAccessToken(sa)
	if err != nil {
		return err
	}
	reviews, err := fetchPlayReviews(pkg, token, limit)
	if err != nil {
		return err
	}
	if len(reviews) == 0 {
		fmt.Println("No reviews found.")
		return nil
	}

	st := loadReviewsState()
	shown := 0
	for _, r := range reviews {
		if onlyNew && st.Android > 0 && r.seconds() <= st.Android {
			continue
		}
		printPlayReview(r)
		shown++
	}
	if onlyNew {
		var newest int64
		for _, r := range reviews {
			if r.seconds() > newest {
				newest = r.seconds()
			}
		}
		if newest > st.Android {
			st.Android = newest
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

type playReview struct {
	ReviewID   string `json:"reviewId"`
	AuthorName string `json:"authorName"`
	Comment    struct {
		Text             string `json:"text"`
		StarRating       int    `json:"starRating"`
		ReviewerLanguage string `json:"reviewerLanguage"`
	} `json:"comment"`
	LastUpdate struct {
		Seconds string `json:"seconds"`
		Nanos   int    `json:"nanos"`
	} `json:"lastUpdate"`
	ReplyResult struct {
		LastEdited struct {
			Seconds string `json:"seconds"`
		} `json:"lastEdited"`
		ReplyText string `json:"replyText"`
	} `json:"replyResult"`
}

func (r playReview) seconds() int64 {
	s, _ := strconv.ParseInt(r.LastUpdate.Seconds, 10, 64)
	return s
}

type playReviewsResp struct {
	Reviews []playReview `json:"reviews"`
}

func fetchPlayReviews(pkg, token string, limit int) ([]playReview, error) {
	if limit > 100 {
		limit = 100
	}
	u := fmt.Sprintf("https://androidpublisher.googleapis.com/androidpublisher/v3/applications/%s/reviews?maxResults=%d", url.PathEscape(pkg), limit)
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
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Play API %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	var out playReviewsResp
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, fmt.Errorf("parsing Play reviews: %w", err)
	}
	return out.Reviews, nil
}

func printPlayReview(r playReview) {
	stars := strings.Repeat("★", r.Comment.StarRating) + strings.Repeat("☆", 5-r.Comment.StarRating)
	fmt.Printf("★ %s  %s — %s\n", stars, shortDate(epochToRFC3339(r.seconds())), r.AuthorName)
	fmt.Printf("  id: %s\n", r.ReviewID)
	if r.Comment.Text != "" {
		fmt.Printf("  %s\n", r.Comment.Text)
	}
	if r.ReplyResult.ReplyText != "" {
		fmt.Printf("  ↳ replied: %s\n", r.ReplyResult.ReplyText)
	} else {
		fmt.Println("  ↳ no reply yet — `irgo reviews android --reply <id> --text \"...\"`")
	}
	fmt.Println()
}

func playReplyReview(pkg, sa, reviewID, text string) error {
	token, err := playAccessToken(sa)
	if err != nil {
		return err
	}
	u := fmt.Sprintf("https://androidpublisher.googleapis.com/androidpublisher/v3/applications/%s/reviews/%s:reply", url.PathEscape(pkg), url.PathEscape(reviewID))
	payload, _ := json.Marshal(map[string]string{"replyText": text, "reviewId": reviewID})
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
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Play reply API %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	var res struct {
		ReplyText string `json:"replyText"`
	}
	_ = json.Unmarshal(body, &res)
	fmt.Printf("Reply posted to review %s: %s\n", reviewID, res.ReplyText)
	return nil
}

// playAccessToken exchanges a service-account JSON for an OAuth2 bearer token
// via a self-signed JWT (client assertion) — no google.golang.org/api needed.
func playAccessToken(saPath string) (string, error) {
	saPath = expandHome(saPath)
	data, err := os.ReadFile(saPath)
	if err != nil {
		return "", fmt.Errorf("reading service account %s: %w", saPath, err)
	}
	var sa struct {
		ClientEmail string `json:"client_email"`
		PrivateKey  string `json:"private_key"`
		TokenURI    string `json:"token_uri"`
	}
	if err := json.Unmarshal(data, &sa); err != nil {
		return "", fmt.Errorf("parsing service account %s: %w", saPath, err)
	}
	if sa.ClientEmail == "" || sa.PrivateKey == "" {
		return "", fmt.Errorf("service account %s is missing client_email/private_key", saPath)
	}
	if sa.TokenURI == "" {
		sa.TokenURI = "https://oauth2.googleapis.com/token"
	}
	key, err := parseRSAPrivateKey(sa.PrivateKey)
	if err != nil {
		return "", err
	}

	now := time.Now()
	header := map[string]string{"alg": "RS256", "typ": "JWT"}
	claims := map[string]any{
		"iss":   sa.ClientEmail,
		"scope": playScope,
		"aud":   sa.TokenURI,
		"iat":   now.Unix(),
		"exp":   now.Add(time.Hour).Unix(),
	}
	h, _ := json.Marshal(header)
	c, _ := json.Marshal(claims)
	signingInput := b64url(h) + "." + b64url(c)
	digest := sha256.Sum256([]byte(signingInput))
	sig, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, digest[:])
	if err != nil {
		return "", fmt.Errorf("signing JWT: %w", err)
	}
	assertion := signingInput + "." + b64url(sig)

	form := url.Values{}
	form.Set("grant_type", "urn:ietf:params:oauth:grant-type:jwt-bearer")
	form.Set("assertion", assertion)
	resp, err := http.PostForm(sa.TokenURI, form)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("OAuth token %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	var tok struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.Unmarshal(body, &tok); err != nil {
		return "", fmt.Errorf("parsing OAuth token: %w", err)
	}
	if tok.AccessToken == "" {
		return "", fmt.Errorf("OAuth token response missing access_token")
	}
	return tok.AccessToken, nil
}

func parseRSAPrivateKey(pemStr string) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode([]byte(pemStr))
	if block == nil {
		return nil, fmt.Errorf("no PEM block in private key")
	}
	if k, err := x509.ParsePKCS8PrivateKey(block.Bytes); err == nil {
		if rk, ok := k.(*rsa.PrivateKey); ok {
			return rk, nil
		}
	}
	if k, err := x509.ParsePKCS1PrivateKey(block.Bytes); err == nil {
		return k, nil
	}
	return nil, fmt.Errorf("private key is not a supported RSA format (PKCS1/PKCS8)")
}
