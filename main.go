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
	Name        string `json:"name"`
	RetroID     int    `json:"retro_id"`
	TechRetroID int    `json:"tech_retro_id"`
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
	CommandTech  Command = "tech"
)

func (d *PostfactoSlackDelegate) Handle(r slackcommand.Command) (string, error) {
	parts := strings.SplitN(r.Text, " ", 2)
	if len(parts) < 2 {
		return "", fmt.Errorf("must be in the form of '%s [happy/meh/sad/tech] [message]'", r.Command)
	}

	c := parts[0]
	description := parts[1]

	var (
		category postfacto.Category
		retroID  int
	)

	postfactoData, ok := d.Config[SlackChannelName(r.ChannelID)]
	if !ok {
		return "", fmt.Errorf("channel '%s' with id '%s' is not configured", r.ChannelName, r.ChannelID)
	}

	switch Command(c) {
	case CommandHappy:
		category = postfacto.CategoryHappy
		retroID = postfactoData.RetroID
	case CommandMeh:
		category = postfacto.CategoryMeh
		retroID = postfactoData.RetroID
	case CommandSad:
		category = postfacto.CategorySad
		retroID = postfactoData.RetroID
	case CommandTech:
		category = postfacto.CategoryHappy
		retroID = postfactoData.TechRetroID
	default:
		return "", errors.New("unknown command: must provide one of 'happy', 'meh', 'sad', or 'tech'")
	}

	if retroID == 0 {
		return "", fmt.Errorf("invalid retro id '%d'", retroID)
	}

	retroItem := postfacto.RetroItem{
		Category:    category,
		Description: fmt.Sprintf("%s [%s]", description, r.UserName),
	}

	client := &postfacto.RetroClient{
		Host: PostfactoAPIURL,
		ID:   string(retroID),
	}

	err := client.Add(retroItem)
	if err != nil {
		return "", err
	}

	return "retro item added", nil
}
