// Package idrac provides a client for the iDRAC6 REST/XML API.
package idrac

import (
	"crypto/tls"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"sync"
	"time"
)

// Client communicates with an iDRAC6 controller via its XML REST API.
type Client struct {
	host     string
	username string
	password string
	baseURL  string

	mu        sync.Mutex
	http      *http.Client
	sessionID string
	st1       string
	st2       string
	newAuth   bool
}

// loginResponse is the XML response from POST /data/login.
type loginResponse struct {
	XMLName    xml.Name `xml:"root"`
	AuthResult int      `xml:"authResult"`
	ForwardURL string   `xml:"forwardUrl"`
	ErrorMsg   string   `xml:"errorMsg"`
}

// NewClient creates a new iDRAC6 API client.
func NewClient(host, username, password string) *Client {
	jar, _ := cookiejar.New(nil)

	return &Client{
		host:     host,
		username: username,
		password: password,
		baseURL:  "https://" + host,
		http: &http.Client{
			Timeout: 15 * time.Second,
			Jar:     jar,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: true, //nolint:gosec // iDRAC6 uses self-signed certs
					// iDRAC6 only supports TLS 1.0/1.1 with legacy ciphers
					MinVersion: tls.VersionTLS10,
					MaxVersion: tls.VersionTLS12,
					CipherSuites: []uint16{
						tls.TLS_RSA_WITH_AES_128_CBC_SHA,
						tls.TLS_RSA_WITH_AES_256_CBC_SHA,
						tls.TLS_RSA_WITH_3DES_EDE_CBC_SHA,
						tls.TLS_RSA_WITH_AES_128_CBC_SHA256,
						tls.TLS_RSA_WITH_AES_128_GCM_SHA256,
						tls.TLS_RSA_WITH_AES_256_GCM_SHA384,
						tls.TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA,
						tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA,
						tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
					},
				},
			},
		},
	}
}

// Login authenticates with the iDRAC6 and stores the session.
func (c *Client) Login() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.login()
}

func (c *Client) login() error {
	// Step 1: Get session cookie from /start.html
	// iDRAC6 sets _appwebSessionId_ on the start page, not on login POST
	sessionReq, err := http.NewRequest("GET", c.baseURL+"/start.html", nil)
	if err != nil {
		return fmt.Errorf("creating session request: %w", err)
	}

	sessionResp, err := c.http.Do(sessionReq)
	if err != nil {
		return fmt.Errorf("session request failed: %w", err)
	}
	sessionResp.Body.Close()

	// Extract session cookie from start.html response
	c.sessionID = ""
	for _, cookie := range sessionResp.Cookies() {
		if cookie.Name == "_appwebSessionId_" {
			c.sessionID = cookie.Value
			break
		}
	}

	// Also check set-cookie header directly
	if c.sessionID == "" {
		setCookie := sessionResp.Header.Get("Set-Cookie")
		if idx := strings.Index(setCookie, "_appwebSessionId_="); idx >= 0 {
			val := setCookie[idx+len("_appwebSessionId_="):]
			if semi := strings.Index(val, ";"); semi >= 0 {
				val = val[:semi]
			}
			c.sessionID = val
		}
	}

	if c.sessionID == "" {
		return fmt.Errorf("no session cookie from /start.html")
	}

	// Step 2: Login with the session cookie
	// IMPORTANT: iDRAC6 requires "user" before "password" in the POST body.
	// Go's url.Values.Encode() sorts alphabetically, which breaks auth.
	formBody := "user=" + url.QueryEscape(c.username) + "&password=" + url.QueryEscape(c.password)

	loginReq, err := http.NewRequest("POST", c.baseURL+"/data/login", strings.NewReader(formBody))
	if err != nil {
		return fmt.Errorf("creating login request: %w", err)
	}
	loginReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	loginReq.AddCookie(&http.Cookie{Name: "_appwebSessionId_", Value: c.sessionID})

	loginResp, err := c.http.Do(loginReq)
	if err != nil {
		return fmt.Errorf("login request failed: %w", err)
	}
	defer loginResp.Body.Close()

	body, err := io.ReadAll(loginResp.Body)
	if err != nil {
		return fmt.Errorf("reading login response: %w", err)
	}

	var result loginResponse
	if err := xml.Unmarshal(body, &result); err != nil {
		return fmt.Errorf("parsing login response: %w", err)
	}

	// authResult: 0=success, non-zero=failure
	// (1=bad credentials, 2=missing user, 3=missing password, 4=privilege, 5=session limit)
	if result.AuthResult != 0 {
		return fmt.Errorf("login failed: authResult=%d, error=%s", result.AuthResult, result.ErrorMsg)
	}

	// Check if login response provides a new/different session cookie
	for _, cookie := range loginResp.Cookies() {
		if cookie.Name == "_appwebSessionId_" {
			c.sessionID = cookie.Value
			break
		}
	}

	// Extract ST1/ST2 tokens for newAuth (firmware >=2.92)
	if result.ForwardURL != "" {
		c.extractTokens(result.ForwardURL)
	}

	return nil
}

// extractTokens parses ST1/ST2 from forwardUrl like "index.html?ST1=abc,ST2=def"
func (c *Client) extractTokens(forwardURL string) {
	parts := strings.SplitN(forwardURL, "?", 2)
	if len(parts) < 2 {
		return
	}

	for _, param := range strings.Split(parts[1], ",") {
		kv := strings.SplitN(param, "=", 2)
		if len(kv) != 2 {
			continue
		}
		switch kv[0] {
		case "ST1":
			c.st1 = kv[1]
			c.newAuth = true
		case "ST2":
			c.st2 = kv[1]
			c.newAuth = true
		}
	}
}

// Get fetches data from the iDRAC6 API. keys are comma-separated data type names
// like "pwState", "temperatures", "sysDesc".
func (c *Client) Get(keys ...string) ([]byte, error) {
	return c.doWithRetry(func() (*http.Response, error) {
		reqURL := fmt.Sprintf("%s/data?get=%s", c.baseURL, strings.Join(keys, ","))
		req, err := http.NewRequest("GET", reqURL, nil)
		if err != nil {
			return nil, err
		}
		c.applySession(req)
		return c.http.Do(req)
	})
}

// Set sends a set command to the iDRAC6 API (e.g., "pwState:1" for power on).
func (c *Client) Set(param string) ([]byte, error) {
	return c.doWithRetry(func() (*http.Response, error) {
		reqURL := fmt.Sprintf("%s/data?set=%s", c.baseURL, url.QueryEscape(param))
		req, err := http.NewRequest("GET", reqURL, nil)
		if err != nil {
			return nil, err
		}
		c.applySession(req)
		return c.http.Do(req)
	})
}

// PostForm sends a POST with form data to the given path.
func (c *Client) PostForm(path string, form url.Values) ([]byte, error) {
	return c.doWithRetry(func() (*http.Response, error) {
		req, err := http.NewRequest("POST", c.baseURL+path, strings.NewReader(form.Encode()))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		c.applySession(req)
		return c.http.Do(req)
	})
}

// applySession adds auth headers/cookies to a request.
func (c *Client) applySession(req *http.Request) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.sessionID != "" {
		req.AddCookie(&http.Cookie{
			Name:  "_appwebSessionId_",
			Value: c.sessionID,
		})
	}
	if c.newAuth && c.st2 != "" {
		req.Header.Set("ST2", c.st2)
	}
}

// doWithRetry executes a request, retrying once on 401 after re-login.
func (c *Client) doWithRetry(fn func() (*http.Response, error)) ([]byte, error) {
	resp, err := fn()
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		resp.Body.Close()

		c.mu.Lock()
		loginErr := c.login()
		c.mu.Unlock()

		if loginErr != nil {
			return nil, fmt.Errorf("re-login after 401 failed: %w", loginErr)
		}

		resp, err = fn()
		if err != nil {
			return nil, fmt.Errorf("retry request failed: %w", err)
		}
		defer resp.Body.Close()
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response body: %w", err)
	}

	return body, nil
}

// Logout terminates the iDRAC6 session.
func (c *Client) Logout() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	req, err := http.NewRequest("GET", c.baseURL+"/data/logout", nil)
	if err != nil {
		return err
	}
	if c.sessionID != "" {
		req.AddCookie(&http.Cookie{
			Name:  "_appwebSessionId_",
			Value: c.sessionID,
		})
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	resp.Body.Close()

	c.sessionID = ""
	c.st1 = ""
	c.st2 = ""
	c.newAuth = false

	return nil
}

// Host returns the configured iDRAC host address.
func (c *Client) Host() string {
	return c.host
}

// BaseURL returns the base URL for the iDRAC API.
func (c *Client) BaseURL() string {
	return c.baseURL
}
