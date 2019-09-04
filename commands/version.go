package commands

import (
	"fmt"
	"github.com/spf13/cobra"
	"io"
)

func NewVersion(version string, output io.Writer) *cobra.Command {
	return &cobra.Command{
		Use:        "version",
		Aliases:    []string{"v"},
		Short:      "prints the om release version",
		Long:       "This command prints the om release version number.",
		Example:    "om version",
		RunE: func(cmd *cobra.Command, args []string) error {
			_, err := output.Write([]byte(version + "\n"))
			if err != nil {
				return fmt.Errorf("could not print version: %s", err)
			}
			return nil
		},
	}
}
