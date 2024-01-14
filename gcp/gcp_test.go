package gcp

import (
	"testing"

	"github.com/google/generative-ai-go/genai"
	"github.com/google/go-cmp/cmp"
	"github.com/slack-go/slack"
	"github.com/stretchr/testify/assert"
)

func TestBuildChatHistory(t *testing.T) {
	UserID := "user"
	BotID := "bot"
	tests := map[string]struct {
		msgs       []slack.Message
		want       []*genai.Content
		inputText  string
		outputText string
	}{
		"normal": {
			msgs: []slack.Message{
				{Msg: slack.Msg{User: UserID, Text: "text1"}},
				{Msg: slack.Msg{User: BotID, Text: "answer1"}},
				{Msg: slack.Msg{User: UserID, Text: "text2"}},
			},
			want: []*genai.Content{
				{Parts: []genai.Part{genai.Text("text1")}, Role: "user"},
				{Parts: []genai.Part{genai.Text("answer1")}, Role: "model"},
			},
			inputText:  "test",
			outputText: "test",
		},
		"no history": {
			msgs:       []slack.Message{},
			want:       []*genai.Content{},
			inputText:  "test",
			outputText: "test",
		},
		"user post twice before": {
			msgs: []slack.Message{
				{Msg: slack.Msg{User: UserID, Text: "text1"}},
				{Msg: slack.Msg{User: UserID, Text: "text2"}},
				{Msg: slack.Msg{User: BotID, Text: "answer1"}},
				{Msg: slack.Msg{User: UserID, Text: "text3"}},
			},
			want: []*genai.Content{
				{Parts: []genai.Part{genai.Text("text1, text2")}, Role: "user"},
				{Parts: []genai.Part{genai.Text("answer1")}, Role: "model"},
			},
			inputText:  "test",
			outputText: "test",
		},
		"user post twice recent": {
			msgs: []slack.Message{
				{Msg: slack.Msg{User: UserID, Text: "text1"}},
				{Msg: slack.Msg{User: BotID, Text: "answer1"}},
				{Msg: slack.Msg{User: UserID, Text: "text2"}},
				{Msg: slack.Msg{User: UserID, Text: "test"}},
			},
			want: []*genai.Content{
				{Parts: []genai.Part{genai.Text("text1")}, Role: "user"},
				{Parts: []genai.Part{genai.Text("answer1")}, Role: "model"},
			},
			inputText:  "test",
			outputText: "text2, test",
		},
		"user post 3times": {
			msgs: []slack.Message{
				{Msg: slack.Msg{User: UserID, Text: "text1"}},
				{Msg: slack.Msg{User: UserID, Text: "text2"}},
				{Msg: slack.Msg{User: UserID, Text: "text3"}},
				{Msg: slack.Msg{User: BotID, Text: "answer1"}},
				{Msg: slack.Msg{User: UserID, Text: "text4"}},
				{Msg: slack.Msg{User: UserID, Text: "text5"}},
				{Msg: slack.Msg{User: UserID, Text: "test"}},
			},
			want: []*genai.Content{
				{Parts: []genai.Part{genai.Text("text1, text2, text3")}, Role: "user"},
				{Parts: []genai.Part{genai.Text("answer1")}, Role: "model"},
			},
			inputText:  "test",
			outputText: "text4, text5, test",
		},
	}
	for testName, arg := range tests {
		arg := arg
		t.Run(testName, func(t *testing.T) {
			got, outputText := buildChatHistory(arg.msgs, BotID, arg.inputText)
			if diff := cmp.Diff(got, arg.want); diff != "" {
				t.Errorf("User value is mismatch (-got +want):\n%s", diff)
			}
			assert.Equal(t, arg.outputText, outputText)
		})
	}
}
