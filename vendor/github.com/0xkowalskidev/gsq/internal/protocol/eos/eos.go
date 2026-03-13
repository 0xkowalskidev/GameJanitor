package eos

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/0xkowalskidev/gsq/internal/protocol"
)

const (
	authURL             = "https://api.epicgames.dev/auth/v1/oauth/token"
	deviceIDURL         = "https://api.epicgames.dev/auth/v1/accounts/deviceid"
	matchmakingURLFmt   = "https://api.epicgames.dev/matchmaking/v1/%s/filter"
	wildcardMatchURLFmt = "https://api.epicgames.dev/wildcard/matchmaking/v1/%s/filter"
)

type eosQuerier struct{}

func init() {
	protocol.Register("eos", &eosQuerier{})
}

func (q *eosQuerier) Query(ctx context.Context, address string, port uint16, opts protocol.QueryOpts) (*protocol.ServerInfo, error) {
	if opts.EOS == nil {
		return nil, fmt.Errorf("eos: no credentials configured (use --game to select an EOS game)")
	}
	cfg := opts.EOS

	start := time.Now()

	token, err := authenticate(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("eos auth: %w", err)
	}

	resolvedIP := opts.ResolvedIP
	if resolvedIP == "" {
		resolvedIP = address
	}

	session, err := findSession(ctx, cfg, token, resolvedIP, port)
	if err != nil {
		return nil, fmt.Errorf("eos query %s:%d: %w", address, port, err)
	}
	ping := time.Since(start)

	info := mapSession(session, cfg.Attributes)
	info.Protocol = "eos"
	info.Address = address
	info.GamePort = port
	info.QueryPort = port
	info.Ping = protocol.Duration{Duration: ping}

	return info, nil
}

// authenticate obtains an access token via client_credentials or device ID flow.
func authenticate(ctx context.Context, cfg *protocol.EOSConfig) (string, error) {
	basicAuth := base64.StdEncoding.EncodeToString([]byte(cfg.ClientID + ":" + cfg.ClientSecret))

	if cfg.UseExternalAuth {
		return authenticateExternal(ctx, basicAuth, cfg.DeploymentID)
	}
	return authenticateClient(ctx, basicAuth, cfg.DeploymentID)
}

func authenticateClient(ctx context.Context, basicAuth, deploymentID string) (string, error) {
	body := url.Values{
		"grant_type":    {"client_credentials"},
		"deployment_id": {deploymentID},
	}

	slog.Debug("eos: authenticating with client_credentials")
	return postForToken(ctx, authURL, basicAuth, body)
}

func authenticateExternal(ctx context.Context, basicAuth, deploymentID string) (string, error) {
	// Step 1: get device ID token
	slog.Debug("eos: requesting device ID token")
	deviceBody := url.Values{"deviceModel": {"PC"}}
	deviceToken, err := postForToken(ctx, deviceIDURL, basicAuth, deviceBody)
	if err != nil {
		return "", fmt.Errorf("device ID: %w", err)
	}

	// Step 2: exchange for access token
	slog.Debug("eos: exchanging device token for access token")
	body := url.Values{
		"grant_type":          {"external_auth"},
		"external_auth_type":  {"deviceid_access_token"},
		"external_auth_token": {deviceToken},
		"nonce":               {"gsq"},
		"deployment_id":       {deploymentID},
		"display_name":        {"User"},
	}
	return postForToken(ctx, authURL, basicAuth, body)
}

func postForToken(ctx context.Context, endpoint, basicAuth string, body url.Values) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(body.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Basic "+basicAuth)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP %d: %s", resp.StatusCode, truncate(string(respBody), 200))
	}

	var result struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", fmt.Errorf("parse token response: %w", err)
	}
	if result.AccessToken == "" {
		return "", fmt.Errorf("empty access_token in response")
	}
	return result.AccessToken, nil
}

// findSession queries the matchmaking API and finds the session matching address:port.
func findSession(ctx context.Context, cfg *protocol.EOSConfig, token, address string, port uint16) (map[string]json.RawMessage, error) {
	urlFmt := matchmakingURLFmt
	if cfg.UseWildcard {
		urlFmt = wildcardMatchURLFmt
	}
	endpoint := fmt.Sprintf(urlFmt, cfg.DeploymentID)

	filterBody := map[string]any{
		"criteria": []map[string]any{
			{
				"key":   "attributes.ADDRESS_s",
				"op":    "EQUAL",
				"value": address,
			},
		},
	}

	bodyBytes, err := json.Marshal(filterBody)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	slog.Debug("eos: querying matchmaking", "endpoint", endpoint, "address", address, "port", port)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, truncate(string(respBody), 200))
	}

	var envelope struct {
		Sessions []map[string]json.RawMessage `json:"sessions"`
		Count    int                          `json:"count"`
	}
	if err := json.Unmarshal(respBody, &envelope); err != nil {
		return nil, fmt.Errorf("parse sessions: %w", err)
	}
	sessions := envelope.Sessions

	// The EOS matchmaking API caps results at 15 sessions and provides no pagination.
	// When an IP hosts more sessions than that (common with large hosting providers),
	// the target server may not appear in the results.
	slog.Debug("eos: got sessions", "returned", len(sessions), "total", envelope.Count)
	if envelope.Count > len(sessions) {
		slog.Warn("eos: API returned partial results — server may not be found", "returned", len(sessions), "total", envelope.Count, "address", address)
	}

	portStr := fmt.Sprintf("%d", port)
	for _, session := range sessions {
		attrs := sessionAttributes(session)
		if attrs == nil {
			continue
		}

		// Match by ADDRESSBOUND_s port suffix — covers both "ip:port" and "0.0.0.0:port" forms
		if bound, ok := attrs["ADDRESSBOUND_s"]; ok {
			if strings.HasSuffix(bound, ":"+portStr) {
				return session, nil
			}
		}
		if gamePort, ok := attrs["GAMESERVER_PORT_l"]; ok {
			if gamePort == portStr {
				return session, nil
			}
		}
	}

	if len(sessions) == 1 {
		// Only one session for this IP — likely the right one even if port doesn't match
		slog.Debug("eos: single session found, using it despite port mismatch")
		return sessions[0], nil
	}

	return nil, fmt.Errorf("no session found for %s:%d (%d sessions at this address)", address, port, len(sessions))
}

// sessionAttributes extracts the attributes map from a session.
func sessionAttributes(session map[string]json.RawMessage) map[string]string {
	raw, ok := session["attributes"]
	if !ok {
		return nil
	}

	var attrs map[string]json.RawMessage
	if err := json.Unmarshal(raw, &attrs); err != nil {
		return nil
	}

	result := make(map[string]string, len(attrs))
	for k, v := range attrs {
		var s string
		if err := json.Unmarshal(v, &s); err != nil {
			// Try as number
			var n json.Number
			if err := json.Unmarshal(v, &n); err == nil {
				s = n.String()
			} else {
				// Try as bool
				var b bool
				if err := json.Unmarshal(v, &b); err == nil {
					s = fmt.Sprintf("%t", b)
				} else {
					s = string(v)
				}
			}
		}
		result[k] = s
	}
	return result
}

// mapSession converts a raw EOS session into ServerInfo using the game's attribute map.
func mapSession(session map[string]json.RawMessage, attrMap map[string]string) *protocol.ServerInfo {
	attrs := sessionAttributes(session)
	info := &protocol.ServerInfo{}

	// Map standard fields using game-specific attribute keys
	if key, ok := attrMap["name"]; ok {
		info.Name = attrs[key]
	} else if v, ok := attrs["CUSTOMSERVERNAME_s"]; ok {
		info.Name = v
	}

	if key, ok := attrMap["map"]; ok {
		info.Map = attrs[key]
	} else if v, ok := attrs["MAPNAME_s"]; ok {
		info.Map = v
	}

	if key, ok := attrMap["version"]; ok {
		info.Version = attrs[key]
	}

	if key, ok := attrMap["password"]; ok {
		if attrs[key] == "true" {
			info.Visibility = "private"
		} else {
			info.Visibility = "public"
		}
	} else if v, ok := attrs["SERVERPASSWORD_b"]; ok {
		if v == "true" {
			info.Visibility = "private"
		} else {
			info.Visibility = "public"
		}
	}

	// Player counts from session-level fields
	info.Players = jsonInt(session, "totalPlayers")
	info.MaxPlayers = jsonInt(session, "settings", "maxPublicPlayers")

	// Player list from publicPlayers array
	if raw, ok := session["publicPlayers"]; ok {
		var playerIDs []string
		if err := json.Unmarshal(raw, &playerIDs); err == nil {
			for _, id := range playerIDs {
				if id != "" {
					info.PlayerList = append(info.PlayerList, protocol.PlayerInfo{Name: id})
				}
			}
		}
	}

	return info
}

// jsonInt extracts an integer from a nested JSON path.
func jsonInt(session map[string]json.RawMessage, keys ...string) int {
	if len(keys) == 0 {
		return 0
	}

	raw, ok := session[keys[0]]
	if !ok {
		return 0
	}

	if len(keys) == 1 {
		var n int
		json.Unmarshal(raw, &n)
		return n
	}

	// Navigate nested object
	var nested map[string]json.RawMessage
	if err := json.Unmarshal(raw, &nested); err != nil {
		return 0
	}
	return jsonInt(nested, keys[1:]...)
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
