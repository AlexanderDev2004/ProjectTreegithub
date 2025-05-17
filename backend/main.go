package main

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

type FileNode struct {
	Name     string      `json:"name"`
	IsDir    bool        `json:"is_dir"`
	Children []*FileNode `json:"children,omitempty"`
}

func main() {
	http.HandleFunc("/tree", handleRepoTree)
	http.Handle("/", http.FileServer(http.Dir("../frontend"))) // Serve static files (index.html, style.css, script.js)

	fmt.Println("Server running at http://localhost:8080")
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		fmt.Println("Error starting server:", err)
	}
}

func handleRepoTree(w http.ResponseWriter, r *http.Request) {
	repoURL := r.URL.Query().Get("url")
	if repoURL == "" {
		http.Error(w, "Missing 'url' parameter", http.StatusBadRequest)
		return
	}

	zipURL := convertGitHubToZipURL(repoURL)
	if zipURL == "" {
		http.Error(w, "Invalid GitHub URL", http.StatusBadRequest)
		return
	}

	tmpDir, err := ioutil.TempDir("", "repo")
	if err != nil {
		http.Error(w, "Failed to create temp dir", 500)
		return
	}
	defer os.RemoveAll(tmpDir)

	zipPath := filepath.Join(tmpDir, "repo.zip")
	if err := downloadFile(zipURL, zipPath); err != nil {
		http.Error(w, "Failed to download repo", 500)
		return
	}

	dirPath := filepath.Join(tmpDir, "unzipped")
	if err := unzip(zipPath, dirPath); err != nil {
		http.Error(w, "Failed to unzip repo", 500)
		return
	}

	files, err := ioutil.ReadDir(dirPath)
	if err != nil || len(files) == 0 {
		http.Error(w, "Empty or invalid repo content", 500)
		return
	}

	rootNode := buildTree(filepath.Join(dirPath, files[0].Name()))
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(rootNode)
}

func convertGitHubToZipURL(url string) string {
	if !strings.HasPrefix(url, "https://github.com/") {
		return ""
	}
	parts := strings.Split(strings.TrimPrefix(url, "https://github.com/"), "/")
	if len(parts) < 2 {
		return ""
	}
	return fmt.Sprintf("https://github.com/%s/%s/archive/refs/heads/main.zip", parts[0], parts[1])
}

func downloadFile(url, dest string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	out, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	return err
}

func unzip(src, dest string) error {
	r, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer r.Close()

	os.MkdirAll(dest, 0755)

	for _, f := range r.File {
		fpath := filepath.Join(dest, f.Name)
		if f.FileInfo().IsDir() {
			os.MkdirAll(fpath, f.Mode())
			continue
		}
		if err := os.MkdirAll(filepath.Dir(fpath), 0755); err != nil {
			return err
		}
		outFile, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return err
		}
		rc, err := f.Open()
		if err != nil {
			return err
		}
		_, err = io.Copy(outFile, rc)
		outFile.Close()
		rc.Close()
		if err != nil {
			return err
		}
	}
	return nil
}

func buildTree(path string) *FileNode {
	info, err := os.Stat(path)
	if err != nil {
		return nil
	}
	node := &FileNode{Name: info.Name(), IsDir: info.IsDir()}
	if info.IsDir() {
		entries, _ := ioutil.ReadDir(path)
		for _, e := range entries {
			child := buildTree(filepath.Join(path, e.Name()))
			if child != nil {
				node.Children = append(node.Children, child)
			}
		}
	}
	return node
}
