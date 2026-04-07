package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

const tarragonManifestTOML = `name = "orego"
description = "Search and open OreGo screenshots"
enabled = true
entrypoint = "orego"
lifecycle_mode = "on_call"
provides_general_suggestions = false
prefix = "orego "
build_dependencies = []
capabilities = ["suggest", "screenshot"]
`

var tarragonCmd = &cobra.Command{
	Use:   "tarragon",
	Short: "Tarragon integration helpers",
}

var tarragonManifestCmd = &cobra.Command{
	Use:   "manifest",
	Short: "Print the Tarragon plugin manifest",
	Long:  "Print a stable Tarragon plugin manifest in TOML format to stdout.",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Fprint(cmd.OutOrStdout(), tarragonManifestTOML)
	},
}

var tarragonQueryCmd = &cobra.Command{
	Use:   "query <text>",
	Short: "Run one-shot Tarragon query mode",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runTarragonOnce(cmd, strings.Join(args, " "))
	},
}

var tarragonSelectCmd = &cobra.Command{
	Use:   "select <result-id> [action]",
	Short: "Run one-shot Tarragon action mode",
	Args:  cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		action := ""
		if len(args) > 1 {
			action = args[1]
		}
		return runTarragonSelect(cmd, args[0], action)
	},
}

func init() {
	tarragonCmd.AddCommand(tarragonManifestCmd)
	tarragonCmd.AddCommand(tarragonQueryCmd)
	tarragonCmd.AddCommand(tarragonSelectCmd)
	rootCmd.AddCommand(tarragonCmd)
}
