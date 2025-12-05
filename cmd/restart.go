package cmd

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"spm/pkg/supervisor"
)

var restartCmd = &cobra.Command{
	Use:   "restart",
	Short: "Restart processes",
	Run:   execRestartCmd,
}

func init() {
	stopCmd.PersistentPreRun = func(cmd *cobra.Command, args []string) {
		rootCmd.PersistentPreRun(cmd, args)
		execRestartPersistentPreRun()
	}

	rootCmd.AddCommand(restartCmd)
}

func execRestartPersistentPreRun() {
	if !isDaemonRunning() {
		log.Fatalln("ERROR: Supervisor has not started. Please check supervisor daemon.")
	}
}

func execRestartCmd(cmd *cobra.Command, args []string) {
	var procs string

	if len(args) == 0 {
		procs = "*"
	} else if len(args) == 1 {
		procs = args[0]
	} else {
		procs = strings.Join(args, "|")
	}

	msg.Action = supervisor.ActionRestart
	msg.Processes = procs

	res := supervisor.ClientRun(msg)
	if res == nil {
		fmt.Println("No processes to restart.")
		return
	}

	for _, proc := range res {
		fmt.Printf("[%s] Restarted %s\t[PID %d]\n", time.UnixMilli(proc.StartAt).Format(time.RFC3339), proc.Name, proc.Pid)
	}
}
