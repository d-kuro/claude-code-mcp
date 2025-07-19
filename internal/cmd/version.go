package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/d-kuro/claude-code-mcp/pkg/version"
)

// NewVersionCmd creates a new version command
func NewVersionCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		Long:  `Print the version information of claude-code-mcp including git commit, build date, and Go version.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			jsonFlag, _ := cmd.Flags().GetBool("json")
			v := version.GetVersion()

			if jsonFlag {
				encoder := json.NewEncoder(os.Stdout)
				encoder.SetIndent("", "  ")
				if err := encoder.Encode(v); err != nil {
					return fmt.Errorf("error encoding version info: %w", err)
				}
			} else {
				fmt.Println(v.String())
			}
			return nil
		},
	}

	cmd.Flags().BoolP("json", "j", false, "Output version information as JSON")
	return cmd
}
