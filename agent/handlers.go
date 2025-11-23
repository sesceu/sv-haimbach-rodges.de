package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func handleIndex(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.ParseFiles("templates/index.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	tmpl.Execute(w, nil)
}

func handleGenerateText(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	body, _ := io.ReadAll(r.Body)
	refined, err := GenerateBlogPostRefinement(string(body))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Write([]byte(refined))
}

func handleGenerateImage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Prompt  string `json:"prompt"`
		Context string `json:"context"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	finalPrompt := req.Prompt
	if finalPrompt == "" {
		finalPrompt = "Create an image that fits to the following blog post: " + req.Context
	}

	imgData, err := GenerateImage(finalPrompt)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Save to temp file to serve
	filename := fmt.Sprintf("gen-%d.png", time.Now().Unix())
	path := filepath.Join("..", "static", "img", "temp", filename)

	// Ensure dir exists
	os.MkdirAll(filepath.Dir(path), 0755)

	if err := os.WriteFile(path, imgData, 0644); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Write([]byte("/img/temp/" + filename))
}

func handleUploadImage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	file, header, err := r.FormFile("image")
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	defer file.Close()

	filename := fmt.Sprintf("upload-%d%s", time.Now().Unix(), filepath.Ext(header.Filename))
	path := filepath.Join("..", "static", "img", "temp", filename)

	os.MkdirAll(filepath.Dir(path), 0755)

	out, err := os.Create(path)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer out.Close()

	io.Copy(out, file)

	w.Write([]byte("/img/temp/" + filename))
}

func handleListDefaultImages(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	files, err := os.ReadDir("../static/img/default")
	if err != nil {
		http.Error(w, "Failed to read default images: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Load metadata
	metadata := make(map[string]string)
	metaDataBytes, err := os.ReadFile("../static/img/default/metadata.json")
	if err == nil {
		json.Unmarshal(metaDataBytes, &metadata)
	}

	type DefaultImage struct {
		Path        string `json:"path"`
		Attribution string `json:"attribution"`
	}

	var images []DefaultImage
	for _, f := range files {
		if !f.IsDir() {
			ext := strings.ToLower(filepath.Ext(f.Name()))
			if ext == ".jpg" || ext == ".jpeg" || ext == ".png" || ext == ".gif" || ext == ".webp" {
				images = append(images, DefaultImage{
					Path:        "/img/default/" + f.Name(),
					Attribution: metadata[f.Name()],
				})
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(images)
}

type PreparePostRequest struct {
	Title             string `json:"title"`
	Date              string `json:"date"`
	Content           string `json:"content"`
	ImagePath         string `json:"imagePath"`
	BannerAttribution string `json:"bannerAttribution"`
}

type PreparePostResponse struct {
	BranchName string `json:"branchName"`
	Diff       string `json:"diff"`
}

func handlePreparePost(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req PreparePostRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// 1. Move or Copy image to final location
	// ImagePath is like /img/temp/foo.png or /img/default/bar.jpg

	var finalImgRelPath string
	var filesToCommit []string

	// Check if it's a default image
	if strings.Contains(req.ImagePath, "/img/default/") {
		// Use existing default image directly
		finalImgRelPath = "img/default/" + filepath.Base(req.ImagePath)
		// We do NOT copy the file.
		// We do NOT add the image to git commit (it's already there).
	} else {
		// It's a temp/uploaded/generated image. Move/Copy to img/blog/
		imgName := filepath.Base(req.ImagePath)
		finalImgRelPath = "img/blog/" + imgName
		finalImgPath := filepath.Join("..", "static", "img", "blog", imgName)

		os.MkdirAll(filepath.Dir(finalImgPath), 0755)

		// Source path (remove leading /)
		srcPath := filepath.Join("..", "static", strings.TrimPrefix(req.ImagePath, "/"))

		// Move temp image
		if err := os.Rename(srcPath, finalImgPath); err != nil {
			// If rename fails (e.g. cross-device), try copy
			// Fallback to copy if rename fails
			input, err := os.ReadFile(srcPath)
			if err != nil {
				http.Error(w, "Failed to read temp image: "+err.Error(), http.StatusInternalServerError)
				return
			}
			if err := os.WriteFile(finalImgPath, input, 0644); err != nil {
				http.Error(w, "Failed to write blog image: "+err.Error(), http.StatusInternalServerError)
				return
			}
			// Try to remove temp file, ignore error
			os.Remove(srcPath)
		}

		// Add to git commit
		filesToCommit = append(filesToCommit, filepath.Join("static", finalImgRelPath))
	}

	// 2. Create Markdown file
	// Parse date to get year
	parsedDate, err := time.Parse("2006-01-02", req.Date)
	year := ""
	if err == nil {
		year = fmt.Sprintf("-%d", parsedDate.Year())
	}

	slug := slugify(req.Title)
	mdFilename := fmt.Sprintf("%s%s.md", slug, year)
	mdPath := filepath.Join("..", "content", "blog", mdFilename)

	attributionLine := ""
	if req.BannerAttribution != "" {
		attributionLine = fmt.Sprintf("banner_attribution = \"%s\"\n", req.BannerAttribution)
	}

	mdContent := fmt.Sprintf(`+++
title = '%s'
date = %sT12:00:00+01:00
draft = false
banner = "/%s"
%s+++

%s
`, req.Title, req.Date, finalImgRelPath, attributionLine, req.Content)

	if err := os.WriteFile(mdPath, []byte(mdContent), 0644); err != nil {
		http.Error(w, "Failed to write markdown: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// 3. Git Automation (Local only)
	branchName, err := GitCreateBranch(req.Title)
	if err != nil {
		http.Error(w, "Git branch failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	filesToCommit = append(filesToCommit, filepath.Join("content", "blog", mdFilename))

	if err := GitAddAndCommit(filesToCommit, "Add blog post: "+req.Title); err != nil {
		http.Error(w, "Git commit failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Get Diff
	diff, err := GitDiff()
	if err != nil {
		http.Error(w, "Git diff failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(PreparePostResponse{
		BranchName: branchName,
		Diff:       diff,
	})
}

type PublishPostRequest struct {
	BranchName string `json:"branchName"`
	Title      string `json:"title"` // Needed for PR title
}

func handlePublishPost(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req PublishPostRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := GitPush(req.BranchName); err != nil {
		http.Error(w, "Git push failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// PR
	prURL, err := GitHubCreatePR(req.Title, "Automated blog post creation")
	msg := fmt.Sprintf("Success! Branch '%s' pushed.", req.BranchName)
	if err == nil {
		msg += fmt.Sprintf("\nPR Created: %s", prURL)
	} else {
		msg += fmt.Sprintf("\nCould not create PR automatically: %v\nPlease create it manually.", err)
	}

	w.Write([]byte(msg))
}
