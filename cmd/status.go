package cmd

import (
	"fmt"
	"log"
	"strings"

	"github.com/spf13/cobra"

	"spm/pkg/supervisor"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Check processed status",
	Run:   execStatusCmd,
}

func init() {
	statusCmd.PersistentPreRun = func(cmd *cobra.Command, args []string) {
		rootCmd.PersistentPreRun(cmd, args)
		execStatusPersistentPreRun()
	}

	rootCmd.AddCommand(statusCmd)
}

func execStatusPersistentPreRun() {
	if !isDaemonRunning() {
		log.Fatalln("ERROR: Supervisor has not started. Please check supervisor daemon.")
	}
}

func execStatusCmd(cmd *cobra.Command, args []string) {
	var procs string

	if len(args) == 0 {
		procs = "*"
	} else if len(args) == 1 {
		procs = args[0]
	} else {
		procs = strings.Join(args, "|")
	}

	msg.Action = supervisor.ActionStatus
	msg.Processes = procs

	res := supervisor.ClientRun(msg)
	if res == nil {
		fmt.Println("No processes found.")
		return
	}

	for _, proc := range res {
		fmt.Printf("%s\t\t%s\t\tPID: %d\n", proc.Name, proc.Status, proc.Pid)
	}
}
