package cloud

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/sentra-lab/cli/internal/utils"
	"github.com/skratchdot/open-golang/open"
)

const (
	authURL     = "https://auth.sentra.dev"
	apiURL      = "https://api.sentra.dev"
	callbackURL = "http://localhost:8765/callback"
)

type AuthClient struct {
	logger       *utils.Logger
	clientID     string
	clientSecret string
	callbackPort int
}

func NewAuthClient(logger *utils.Logger) *AuthClient {
	return &AuthClient{
		logger:       logger,
		clientID:     "sentra-lab-cli",
		clientSecret: "",
		callbackPort: 8765,
	}
}

func (ac *AuthClient) Login(ctx context.Context) (string, *User, error) {
	state, err := generateState()
	if err != nil {
		return "", nil, fmt.Errorf("failed to generate state: %w", err)
	}

	authCodeChan := make(chan string, 1)
	errorChan := make(chan error, 1)
	server := &http.Server{
		Addr: fmt.Sprintf(":%d", ac.callbackPort),
	}

	http.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		code := r.URL.Query().Get("code")
		receivedState := r.URL.Query().Get("state")

		if receivedState != state {
			errorChan <- fmt.Errorf("state mismatch (CSRF protection)")
			http.Error(w, "Authentication failed", http.StatusBadRequest)
			return
		}

		if code == "" {
			errorChan <- fmt.Errorf("no authorization code received")
			http.Error(w, "Authentication failed", http.StatusBadRequest)
			return
		}

		authCodeChan <- code

		w.Header().Set("Content-Type", "text/html")
		fmt.Fprintf(w, `
			<html>
			<head><title>Sentra Lab - Authentication Success</title></head>
			<body style="font-family: system-ui; text-align: center; padding: 50px;">
				<h1>✅ Authentication Successful</h1>
				<p>You can close this window and return to the CLI.</p>
			</body>
			</html>
		`)
	})

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errorChan <- fmt.Errorf("callback server failed: %w", err)
		}
	}()

	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		server.Shutdown(shutdownCtx)
	}()

	authURL := ac.buildAuthURL(state)

	ac.logger.Info(fmt.Sprintf("Opening browser: %s", authURL))

	if err := open.Run(authURL); err != nil {
		ac.logger.Warn("Failed to open browser automatically")
		ac.logger.Info(fmt.Sprintf("Please open this URL manually: %s", authURL))
	}

	ac.logger.Info("Waiting for authentication...")

	select {
	case code := <-authCodeChan:
		ac.logger.Info("✓ Authorization code received")

		token, user, err := ac.exchangeCodeForToken(ctx, code)
		if err != nil {
			return "", nil, fmt.Errorf("token exchange failed: %w", err)
		}

		return token, user, nil

	case err := <-errorChan:
		return "", nil, err

	case <-ctx.Done():
		return "", nil, ctx.Err()

	case <-time.After(5 * time.Minute):
		return "", nil, fmt.Errorf("authentication timeout")
	}
}

func (ac *AuthClient) buildAuthURL(state string) string {
	params := url.Values{}
	params.Set("client_id", ac.clientID)
	params.Set("redirect_uri", callbackURL)
	params.Set("response_type", "code")
	params.Set("state", state)
	params.Set("scope", "read write")

	return fmt.Sprintf("%s/oauth/authorize?%s", authURL, params.Encode())
}

func (ac *AuthClient) exchangeCodeForToken(ctx context.Context, code string) (string, *User, error) {
	data := url.Values{}
	data.Set("grant_type", "authorization_code")
	data.Set("code", code)
	data.Set("redirect_uri", callbackURL)
	data.Set("client_id", ac.clientID)

	req, err := http.NewRequestWithContext(ctx, "POST", fmt.Sprintf("%s/oauth/token", authURL), nil)
	if err != nil {
		return "", nil, err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.URL.RawQuery = data.Encode()

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", nil, fmt.Errorf("token exchange failed: %s (status: %d)", string(body), resp.StatusCode)
	}

	var tokenResp struct {
		AccessToken string `json:"access_token"`
		TokenType   string `json:"token_type"`
		ExpiresIn   int    `json:"expires_in"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return "", nil, fmt.Errorf("failed to decode token response: %w", err)
	}

	user, err := ac.getUserInfo(ctx, tokenResp.AccessToken)
	if err != nil {
		return "", nil, fmt.Errorf("failed to get user info: %w", err)
	}

	return tokenResp.AccessToken, user, nil
}

func (ac *AuthClient) getUserInfo(ctx context.Context, token string) (*User, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("%s/v1/user", apiURL), nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get user info: status %d", resp.StatusCode)
	}

	var userResp struct {
		Email string `json:"email"`
		Team  string `json:"team"`
		Plan  string `json:"plan"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&userResp); err != nil {
		return nil, err
	}

	return &User{
		Email: userResp.Email,
		Team:  userResp.Team,
		Plan:  userResp.Plan,
	}, nil
}

func generateState() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}
