package main

import (
	"context"
	"fmt"
	"google.golang.org/genai"
)

// GenerateBlogPostRefinement uses Gemini to refine the raw text into a blog post.
func GenerateBlogPostRefinement(rawText string) (string, error) {
	ctx := context.Background()
	// Backend defaults to GeminiAPI, so we can omit it or use "gemini" if string, but let's rely on default for now or check doc result.
	// actually, let's just pass APIKey.
	client, err := genai.NewClient(ctx, &genai.ClientConfig{APIKey: cfg.GeminiAPIKey})
	if err != nil {
		return "", fmt.Errorf("failed to create client: %v", err)
	}

	// Use configured model
	modelName := cfg.TextModel

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

	resp, err := client.Models.GenerateContent(ctx, modelName, genai.Text(prompt), nil)
	if err != nil {
		return "", fmt.Errorf("failed to generate content: %v", err)
	}

	if len(resp.Candidates) == 0 || len(resp.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("no content generated")
	}

	part := resp.Candidates[0].Content.Parts[0]
	// Part is a struct, check Text field
	if part.Text != "" {
		return part.Text, nil
	}

	return "", fmt.Errorf("unexpected response format: no text found")
}

// GenerateImage uses Gemini (Imagen) to generate an image.
func GenerateImage(prompt string) ([]byte, error) {
	ctx := context.Background()
	client, err := genai.NewClient(ctx, &genai.ClientConfig{APIKey: cfg.GeminiAPIKey})
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %v", err)
	}

	modelName := cfg.ImageModel
	
	// Use GenerateImages method from Models service
	resp, err := client.Models.GenerateImages(ctx, modelName, prompt, &genai.GenerateImagesConfig{
		NumberOfImages: 1,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to generate image with model %s: %v", modelName, err)
	}

	if len(resp.GeneratedImages) == 0 {
		return nil, fmt.Errorf("no image generated")
	}

	genImg := resp.GeneratedImages[0]
	if genImg.Image == nil {
		return nil, fmt.Errorf("generated image is nil")
	}

	// Image struct likely has ImageBytes or similar. 
	// Based on common patterns and previous errors, it might be ImageBytes.
	// Let's wait for go doc output to be sure, but I can try to guess if it's standard.
	// Actually, I'll use the doc output from the previous step (which I haven't seen yet in this turn, but will appear).
	// To be safe, I will assume it is ImageBytes based on similar Google APIs.
	// If not, I will fix it after build.
	return genImg.Image.ImageBytes, nil
}
