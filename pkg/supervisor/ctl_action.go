package supervisor

type ActionCtl int

const (
	ActionRun ActionCtl = iota
	ActionLog
	ActionKill
	ActionStart
	ActionStop
	ActionStatus
	ActionRestart
	ActionShutdown
	ActionReload
)

var actionResponse = map[ActionCtl]string{
	ActionRun:     "Run command successfully",
	ActionStart:   "Start processes successfully",
	ActionStop:    "Stop processes successfully",
	ActionStatus:  "Check processes stattus successfully",
	ActionRestart: "Restart processes successfully",
}

type ActionMsg struct {
	Action    ActionCtl `msgpack:"action" json:"action"`
	WorkDir   string    `msgpack:"workdir" json:"workdir"`
	Procfile  string    `msgpack:"procfile" json:"procfile"`
	Projects  string    `msgpack:"projects" json:"projects"`
	Processes string    `msgpack:"processes" json:"processes"`
	CmdLine   []string  `msgpack:"cmdline" json:"cmdline"`
}
