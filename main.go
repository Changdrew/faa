package main

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"

	"fmt"
	"strings"

	"github.com/mariash/faa/postfacto"
	"github.com/mariash/faa/slackcommand"
)

type PostfactoConfig map[SlackChannelName]PostfactoData

type SlackChannelName string
type PostfactoData struct {
	Name     string `json:"name"`
	RetroID  int    `json:"retro_id"`
	Password string `json:"password"`
}

const PostfactoAPIURL = "https://retro-api.cfapps.io"

func main() {
	var (
		port string
		ok   bool
	)
	port, ok = os.LookupEnv("PORT")
	if !ok {
		port = "8080"
	}

	vToken, ok := os.LookupEnv("SLACK_VERIFICATION_TOKEN")
	if !ok {
		panic(errors.New("Must provide SLACK_VERIFICATION_TOKEN"))
	}

	postfactoConfigJSON, ok := os.LookupEnv("POSTFACTO_CONFIG")
	if !ok {
		panic(errors.New("Must provide POSTFACTO_CONFIG"))
	}

	var postfactoConfig PostfactoConfig
	err := json.Unmarshal([]byte(postfactoConfigJSON), &postfactoConfig)
	if err != nil {
		panic(errors.New("Failed to parse POSTFACTO_CONFIG: " + err.Error()))
	}

	server := slackcommand.Server{
		VerificationToken: vToken,
		Delegate: &PostfactoSlackDelegate{
			Config: postfactoConfig,
		},
	}

	http.Handle("/", server)

	log.Fatal(http.ListenAndServe(":"+port, nil))
}

type PostfactoSlackDelegate struct {
	Config PostfactoConfig
}

type Command string

const (
	CommandHappy Command = "happy"
	CommandMeh   Command = "meh"
	CommandSad   Command = "sad"
)

func (d *PostfactoSlackDelegate) Handle(r slackcommand.Command) (string, error) {
	parts := strings.SplitN(r.Text, " ", 2)
	if len(parts) < 2 {
		return "", fmt.Errorf("must be in the form of '%s [happy/meh/sad] [message]'", r.Command)
	}

	c := parts[0]
	description := parts[1]

	var (
		category postfacto.Category
	)

	postfactoData, ok := d.Config[SlackChannelName(r.ChannelID)]
	if !ok {
		return "", fmt.Errorf("channel '%s' with ID '%s' is not configured", r.ChannelName, r.ChannelID)
	}

	if postfactoData.RetroID == 0 {
		return "", fmt.Errorf("retro ID is not set")
	}

	switch Command(c) {
	case CommandHappy:
		category = postfacto.CategoryHappy
	case CommandMeh:
		category = postfacto.CategoryMeh
	case CommandSad:
		category = postfacto.CategorySad
	default:
		return "", errors.New("unknown command: must provide one of 'happy', 'meh', or 'sad'")
	}

	retroItem := postfacto.RetroItem{
		Category:    category,
		Description: fmt.Sprintf("%s [%s]", description, r.UserName),
	}

	client := &postfacto.RetroClient{
		Host:     PostfactoAPIURL,
		ID:       postfactoData.RetroID,
		Password: postfactoData.Password,
	}

	err := client.Add(retroItem)
	if err != nil {
		return "", err
	}

	return "retro item added", nil
}
