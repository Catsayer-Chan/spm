// Package cmd
package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

var Version string

// versionCmd represents the version command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version and exit",
	Long:  `Print version and exit`,
	Run:   execVersionCmd,
}

func init() {
	rootCmd.AddCommand(versionCmd)
}

func execVersionCmd(cmd *cobra.Command, _ []string) {
	fmt.Printf("%s v%s\n", filepath.Base(os.Args[0]), Version)
}
