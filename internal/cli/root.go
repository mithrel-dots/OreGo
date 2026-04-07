package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

var onceQuery string
var selectResultID string
var selectAction string

var rootCmd = &cobra.Command{
	Use:   "orego",
	Short: "Context-Aware Screenshot Tool",
	Long:  `orego captures screenshots with rich metadata (window class, title, workspace state) and stores them in a searchable SQLite database.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("select") || cmd.Flags().Changed("result-id") {
			if strings.TrimSpace(selectResultID) == "" {
				return fmt.Errorf("missing required --select value")
			}
			return runTarragonSelect(cmd, selectResultID, selectAction)
		}
		if cmd.Flags().Changed("once") {
			return runTarragonOnce(cmd, onceQuery)
		}
		return cmd.Help()
	},
}

func init() {
	rootCmd.PersistentFlags().StringVar(&onceQuery, "once", "", "Run one-shot Tarragon query mode")
	rootCmd.PersistentFlags().StringVar(&selectResultID, "select", "", "Run one-shot Tarragon selection mode")
	rootCmd.PersistentFlags().StringVar(&selectAction, "action", "", "Action name for Tarragon selection mode")
	rootCmd.PersistentFlags().StringVar(&selectResultID, "result-id", "", "Alias for --select")
	_ = rootCmd.PersistentFlags().MarkHidden("once")
	_ = rootCmd.PersistentFlags().MarkHidden("select")
	_ = rootCmd.PersistentFlags().MarkHidden("result-id")
	_ = rootCmd.PersistentFlags().MarkHidden("action")
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
