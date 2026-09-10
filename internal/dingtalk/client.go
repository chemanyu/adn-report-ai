// Package dingtalk adapts the external DingTalk OAuth API.
package dingtalk

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type Profile struct {
	UnionID string `json:"unionId"`
	Nick    string `json:"nick"`
}
type Client struct{ HTTPClient *http.Client }

func (c *Client) Exchange(ctx context.Context, clientID, secret, code string) (Profile, error) {
	var profile Profile
	body, err := json.Marshal(map[string]string{"clientId": clientID, "clientSecret": secret, "code": code, "grantType": "authorization_code"})
	if err != nil {
		return profile, err
	}
	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.dingtalk.com/v1.0/oauth2/userAccessToken", bytes.NewReader(body))
	if err != nil {
		return profile, err
	}
	req.Header.Set("Content-Type", "application/json")
	var token struct {
		AccessToken string `json:"accessToken"`
	}
	if err = c.request(req, &token); err != nil {
		return profile, err
	}
	if token.AccessToken == "" {
		return profile, fmt.Errorf("钉钉未返回用户 token")
	}
	req, err = http.NewRequestWithContext(ctx, "GET", "https://api.dingtalk.com/v1.0/contact/users/me", nil)
	if err != nil {
		return profile, err
	}
	req.Header.Set("x-acs-dingtalk-access-token", token.AccessToken)
	if err = c.request(req, &profile); err != nil {
		return profile, err
	}
	if profile.UnionID == "" {
		return profile, fmt.Errorf("钉钉未返回用户身份")
	}
	return profile, nil
}
func (c *Client) request(req *http.Request, out any) error {
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("钉钉返回 HTTP %d", resp.StatusCode)
	}
	return json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(out)
}
