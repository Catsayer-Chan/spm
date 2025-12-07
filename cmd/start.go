package cmd

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"spm/pkg/config"
	"spm/pkg/supervisor"
)

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Starts processes and/or the supervisor",
	Run:   execStartCmd,
}

func init() {
	startCmd.PersistentFlags().BoolVarP(&config.ForegroundFlag, "foregroud", "f", false, "Run the supervisor in the foreground")

	startCmd.PersistentPreRun = func(cmd *cobra.Command, args []string) {
		rootCmd.PersistentPreRun(cmd, args)
		execStartPersistentPreRun()
	}

	rootCmd.AddCommand(startCmd)
}

func execStartPersistentPreRun() {
	if !config.ForegroundFlag {
		if isDaemonRunning() {
			return
		}

		if err := tryRunDaemon(); err != nil {
			log.Fatal(err)
		}

		time.Sleep(1 * time.Second)
	}
}

func execStartCmd(cmd *cobra.Command, args []string) {
	sendStartCmd := func(args []string) {
		var procs string

		if len(args) == 0 {
			procs = "*"
		} else if len(args) == 1 {
			procs = args[0]
		} else {
			procs = strings.Join(args, ";")
		}

		msg.Action = supervisor.ActionStart
		msg.Processes = procs

		res := supervisor.ClientRun(msg)
		if res == nil {
			fmt.Println("No processes to start.")
			return
		}

		for _, proc := range res {
			fmt.Printf("%s %s\t[PID %d] %s\n", time.UnixMilli(proc.StartAt).Format(time.RFC3339), proc.Name, proc.Pid, proc.Status)
		}
	}

	if config.ForegroundFlag && !isDaemonRunning() {
		sv := supervisor.NewSupervisor()

		opt, err := supervisor.LoadProcfileOption(config.WorkDirFlag, config.ProcfileFlag)
		if err != nil {
			log.Fatal(err)
		}

		// 注册当前项目中的进程表
		proj, _ := sv.UpdateApp(true, opt)
		if proj == nil {
			log.Fatalf("Cannot find project in work directory %s", config.WorkDirFlag)
		}

		// 设置前台启动进程的回调方法
		sv.AfterStart = func() {
			sendStartCmd(args)
		}
		sv.Daemon()
	} else {
		sendStartCmd(args)
	}
}
