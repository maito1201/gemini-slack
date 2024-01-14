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
	"regexp"
	"time"

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

var mentionRxp = regexp.MustCompile("<@.*>")

func GeminiSlack(w http.ResponseWriter, r *http.Request) {
	p, doNext := handleParameter(w, r)
	if !doNext {
		return
	}
	go func(p *Payload) {
		ctx, _ := context.WithTimeout(context.Background(), time.Minute*3)
		// Initialize Slack
		api := slack.New(botToken)

		// Initialize Gemini API
		client, err := genai.NewClient(ctx, option.WithAPIKey(geminiAPIKey))
		if err != nil {
			log.Fatalf("genai.NewClient %s", err)
		}
		defer client.Close()
		model := client.GenerativeModel("gemini-pro")
		cs := model.StartChat()
		inputText := mentionRxp.ReplaceAllString(p.Event.Text, "")

		// Get Reply History
		replyParams := &slack.GetConversationRepliesParameters{
			ChannelID:          p.Event.Channel,
			IncludeAllMetadata: false,
		}
		if p.Event.ThreadTS != "" {
			replyParams.Timestamp = p.Event.ThreadTS
		} else if p.Event.TS != "" {
			replyParams.Timestamp = p.Event.TS
		}

		// Build Chat History
		msgs, _, _, err := api.GetConversationRepliesContext(ctx, replyParams)
		history, newText := buildChatHistory(msgs, p.Authorizations[0].UserID, inputText)
		cs.History = history
		inputText = newText
		// Send Chat

		resp, err := cs.SendMessage(ctx, genai.Text(inputText))
		if err != nil {
			log.Printf("cs.SendMessage %s", err)
			fmt.Printf("%+v\n", resp)
			return
		}

		// Post Reply
		reply := fmt.Sprintf("<@%s> %s", p.Event.User, resp.Candidates[0].Content.Parts[0])
		_, _, err = api.PostMessageContext(
			ctx,
			p.Event.Channel,
			slack.MsgOptionText(reply, false),
			slack.MsgOptionTS(p.Event.EventTS),
		)
		if err != nil {
			log.Printf("api.PostMessage %s", err)
			return
		}
	}(p)
}

func handleParameter(w http.ResponseWriter, r *http.Request) (*Payload, bool) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		handleErr(w, err)
		return nil, false
	}
	defer r.Body.Close()
	p := &Payload{}
	err = json.Unmarshal(body, p)
	if err != nil {
		handleErr(w, err)
		return nil, false
	}

	if p == nil {
		handleErr(w, errors.New("parameter not found"))
		return nil, false
	}

	if p.Type != nil && *p.Type == "url_verification" {
		res, err := json.Marshal(ChallengeResp{p.Challenge})
		if err != nil {
			handleErr(w, err)
			return nil, false
		}
		w.WriteHeader(http.StatusOK)
		w.Write(res)
		return nil, false
	}

	if p.Event == nil {
		handleErr(w, errors.New("event not found"))
		return nil, false
	}
	if p.Event.Type != "app_mention" || p.Event.Text == "" {
		w.WriteHeader(http.StatusOK)
		return nil, false
	}
	w.WriteHeader(http.StatusOK)
	return p, true
}

func handleErr(w http.ResponseWriter, err error) {
	w.WriteHeader(http.StatusInternalServerError)
	w.Header().Add("X-Slack-No-Retry", "1")
	log.Printf("error %v\n", err)
	w.Write([]byte(err.Error()))
}

func buildChatHistory(msgs []slack.Message, botID string, inputText string) ([]*genai.Content, string) {
	if len(msgs) > 0 {
		msgs = msgs[:len(msgs)-1]
	}
	histories := []*genai.Content{}
	// Merge consecutive posts of users
	for i, v := range msgs {
		role := "user"
		t := mentionRxp.ReplaceAllString(v.Text, "")

		if v.User == botID {
			role = "model"
		}
		history := &genai.Content{
			Parts: []genai.Part{
				genai.Text(t),
			},
			Role: role,
		}
		if i > 0 && len(histories) >= 1 {
			previousContent := histories[len(histories)-1]
			if previousContent.Role == role {
				previousContent.Parts = []genai.Part{genai.Text(fmt.Sprintf("%s, %s", previousContent.Parts[0], t))}
			} else {
				histories = append(histories, history)
			}
		} else {
			histories = append(histories, history)
		}
	}
	// If user post is last, It is merged to input Text
	if len(histories) > 0 && histories[len(histories)-1].Role == "user" {
		newText := fmt.Sprintf("%s, %s", histories[len(histories)-1].Parts[0], inputText)
		inputText = newText
		histories = histories[:len(histories)-1]
	}
	return histories, inputText
}
