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

type CreatePostRequest struct {
	Title     string `json:"title"`
	Date      string `json:"date"`
	Content   string `json:"content"`
	ImagePath string `json:"imagePath"`
}

func handleCreatePost(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req CreatePostRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// 1. Move image to final location
	// ImagePath is like /img/temp/foo.png. We want to move it to /img/blog/year/foo.png or similar.
	// For simplicity, let's put it in static/img/blog/
	
	imgName := filepath.Base(req.ImagePath)
	finalImgRelPath := "img/blog/" + imgName
	finalImgPath := filepath.Join("..", "static", "img", "blog", imgName)
	
	os.MkdirAll(filepath.Dir(finalImgPath), 0755)
	
	// Source path (remove leading /)
	srcPath := filepath.Join("..", "static", strings.TrimPrefix(req.ImagePath, "/"))
	
	if err := os.Rename(srcPath, finalImgPath); err != nil {
		// If rename fails (e.g. cross-device), try copy
		// For now, just error out or assume it works as it's likely same volume
		http.Error(w, "Failed to move image: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// 2. Create Markdown file
	slug := slugify(req.Title)
	mdFilename := fmt.Sprintf("%s.md", slug)
	mdPath := filepath.Join("..", "content", "blog", mdFilename)

	mdContent := fmt.Sprintf(`+++
title = '%s'
date = %sT12:00:00+01:00
draft = false
banner = "/%s"
+++

%s
`, req.Title, req.Date, finalImgRelPath, req.Content)

	if err := os.WriteFile(mdPath, []byte(mdContent), 0644); err != nil {
		http.Error(w, "Failed to write markdown: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// 3. Git Automation
	branchName, err := GitCreateBranch(req.Title)
	if err != nil {
		http.Error(w, "Git branch failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	files := []string{
		filepath.Join("content", "blog", mdFilename),
		filepath.Join("static", finalImgRelPath),
	}
	
	if err := GitAddAndCommit(files, "Add blog post: "+req.Title); err != nil {
		http.Error(w, "Git commit failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	if err := GitPush(branchName); err != nil {
		http.Error(w, "Git push failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// 4. PR
	prURL, err := GitHubCreatePR(req.Title, "Automated blog post creation")
	msg := fmt.Sprintf("Success! Branch '%s' pushed.", branchName)
	if err == nil {
		msg += fmt.Sprintf("\nPR Created: %s", prURL)
	} else {
		msg += fmt.Sprintf("\nCould not create PR automatically: %v\nPlease create it manually.", err)
	}

	w.Write([]byte(msg))
}
