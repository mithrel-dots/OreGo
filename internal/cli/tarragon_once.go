package cli

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	_ "github.com/mattn/go-sqlite3"
	"github.com/spf13/cobra"
	"orego/internal/db"
)

const tarragonOnceLimit = 50
const tarragonSearchScanLimit = 250

type tarragonResultItem struct {
	ID          string           `json:"id"`
	Label       string           `json:"label"`
	Description string           `json:"description,omitempty"`
	Category    string           `json:"category,omitempty"`
	PreviewPath string           `json:"preview_path,omitempty"`
	Actions     []tarragonAction `json:"actions,omitempty"`
	Score       float64          `json:"score,omitempty"`
}

type tarragonAction struct {
	Name    string `json:"name"`
	Default bool   `json:"default,omitempty"`
}

type tarragonSearchResponse struct {
	Results []tarragonResultItem `json:"results"`
}

type screenshotCandidate struct {
	ID       int64
	Path     string
	Class    string
	Title    string
	Score    float64
	TieBreak int64
}

func runTarragonOnce(cmd *cobra.Command, query string) error {
	results := searchScreenshots(strings.TrimSpace(query))

	resp := tarragonSearchResponse{Results: make([]tarragonResultItem, 0, len(results))}
	for _, r := range results {
		resp.Results = append(resp.Results, tarragonResultItem{
			ID:          strconv.FormatInt(r.ID, 10),
			Label:       formatResultLabel(r.Class, r.Title, r.Path),
			Description: formatResultDescription(r.Class, r.Title, r.Path),
			Category:    "screenshots",
			PreviewPath: r.Path,
			Actions: []tarragonAction{
				{Name: "open", Default: true},
				{Name: "delete"},
			},
			Score: r.Score,
		})
	}

	enc := json.NewEncoder(cmd.OutOrStdout())
	return enc.Encode(resp)
}

func runTarragonSelect(cmd *cobra.Command, resultID string, action string) error {
	id, err := strconv.ParseInt(strings.TrimSpace(resultID), 10, 64)
	if err != nil {
		return fmt.Errorf("invalid result id %q: %w", resultID, err)
	}

	selectedAction := strings.TrimSpace(action)
	if selectedAction == "" || selectedAction == "execute" {
		selectedAction = "open"
	}

	switch selectedAction {
	case "open":
		if err := openScreenshotByID(id); err != nil {
			return err
		}
	case "delete":
		if err := deleteScreenshotByID(id); err != nil {
			return err
		}
	default:
		return fmt.Errorf("unsupported action %q", selectedAction)
	}

	return json.NewEncoder(cmd.OutOrStdout()).Encode(map[string]any{
		"ok":        true,
		"action":    selectedAction,
		"result_id": strconv.FormatInt(id, 10),
	})
}

func openScreenshotByID(id int64) error {
	path, err := getScreenshotPathByID(id)
	if err != nil {
		return err
	}
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("file no longer exists: %s", path)
		}
		return err
	}
	if err := exec.Command("xdg-open", path).Start(); err != nil {
		return fmt.Errorf("open screenshot: %w", err)
	}
	return nil
}

func deleteScreenshotByID(id int64) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	dbPath := filepath.Join(homeDir, ".local", "share", "orego", "orego.db")
	store, err := db.New(dbPath)
	if err != nil {
		return err
	}
	defer store.Close()

	if err := store.DeleteScreenshot(id); err != nil {
		return err
	}
	return nil
}

func getScreenshotPathByID(id int64) (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	dbPath := filepath.Join(homeDir, ".local", "share", "orego", "orego.db")
	store, err := db.New(dbPath)
	if err != nil {
		return "", err
	}
	defer store.Close()

	return store.GetScreenshotPath(id)
}

func searchScreenshots(query string) []screenshotCandidate {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil
	}

	dbPath := filepath.Join(homeDir, ".local", "share", "orego", "orego.db")
	if _, err := os.Stat(dbPath); err != nil {
		return nil
	}

	dsn := fmt.Sprintf("file:%s?mode=ro", dbPath)
	dbConn, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil
	}
	defer dbConn.Close()

	rows, err := dbConn.Query(`
		SELECT id, file_path, active_window_class, active_window_title
		FROM screenshots
		ORDER BY id DESC
		LIMIT ?
	`, tarragonSearchScanLimit)
	if err != nil {
		return nil
	}
	defer rows.Close()

	q := strings.ToLower(strings.TrimSpace(query))
	hits := make([]screenshotCandidate, 0, tarragonOnceLimit)

	for rows.Next() {
		var c screenshotCandidate
		if err := rows.Scan(&c.ID, &c.Path, &c.Class, &c.Title); err != nil {
			continue
		}
		c.TieBreak = c.ID

		c.Score = scoreCandidate(c, q)
		if q != "" && c.Score == 0 {
			continue
		}
		hits = append(hits, c)
	}

	sort.Slice(hits, func(i, j int) bool {
		if hits[i].Score == hits[j].Score {
			return hits[i].TieBreak > hits[j].TieBreak
		}
		return hits[i].Score > hits[j].Score
	})

	if len(hits) > tarragonOnceLimit {
		hits = hits[:tarragonOnceLimit]
	}

	return hits
}

func scoreCandidate(c screenshotCandidate, q string) float64 {
	if q == "" {
		return 1.0
	}

	title := strings.ToLower(c.Title)
	class := strings.ToLower(c.Class)
	base := strings.ToLower(filepath.Base(c.Path))

	score := 0.0
	if strings.Contains(title, q) {
		score += 3.0
	}
	if strings.Contains(class, q) {
		score += 2.0
	}
	if strings.Contains(base, q) {
		score += 1.0
	}
	if strings.HasPrefix(title, q) {
		score += 1.0
	}

	return score
}

func formatResultLabel(class string, title string, filePath string) string {
	cleanClass := strings.TrimSpace(class)
	cleanTitle := strings.TrimSpace(title)

	switch {
	case cleanClass != "" && cleanTitle != "":
		return cleanClass + " - " + cleanTitle
	case cleanTitle != "":
		return cleanTitle
	case cleanClass != "":
		return cleanClass
	default:
		return filepath.Base(filePath)
	}
}

func formatResultDescription(class string, title string, filePath string) string {
	parts := make([]string, 0, 2)
	if strings.TrimSpace(class) != "" {
		parts = append(parts, strings.TrimSpace(class))
	}
	if strings.TrimSpace(title) != "" {
		parts = append(parts, strings.TrimSpace(title))
	}
	parts = append(parts, filepath.Base(filePath))
	return strings.Join(parts, " | ")
}
