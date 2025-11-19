package main

import (
	"context"
	"fmt"
	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
)

// ListModels prints available models to stdout.
func ListModels() error {
	ctx := context.Background()
	client, err := genai.NewClient(ctx, option.WithAPIKey(cfg.GeminiAPIKey))
	if err != nil {
		return fmt.Errorf("failed to create client: %v", err)
	}
	defer client.Close()

	iter := client.ListModels(ctx)
	for {
		m, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return err
		}
		fmt.Printf("Model: %s\n", m.Name)
	}
	return nil
}

// GenerateBlogPostRefinement uses Gemini to refine the raw text into a blog post.
func GenerateBlogPostRefinement(rawText string) (string, error) {
	ctx := context.Background()
	client, err := genai.NewClient(ctx, option.WithAPIKey(cfg.GeminiAPIKey))
	if err != nil {
		return "", fmt.Errorf("failed to create client: %v", err)
	}
	defer client.Close()

	// Use configured model
	modelName := cfg.TextModel

	model := client.GenerativeModel(modelName)
	model.SetTemperature(0.7)

	prompt := fmt.Sprintf(`
You are a helpful assistant for a shooting club (Schützenverein) "Gut Schuß Haimbach/Rodges". 
Please rewrite the following raw text into a friendly, engaging blog post in German. 
Keep the tone professional but welcoming. Use the "Wir" (We) form.
The style should be similar to: "Am vergangenen Sonntag... haben wir... Wir möchten uns bedanken...". Don't be too verbose.
End with a bold summary sentence or looking forward statement.
Do not add a title or frontmatter, just the body text.
Use Markdown formatting.

Raw Text:
%s
`, rawText)

	resp, err := model.GenerateContent(ctx, genai.Text(prompt))
	if err != nil {
		return "", fmt.Errorf("failed to generate content: %v", err)
	}

	if len(resp.Candidates) == 0 || len(resp.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("no content generated")
	}

	part := resp.Candidates[0].Content.Parts[0]
	if txt, ok := part.(genai.Text); ok {
		return string(txt), nil
	}

	return "", fmt.Errorf("unexpected response format")
}

// GenerateImage uses Gemini (Imagen) to generate an image.
func GenerateImage(prompt string) ([]byte, error) {
	ctx := context.Background()
	client, err := genai.NewClient(ctx, option.WithAPIKey(cfg.GeminiAPIKey))
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %v", err)
	}
	defer client.Close()

	modelName := cfg.ImageModel
	model := client.GenerativeModel(modelName)
	
	resp, err := model.GenerateContent(ctx, genai.Text(prompt))
	if err != nil {
		return nil, fmt.Errorf("failed to generate image: %v", err)
	}

	if len(resp.Candidates) == 0 || len(resp.Candidates[0].Content.Parts) == 0 {
		return nil, fmt.Errorf("no image generated")
	}

	for _, part := range resp.Candidates[0].Content.Parts {
		if blob, ok := part.(genai.Blob); ok {
			if blob.MIMEType == "image/png" || blob.MIMEType == "image/jpeg" {
				return blob.Data, nil
			}
		}
	}
	
	return nil, fmt.Errorf("no image data found in response")
}
