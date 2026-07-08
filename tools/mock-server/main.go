package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
)

func main() {
	port := "9090"
	if p := os.Getenv("PORT"); p != "" {
		port = p
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/mock/chat/completions", handleChatCompletion)
	mux.HandleFunc("/mock/models", handleListModels)
	mux.HandleFunc("/mock/", handleNotFound)

	addr := fmt.Sprintf(":%s", port)
	log.Printf("Mock provider listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}

func handleChatCompletion(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"id":      "mock-chatcmpl-123",
		"object":  "chat.completion",
		"created": 1677652288,
		"model":   "gpt-4",
		"choices": []map[string]interface{}{
			{
				"index":         0,
				"finish_reason": "stop",
				"message": map[string]string{
					"role":    "assistant",
					"content": "Mock response from ai-gateway",
				},
			},
		},
		"usage": map[string]int{
			"prompt_tokens":     10,
			"completion_tokens": 10,
			"total_tokens":      20,
		},
	})
}

func handleListModels(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"object": "list",
		"data": []map[string]string{
			{"id": "gpt-4", "object": "model"},
			{"id": "gpt-4-turbo", "object": "model"},
			{"id": "gpt-3.5-turbo", "object": "model"},
		},
	})
}

func handleNotFound(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNotFound)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"error": map[string]interface{}{
			"message": "Not found",
			"type":    "not_found",
			"code":    404,
		},
	})
}
