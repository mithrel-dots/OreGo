package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"

	"github.com/spf13/cobra"
	"orego/internal/db"
)

var copyCmd = &cobra.Command{
	Use:   "copy [id]",
	Short: "Copy a screenshot to the clipboard",
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			return fmt.Errorf("accepts 1 arg(s), received %d", len(args))
		}
		return nil
	},
	Run: runCopy,
}

func init() {
	rootCmd.AddCommand(copyCmd)
}

func runCopy(cmd *cobra.Command, args []string) {
	id, err := strconv.ParseInt(args[0], 10, 64)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Invalid ID: %v\n", err)
		os.Exit(1)
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting home dir: %v\n", err)
		os.Exit(1)
	}

	dbPath := filepath.Join(homeDir, ".local/share/orego/orego.db")
	store, err := db.New(dbPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing DB: %v\n", err)
		os.Exit(1)
	}
	defer store.Close()

	path, err := store.GetScreenshotPath(id)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Fprintf(os.Stderr, "File no longer exists: %s\n", path)
			fmt.Println("Tip: Run 'orego cleanup' to remove stale records.")
			os.Exit(1)
		}
		fmt.Fprintf(os.Stderr, "Error opening file: %v\n", err)
		os.Exit(1)
	}
	defer file.Close()

	copyCmd := exec.Command("wl-copy", "--type", "image/png")
	copyCmd.Stdin = file
	if err := copyCmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error copying to clipboard: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Copied screenshot to clipboard.")
}
