package supervisor

type ResponseCtl int

const (
	ResponseNormal ResponseCtl = iota
	ResponseShutdown
	ResponseReload
	ResponseMsgErr
)

type ProcInfo struct {
	Pid     int          `msgpack:"pid"`
	Name    string       `msgpack:"name"`
	StartAt int64        `msgpack:"start_at"`
	StopAt  int64        `msgpack:"stop_at"`
	Status  ProcessState `msgpack:"status"`
}

type ResponseMsg struct {
	Code      int         `msgpack:"code"`
	Message   string      `msgpack:"message"`
	Processes []*ProcInfo `msgpack:"processes"`
}
