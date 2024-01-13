package gcp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/google/generative-ai-go/genai"
	"github.com/slack-go/slack"
	"google.golang.org/api/option"
)

// "AIzaSyAplXPRfJM65dXwMfWnfq-_8c1gcrfYEiw"

type Payload struct {
	Challenge      string          `json:"challenge"`
	Type           *string         `json:"type"`
	Event          *SlackEvent     `json:"event"`
	Authorizations []Authorization `json:"authorizations"`
}

type SlackEvent struct {
	Text     string `json:"text"`
	Type     string `json:"type"`
	TS       string `json:"ts"`
	ThreadTS string `json:"thread_ts"`
	EventTS  string `json:"event_ts"`
	Channel  string `json:"channel"`
	User     string `json:"user"`
}

type Authorization struct {
	UserID string `json:"user_id"`
}

type ChallengeResp struct {
	Challenge string `json:"challenge"`
}

var (
	botToken     string
	geminiAPIKey string
)

func init() {
	botToken = os.Getenv("BOT_USER_TOKEN")
	geminiAPIKey = os.Getenv("GEMINI_API_KEY")
}

func GeminiSlack(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		HandleErr(w, err)
	}
	// finally must close
	defer r.Body.Close()
	var p *Payload
	err = json.Unmarshal(body, p)
	if err != nil {
		HandleErr(w, err)
	}

	if p.Type != nil && *p.Type == "url_verification" {
		res, err := json.Marshal(ChallengeResp{p.Challenge})
		if err != nil {
			HandleErr(w, err)
		}
		w.WriteHeader(http.StatusOK)
		w.Write(res)
		return
	}

	if p.Event == nil {
		HandleErr(w, errors.New("event not found"))
	}

	ctx := context.Background()
	// Access your API key as an environment variable (see "Set up your API key" above)
	client, err := genai.NewClient(ctx, option.WithAPIKey(geminiAPIKey))
	if err != nil {
		HandleErr(w, err)
	}
	defer client.Close()

	// For text-only input, use the gemini-pro model
	model := client.GenerativeModel("gemini-pro")
	resp, err := model.GenerateContent(ctx, genai.Text("こんにちは"))
	if err != nil {
		HandleErr(w, err)
	}

	reply := fmt.Sprintf("%s", resp.Candidates[0].Content.Parts[0])

	api := slack.New(botToken)
	// If you set debugging, it will log all requests to the console
	// Useful when encountering issues
	// slack.New("YOUR_TOKEN_HERE", slack.OptionDebug(true))
	params := slack.NewPostMessageParameters()
	params.Channel = p.Event.Channel
	params.ThreadTimestamp = p.Event.ThreadTS
	params.User = p.Event.User
	opts := slack.MsgOptionPostMessageParameters(params)
	_, _, err = api.PostMessage(params.Channel, slack.MsgOptionText(reply, true), opts)
	if err != nil {
		HandleErr(w, err)
	}
}

func HandleErr(w http.ResponseWriter, err error) {
	w.WriteHeader(http.StatusInternalServerError)
	res, err := json.Marshal(err)
	if err != nil {
		log.Fatal(err)
		return
	}
	w.Write(res)
}
