package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
)

type Config struct {
	GeminiAPIKey string
	TextModel    string
	ImageModel   string
	Port         string
}

var cfg Config

func main() {
	// Parse flags
	apiKeyFlag := flag.String("key", "", "Gemini API Key")
	textModelFlag := flag.String("text-model", "gemini-2.5-flash", "Gemini Text Model to use")
	imageModelFlag := flag.String("image-model", "imagen-4.0-fast-generate-001", "Gemini Image Model to use")
	portFlag := flag.String("port", "8080", "Port to run the server on")
	listModelsFlag := flag.Bool("list-models", false, "List available Gemini models and exit")
	flag.Parse()

	// Load config
	cfg.GeminiAPIKey = *apiKeyFlag
	if cfg.GeminiAPIKey == "" {
		cfg.GeminiAPIKey = os.Getenv("GEMINI_API_KEY")
	}
	
	cfg.TextModel = *textModelFlag
	if cfg.TextModel == "gemini-2.5-flash" && os.Getenv("GEMINI_TEXT_MODEL") != "" {
		cfg.TextModel = os.Getenv("GEMINI_TEXT_MODEL")
	}

	cfg.ImageModel = *imageModelFlag
	if cfg.ImageModel == "imagen-4.0-fast-generate-001" && os.Getenv("GEMINI_IMAGE_MODEL") != "" {
		cfg.ImageModel = os.Getenv("GEMINI_IMAGE_MODEL")
	}

	cfg.Port = *portFlag

	if cfg.GeminiAPIKey == "" {
		log.Fatal("Gemini API Key is required. Set GEMINI_API_KEY env var or use -key flag.")
	}

	if *listModelsFlag {
		fmt.Println("Listing available models...")
		if err := ListModels(); err != nil {
			log.Fatalf("Failed to list models: %v", err)
		}
		return
	}

	// Setup routes
	http.HandleFunc("/", handleIndex)
	http.HandleFunc("/generate-text", handleGenerateText)
	http.HandleFunc("/generate-image", handleGenerateImage)
	http.HandleFunc("/upload-image", handleUploadImage)
	http.HandleFunc("/create-post", handleCreatePost)

	// Start server
	fmt.Printf("Agent started on http://localhost:%s\n", cfg.Port)
	log.Fatal(http.ListenAndServe(":"+cfg.Port, nil))
}
