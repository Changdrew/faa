package postfacto

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httputil"
)

type RetroClient struct {
	Host     string
	Name     string
	Password string
}

type Category string

const (
	CategoryHappy Category = "happy"
	CategoryMeh   Category = "meh"
	CategorySad   Category = "sad"
)

type RetroItem struct {
	Description string   `json:"description"`
	Category    Category `json:"category"`
}

type TokenRequest struct {
	Retro RetroConfig `json:"retro"`
}

type RetroConfig struct {
	Password string `json:"password"`
}

type TokenResponse struct {
	Token string `json:"token"`
}

func (c *RetroClient) Add(i RetroItem) error {
	retroURL := fmt.Sprintf("%s/retros/%s", c.Host, c.Name)
	var authorizationToken string

	if c.Password != "" {
		tokenRequest := TokenRequest{
			Retro: RetroConfig{
				Password: c.Password,
			},
		}
		tokenRequestJSON, err := json.Marshal(tokenRequest)
		if err != nil {
			return err
		}

		b := bytes.NewReader(tokenRequestJSON)
		req, err := http.NewRequest("PUT", retroURL+"/login", b)
		if err != nil {
			return err
		}
		req.Header.Add("Content-Type", "application/json")
		req.Header.Add("Accept", "application/json")

		res, err := http.DefaultClient.Do(req)
		if err != nil {
			return fmt.Errorf("failed sending token request: " + err.Error())
		}
		defer res.Body.Close()

		var tokenResponse TokenResponse
		err = json.NewDecoder(res.Body).Decode(&tokenResponse)
		if err != nil {
			return fmt.Errorf("failed to decode token response: " + err.Error())
		}
		authorizationToken = tokenResponse.Token
	}

	b := new(bytes.Buffer)
	err := json.NewEncoder(b).Encode(i)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", retroURL+"/items", b)
	if err != nil {
		return err
	}

	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Accept", "application/json")
	if authorizationToken != "" {
		req.Header.Add("Authorization", "Bearer "+authorizationToken)
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusCreated {
		b, _ := httputil.DumpResponse(res, true)
		return fmt.Errorf("unexpected response code (%d) - %s", res.StatusCode, string(b))
	}

	return nil
}
