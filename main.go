package main

import (
	"net/http"

	"github.com/maito1201/gemini-slack/gcp"
)

func main() {
	http.HandleFunc("/", gcp.GeminiSlack)
	http.ListenAndServe(":8080", nil)
}
