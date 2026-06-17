package notification

import (
	"bytes"
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"datn-backend/internal/domain"
)

const (
	firebaseMessagingScope     = "https://www.googleapis.com/auth/firebase.messaging"
	alertNotificationChannelID = "security_alerts"
)

type FCMSender struct {
	serviceAccountFile string
	projectID          string
	httpClient         *http.Client

	mu             sync.Mutex
	serviceAccount *firebaseServiceAccount
	token          oauthToken
}

type firebaseServiceAccount struct {
	ProjectID   string `json:"project_id"`
	ClientEmail string `json:"client_email"`
	PrivateKey  string `json:"private_key"`
	TokenURI    string `json:"token_uri"`
}

type oauthToken struct {
	AccessToken string
	Expiry      time.Time
}

type FCMError struct {
	StatusCode int
	Body       string
	ErrorCode  string
}

func (e *FCMError) Error() string {
	if e == nil {
		return ""
	}
	if e.ErrorCode != "" {
		return fmt.Sprintf("fcm send failed status=%d error_code=%s body=%s", e.StatusCode, e.ErrorCode, e.Body)
	}
	return fmt.Sprintf("fcm send failed status=%d body=%s", e.StatusCode, e.Body)
}

func NewFCMSender(serviceAccountFile string, projectID string) *FCMSender {
	return &FCMSender{
		serviceAccountFile: strings.TrimSpace(serviceAccountFile),
		projectID:          strings.TrimSpace(projectID),
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

func (s *FCMSender) SendAlert(ctx context.Context, alert domain.Alert, devices []domain.MobileDevice) (SendReport, error) {
	accessToken, projectID, err := s.accessToken(ctx)
	if err != nil {
		return SendReport{}, err
	}

	var report SendReport
	var sendErrors []error
	for _, device := range devices {
		token := strings.TrimSpace(device.FCMToken)
		if token == "" {
			continue
		}
		report.Targeted++
		if err := s.sendToToken(ctx, accessToken, projectID, alert, token); err != nil {
			report.Failed = append(report.Failed, SendFailure{
				Token:        token,
				Err:          err,
				InvalidToken: isInvalidFCMTokenError(err),
			})
			sendErrors = append(sendErrors, err)
			continue
		}
		report.Sent++
	}

	return report, errors.Join(sendErrors...)
}

func (s *FCMSender) sendToToken(ctx context.Context, accessToken string, projectID string, alert domain.Alert, token string) error {
	title := "Laptop Guard cảnh báo"
	body := strings.TrimSpace(alert.Message)
	if body == "" {
		body = "Phát hiện sự kiện bất thường trên thiết bị của bạn."
	}

	payload := map[string]any{
		"message": map[string]any{
			"token": token,
			"notification": map[string]string{
				"title": title,
				"body":  body,
			},
			"data": map[string]string{
				"alert_id":     alert.ID,
				"pc_agent_id":  alert.AgentID,
				"alert_type":   alert.Type,
				"triggered_at": alert.CreatedAt.Format(time.RFC3339),
			},
			"android": map[string]any{
				"priority": "HIGH",
				"notification": map[string]any{
					"channel_id":              alertNotificationChannelID,
					"notification_priority":   "PRIORITY_HIGH",
					"default_sound":           true,
					"default_vibrate_timings": true,
				},
			},
		},
	}

	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	endpoint := fmt.Sprintf("https://fcm.googleapis.com/v1/projects/%s/messages:send", url.PathEscape(projectID))
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(bodyBytes))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		responseBody, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		trimmedBody := strings.TrimSpace(string(responseBody))
		return &FCMError{
			StatusCode: resp.StatusCode,
			Body:       trimmedBody,
			ErrorCode:  parseFCMErrorCode(responseBody),
		}
	}

	responseBody, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	log.Printf("fcm send accepted: alert_id=%s response=%s", alert.ID, strings.TrimSpace(string(responseBody)))
	return nil
}

func isInvalidFCMTokenError(err error) bool {
	var fcmErr *FCMError
	if !errors.As(err, &fcmErr) {
		return false
	}

	return strings.EqualFold(fcmErr.ErrorCode, "UNREGISTERED")
}

func parseFCMErrorCode(body []byte) string {
	var response struct {
		Error struct {
			Details []struct {
				ErrorCode string `json:"errorCode"`
			} `json:"details"`
		} `json:"error"`
	}
	if err := json.Unmarshal(body, &response); err != nil {
		return ""
	}

	for _, detail := range response.Error.Details {
		if errorCode := strings.TrimSpace(detail.ErrorCode); errorCode != "" {
			return errorCode
		}
	}
	return ""
}

func (s *FCMSender) accessToken(ctx context.Context) (string, string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.token.AccessToken != "" && time.Now().Before(s.token.Expiry.Add(-1*time.Minute)) {
		return s.token.AccessToken, s.resolvedProjectID(), nil
	}

	serviceAccount, err := s.loadServiceAccount()
	if err != nil {
		return "", "", err
	}

	projectID := strings.TrimSpace(s.projectID)
	if projectID == "" {
		projectID = strings.TrimSpace(serviceAccount.ProjectID)
	}
	if projectID == "" {
		return "", "", errors.New("firebase project id is empty")
	}

	assertion, err := buildJWTAssertion(serviceAccount)
	if err != nil {
		return "", "", err
	}

	token, err := s.requestAccessToken(ctx, serviceAccount.TokenURI, assertion)
	if err != nil {
		return "", "", err
	}

	s.token = token
	return token.AccessToken, projectID, nil
}

func (s *FCMSender) resolvedProjectID() string {
	if strings.TrimSpace(s.projectID) != "" {
		return strings.TrimSpace(s.projectID)
	}
	if s.serviceAccount != nil {
		return strings.TrimSpace(s.serviceAccount.ProjectID)
	}
	return ""
}

func (s *FCMSender) loadServiceAccount() (*firebaseServiceAccount, error) {
	if s.serviceAccount != nil {
		return s.serviceAccount, nil
	}

	if s.serviceAccountFile == "" {
		return nil, errors.New("firebase service account file is empty")
	}

	content, err := os.ReadFile(s.serviceAccountFile)
	if err != nil {
		return nil, err
	}

	var serviceAccount firebaseServiceAccount
	if err := json.Unmarshal(content, &serviceAccount); err != nil {
		return nil, err
	}
	if serviceAccount.TokenURI == "" {
		serviceAccount.TokenURI = "https://oauth2.googleapis.com/token"
	}
	if strings.TrimSpace(serviceAccount.ClientEmail) == "" || strings.TrimSpace(serviceAccount.PrivateKey) == "" {
		return nil, errors.New("firebase service account missing client_email or private_key")
	}

	s.serviceAccount = &serviceAccount
	return s.serviceAccount, nil
}

func (s *FCMSender) requestAccessToken(ctx context.Context, tokenURI string, assertion string) (oauthToken, error) {
	form := url.Values{}
	form.Set("grant_type", "urn:ietf:params:oauth:grant-type:jwt-bearer")
	form.Set("assertion", assertion)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, tokenURI, strings.NewReader(form.Encode()))
	if err != nil {
		return oauthToken{}, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return oauthToken{}, err
	}
	defer resp.Body.Close()

	var response struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int64  `json:"expires_in"`
		Error       string `json:"error"`
		Description string `json:"error_description"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return oauthToken{}, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return oauthToken{}, fmt.Errorf("firebase token request failed status=%d error=%s description=%s", resp.StatusCode, response.Error, response.Description)
	}
	if response.AccessToken == "" {
		return oauthToken{}, errors.New("firebase token response missing access_token")
	}
	if response.ExpiresIn <= 0 {
		response.ExpiresIn = 3600
	}

	return oauthToken{
		AccessToken: response.AccessToken,
		Expiry:      time.Now().Add(time.Duration(response.ExpiresIn) * time.Second),
	}, nil
}

func buildJWTAssertion(serviceAccount *firebaseServiceAccount) (string, error) {
	now := time.Now()
	claims := map[string]any{
		"iss":   serviceAccount.ClientEmail,
		"scope": firebaseMessagingScope,
		"aud":   serviceAccount.TokenURI,
		"iat":   now.Unix(),
		"exp":   now.Add(time.Hour).Unix(),
	}
	header := map[string]string{
		"alg": "RS256",
		"typ": "JWT",
	}

	headerBytes, err := json.Marshal(header)
	if err != nil {
		return "", err
	}
	claimsBytes, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}

	signingInput := base64.RawURLEncoding.EncodeToString(headerBytes) + "." + base64.RawURLEncoding.EncodeToString(claimsBytes)
	signature, err := signRS256(serviceAccount.PrivateKey, []byte(signingInput))
	if err != nil {
		return "", err
	}

	return signingInput + "." + base64.RawURLEncoding.EncodeToString(signature), nil
}

func signRS256(privateKeyPEM string, payload []byte) ([]byte, error) {
	block, _ := pem.Decode([]byte(privateKeyPEM))
	if block == nil {
		return nil, errors.New("invalid firebase private key pem")
	}

	key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, err
	}

	rsaKey, ok := key.(*rsa.PrivateKey)
	if !ok {
		return nil, errors.New("firebase private key is not RSA")
	}

	digest := sha256.Sum256(payload)
	return rsa.SignPKCS1v15(rand.Reader, rsaKey, crypto.SHA256, digest[:])
}
