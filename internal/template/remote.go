package template

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

const (
	// DefaultRemoteURL is the default remote template repository URL.
	DefaultRemoteURL = "https://raw.githubusercontent.com/Li-Qifeng/subai/main/templates"

	// cacheSubdir is the subdirectory under the user's home for template cache.
	cacheSubdir = ".subai/templates"
)

// cacheDir returns the local template cache directory, creating it if needed.
func cacheDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("cannot get home dir: %w", err)
	}
	dir := filepath.Join(home, cacheSubdir)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("cannot create cache dir %s: %w", dir, err)
	}
	return dir, nil
}

// getWithTimeout fetches a URL with a 30-second timeout.
func getWithTimeout(url string) ([]byte, error) {
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("GET %s failed: %w", url, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GET %s returned %d", url, resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read %s failed: %w", url, err)
	}
	return body, nil
}

// TemplateIndexEntry describes one template in the remote index.
type TemplateIndexEntry struct {
	Name        string `json:"name"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Category    string `json:"category"`
	File        string `json:"file"`
}

// syncIndex downloads the remote index.json and saves it to the cache dir.
func syncIndex(remoteURL string, dir string) ([]TemplateIndexEntry, error) {
	indexURL := remoteURL + "/index.json"
	body, err := getWithTimeout(indexURL)
	if err != nil {
		return nil, fmt.Errorf("fetch index failed: %w", err)
	}

	var index []TemplateIndexEntry
	if err := json.Unmarshal(body, &index); err != nil {
		return nil, fmt.Errorf("parse index.json failed: %w", err)
	}

	idxPath := filepath.Join(dir, "index.json")
	if err := os.WriteFile(idxPath, body, 0644); err != nil {
		return nil, fmt.Errorf("write index.json failed: %w", err)
	}

	return index, nil
}

// SyncTemplates fetches the latest template index and template files
// from the remote repository and caches them locally.
func SyncTemplates(remoteURL string) error {
	if remoteURL == "" {
		remoteURL = DefaultRemoteURL
	}

	dir, err := cacheDir()
	if err != nil {
		return err
	}

	fmt.Fprintf(ruleLogWriter, "  📥 Syncing templates from %s ...\n", remoteURL)

	index, err := syncIndex(remoteURL, dir)
	if err != nil {
		return err
	}

	downloaded := 0
	for _, entry := range index {
		fileURL := remoteURL + "/" + entry.File
		body, err := getWithTimeout(fileURL)
		if err != nil {
			fmt.Fprintf(ruleLogWriter, "  ⚠️  %s: %v\n", entry.Name, err)
			continue
		}
		dst := filepath.Join(dir, entry.File)
		if err := os.WriteFile(dst, body, 0644); err != nil {
			fmt.Fprintf(ruleLogWriter, "  ⚠️  %s: write failed: %v\n", entry.Name, err)
			continue
		}
		downloaded++
	}

	fmt.Fprintf(ruleLogWriter, "  ✅ Synced %d/%d templates\n", downloaded, len(index))
	return nil
}

// ListCachedTemplates returns template entries from the local cache index.
// Returns nil if no cache exists.
func ListCachedTemplates() ([]TemplateIndexEntry, error) {
	dir, err := cacheDir()
	if err != nil {
		return nil, err
	}

	idxPath := filepath.Join(dir, "index.json")
	if _, err := os.Stat(idxPath); os.IsNotExist(err) {
		return nil, nil // no cache yet
	}

	body, err := os.ReadFile(idxPath)
	if err != nil {
		return nil, fmt.Errorf("read cache index failed: %w", err)
	}

	var index []TemplateIndexEntry
	if err := json.Unmarshal(body, &index); err != nil {
		return nil, fmt.Errorf("parse cache index failed: %w", err)
	}

	return index, nil
}

// LoadCachedTemplate loads a template config from the local cache by name.
func LoadCachedTemplate(name string) (*Config, error) {
	dir, err := cacheDir()
	if err != nil {
		return nil, err
	}

	entry, err := findCachedEntry(name)
	if err != nil {
		return nil, err
	}

	path := filepath.Join(dir, entry.File)
	body, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read cached template %s: %w", name, err)
	}

	return parseTemplateYAML(body)
}

// findCachedEntry looks up a template entry in the cached index by name.
func findCachedEntry(name string) (*TemplateIndexEntry, error) {
	index, err := ListCachedTemplates()
	if err != nil {
		return nil, err
	}
	if index == nil {
		return nil, fmt.Errorf("no cached templates (run 'subai template sync' first)")
	}

	for _, entry := range index {
		if entry.Name == name || entry.File == name+".yaml" {
			return &entry, nil
		}
	}
	return nil, fmt.Errorf("template %q not found in cache", name)
}

// parseTemplateYAML unmarshals a template YAML body (with optional --- frontmatter) into Config.
func parseTemplateYAML(body []byte) (*Config, error) {
	// Strip YAML frontmatter (lines between --- markers)
	clean := body
	if len(body) > 3 && body[0] == '-' && body[1] == '-' && body[2] == '-' {
		parts := splitFrontMatter(clean)
		if len(parts) == 2 {
			clean = parts[1]
		}
	}

	var cfg Config
	if err := yaml.Unmarshal(clean, &cfg); err != nil {
		return nil, fmt.Errorf("parse template YAML: %w", err)
	}

	return &cfg, nil
}

// splitFrontMatter splits YAML frontmatter (--- ... ---) from the body.
func splitFrontMatter(data []byte) [][]byte {
	// Find the first ---
	lines := splitLines(data)
	if len(lines) < 2 {
		return nil
	}
	if string(lines[0]) != "---" {
		return nil
	}
	// Find the closing ---
	for i := 1; i < len(lines); i++ {
		if string(lines[i]) == "---" {
			// frontmatter is lines[1:i], body is lines[i+1:]
			var body [][]byte
			for j := i + 1; j < len(lines); j++ {
				body = append(body, lines[j])
			}
			return [][]byte{joinLines(lines[1:i]), joinLines(body)}
		}
	}
	return nil
}

// splitLines splits a byte slice into lines.
func splitLines(data []byte) [][]byte {
	var lines [][]byte
	start := 0
	for i, b := range data {
		if b == '\n' {
			// Trim trailing \r
			line := data[start:i]
			if len(line) > 0 && line[len(line)-1] == '\r' {
				line = line[:len(line)-1]
			}
			lines = append(lines, line)
			start = i + 1
		}
	}
	if start < len(data) {
		lines = append(lines, data[start:])
	}
	return lines
}

// joinLines joins byte slices with newlines.
func joinLines(lines [][]byte) []byte {
	var result []byte
	for i, line := range lines {
		if i > 0 {
			result = append(result, '\n')
		}
		result = append(result, line...)
	}
	return result
}