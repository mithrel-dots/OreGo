package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"orego/internal/db"
	"orego/internal/tui"
)

var (
	filterField string
	filterValue string
	useTui      bool
	useTv       bool
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List recent screenshots or open TUI",
	Run:   runList,
}

func init() {
	listCmd.Flags().StringVar(&filterField, "filter-by", "", "Field to filter by (app, title)")
	listCmd.Flags().StringVar(&filterValue, "value", "", "Value to search for")
	listCmd.Flags().BoolVar(&useTui, "tui", false, "Open interactive TUI")
	listCmd.Flags().BoolVar(&useTv, "tv", false, "Output tab-separated rows for television")
	rootCmd.AddCommand(listCmd)
}

func runList(cmd *cobra.Command, args []string) {
	if filterValue == "" && len(args) > 0 {
		filterValue = args[0]
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

	if useTv {
		screenshots, err := store.ListScreenshots(0, "", "")
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error listing screenshots: %v\n", err)
			os.Exit(1)
		}
		for _, sc := range screenshots {
			fmt.Fprintf(os.Stdout, "%d\t%s\t%s\t%s\t%s\n",
				sc.ID,
				sc.Capture.Ts.Local().Format("2006-01-02 15:04"),
				sc.ActiveWindow.Class,
				sc.ActiveWindow.Title,
				sc.FilePath,
			)
		}
		return
	}

	if useTui {
		if err := tui.RenderTable(store); err != nil {
			fmt.Fprintf(os.Stderr, "Error running TUI: %v\n", err)
			os.Exit(1)
		}
		return
	}

	screenshots, err := store.ListScreenshots(50, filterField, filterValue)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error listing screenshots: %v\n", err)
		os.Exit(1)
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	fmt.Fprintln(w, "ID\tTIME\tAPP\tTITLE\tFILE")
	for _, sc := range screenshots {
		fmt.Fprintf(w, "%d\t%s\t%s\t%.30s\t%s\n",
			sc.ID,
			sc.Capture.Ts.Local().Format("2006-01-02 15:04"),
			sc.ActiveWindow.Class,
			sc.ActiveWindow.Title,
			filepath.Base(sc.FilePath),
		)
	}
	w.Flush()
}
