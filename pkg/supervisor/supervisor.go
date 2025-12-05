package supervisor

import (
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"slices"
	"strings"
	"sync"
	"syscall"
	"time"

	"spm/pkg/config"
	"spm/pkg/logger"
	"spm/pkg/utils"

	"github.com/gnuos/daemon"
	"go.uber.org/zap"
)

var daemonCtx *daemon.Context
var maxCpus = runtime.NumCPU()

func GetDaemon() *daemon.Context {
	if daemonCtx == nil {
		daemonCtx = &daemon.Context{
			PidFileName: config.GetConfig().PidFile,
			PidFilePerm: 0644,
			WorkDir:     config.WorkDirFlag,
			Umask:       027,
			Args:        os.Args,
		}
	}

	return daemonCtx
}

type ProcTable struct {
	mu sync.RWMutex

	table map[string]*Process
}

func (pt *ProcTable) Get(name string) *Process {
	p, ok := pt.table[name]
	if ok {
		return p
	}

	return nil
}

func (pt *ProcTable) Add(name string, proc *Process) bool {
	pt.mu.RLock()
	defer pt.mu.RUnlock()

	if _, ok := pt.table[name]; ok {
		return false
	}

	pt.table[name] = proc

	return true
}

func (pt *ProcTable) Del(name string) bool {
	pt.mu.RLock()
	defer pt.mu.RUnlock()

	p, ok := pt.table[name]
	if !ok {
		return false
	}

	_ = p.logger.Sync()

	delete(pt.table, name)

	return true
}

func (pt *ProcTable) Iter() map[string]*Process {
	pt.mu.Lock()
	defer pt.mu.Unlock()

	return pt.table
}

// Supervisor 是管理维护进程组的核心
type Supervisor struct {
	AfterStart func()
	StartedAt  time.Time
	Pid        int

	mu           sync.RWMutex
	logger       *zap.SugaredLogger
	projectTable *ProjectTable
	procTable    *ProcTable
}

func NewSupervisor() *Supervisor {
	signal.Notify(utils.StopChan, os.Interrupt, syscall.SIGTERM, syscall.SIGQUIT)

	return &Supervisor{
		AfterStart: func() {},
		StartedAt:  time.Now(),
		Pid:        utils.SupervisorPid,
		logger:     logger.Logging("supervisor"),
		projectTable: &ProjectTable{
			table: make(map[string]*Project),
		},
		procTable: &ProcTable{
			table: make(map[string]*Process),
		},
	}
}

func (sv *Supervisor) Daemon() {
	defer func() {
		if config.ForegroundFlag {
			_ = os.Remove(config.GetConfig().PidFile)
		} else {
			_ = GetDaemon().Release()
		}
		_ = os.Remove(config.GetConfig().Socket)
	}()

	sv.StartedAt = time.Now()

	if config.ForegroundFlag {
		err := utils.WriteDaemonPid(utils.SupervisorPid)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			os.Exit(1)
		}
	} else {
		d, err := GetDaemon().Reborn()
		if err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			_ = GetDaemon().Release()
			os.Exit(1)
		}

		if d != nil {
			sv.Pid = d.Pid
			return
		}
	}

	fmt.Printf("\033[1;33;40mSpm supervisor started at %s\033[0m\n\n", sv.StartedAt.Format(time.RFC3339))

	go StartServer(sv)

	sv.logger.Infof("Spm supervisor PID %d", sv.Pid)

	if config.ForegroundFlag {
		go sv.AfterStart()
	}

	sig := <-utils.StopChan

	switch sig {
	case os.Interrupt, syscall.SIGTERM:
		utils.FinishChan <- struct{}{}
		sv.Shutdown()
	}
	close(utils.StopChan)

	sv.logger.Info("Supervisor daemon stopped")
}

func (sv *Supervisor) Shutdown() {
	_ = sv.StopAll("*")

	pt := sv.procTable.Iter()
	for _, p := range pt {
		_ = p.logger.Sync()
	}

	sv.logger.Info("Shutdown supervisor...")
}

func (sv *Supervisor) Reload(changed []*Process) []*ProcInfo {
	sv.logger.Info("Reloading configuration")
	config.SetConfig(utils.GlobalConfigFile)

	pInfo := make([]*ProcInfo, 0)

	if len(changed) > 0 {
		for _, p := range changed {
			pInfo = append(pInfo, &ProcInfo{
				Pid:     p.Pid,
				Name:    p.FullName,
				StartAt: p.StartAt.UnixMilli(),
				StopAt:  p.StopAt.UnixMilli(),
				Status:  p.State,
			})
		}
	}

	return pInfo
}

func (sv *Supervisor) Status(name string) *Process {
	sv.mu.Lock()
	defer sv.mu.Unlock()

	p := sv.procTable.Get(name)
	if p == nil {
		return nil
	}

	if p.IsRunning() {
		appName := strings.Split(name, "::")[0]
		proj := sv.projectTable.Get(appName)
		proj.SetState(p.Name, true)
	}

	return p
}

func (sv *Supervisor) StatusAll(appName string) (procs []*Process) {
	procs = make([]*Process, 0)
	var proj *Project
	if appName != "*" {
		proj = sv.projectTable.Get(appName)
		if proj == nil {
			return
		}

		plist := proj.GetProcNames()
		for _, name := range plist {
			fullName := fmt.Sprintf("%s::%s", appName, name)
			p := sv.Status(fullName)
			if p != nil {
				procs = append(procs, p)
			}
		}
	} else {
		pt := sv.procTable.Iter()
		for name := range pt {
			p := sv.Status(name)
			procs = append(procs, p)
		}
	}

	return
}

func (sv *Supervisor) Start(name string) *Process {
	sv.mu.Lock()
	defer sv.mu.Unlock()

	p := sv.procTable.Get(name)
	if p == nil {
		return nil
	}

	appName := strings.Split(name, "::")[0]
	proj := sv.projectTable.Get(appName)

	if p.IsRunning() {
		p.logger.Warnf("%s already running with PID %d", p.FullName, p.Pid)

		proj.SetState(p.Name, true)
		return p
	}

	state := p.Start()
	proj.SetState(p.Name, state)

	if state {
		return p
	} else {
		return nil
	}
}

func (sv *Supervisor) StartAll(appName string) (procs []*Process) {
	procs = make([]*Process, 0)

	var proj *Project
	if appName != "*" {
		proj = sv.projectTable.Get(appName)
		if proj == nil {
			return
		}

		plist := proj.GetProcNames()
		for _, name := range plist {
			fullName := fmt.Sprintf("%s::%s", appName, name)
			p := sv.Start(fullName)
			if p != nil {
				procs = append(procs, p)
			}
		}
	} else {
		pt := sv.procTable.Iter()
		for name := range pt {
			p := sv.Start(name)
			procs = append(procs, p)
		}
	}

	return
}

func (sv *Supervisor) Stop(name string) *Process {
	sv.mu.Lock()
	defer sv.mu.Unlock()

	p := sv.procTable.Get(name)
	if p == nil {
		return nil
	}

	appName := strings.Split(name, "::")[0]
	proj := sv.projectTable.Get(appName)

	if p.State == processStopped {
		p.logger.Infof("%s is stopped.", p.FullName)
		proj.SetState(p.Name, false)
		return p
	}

	if proj.GetState(p.Name) {
		if p.Stop() {
			proj.SetState(p.Name, false)
			return p
		}
	}

	return nil
}

func (sv *Supervisor) StopAll(appName string) (procs []*Process) {
	procs = make([]*Process, 0)

	var proj *Project
	if appName != "*" {
		proj = sv.projectTable.Get(appName)
		if proj == nil {
			return
		}

		plist := proj.GetProcNames()
		for _, name := range plist {
			if proj.GetState(name) {
				fullName := fmt.Sprintf("%s::%s", appName, name)
				p := sv.Stop(fullName)
				if p != nil {
					procs = append(procs, p)
				}
			}
		}
	} else {
		pt := sv.procTable.Iter()
		for name := range pt {
			p := sv.Stop(name)
			procs = append(procs, p)
		}
	}

	return
}

func (sv *Supervisor) Restart(name string) *Process {
	sv.Stop(name)
	return sv.Start(name)
}

func (sv *Supervisor) RestartAll(appName string) []*Process {
	sv.StopAll(appName)
	return sv.StartAll(appName)
}

func (sv *Supervisor) UpdateApp(
	force bool,
	procOpts *ProcfileOption,
) (*Project, []*Process) {
	sv.mu.Lock()
	defer sv.mu.Unlock()

	newProj := CreateProject(procOpts)
	oldProj := sv.projectTable.Get(procOpts.AppName)
	if force {
		if oldProj == nil {
			if len(procOpts.Processes) == 0 || procOpts.WorkDir == "" {
				return nil, nil
			}

			_ = sv.projectTable.Set(procOpts.AppName, newProj)

			for name, opt := range procOpts.Processes {
				fullName := fmt.Sprintf("%s::%s", procOpts.AppName, name)
				proc := NewProcess(fullName, opt)
				proc.SetPidPath()

				sv.procTable.Add(fullName, proc)
				newProj.SetState(name, false)
			}

			return newProj, nil
		}
	} else {
		if oldProj != nil {
			// 记录新增的进程信息
			pList := make([]*Process, 0)
			oldProcList := oldProj.GetProcNames()
			newProcList := newProj.GetProcNames()

			for _, name := range oldProcList {
				if !newProj.IsExist(name) && !oldProj.GetState(name) {
					fullName := fmt.Sprintf("%s::%s", oldProj.Name, name)
					_ = sv.procTable.Del(fullName)
				}
			}

			for _, name := range newProcList {
				fullName := fmt.Sprintf("%s::%s", newProj.Name, name)
				if exist := sv.procTable.Get(fullName); exist != nil {
					continue
				}

				opt := procOpts.Processes[name]
				proc := NewProcess(fullName, opt)
				proc.SetPidPath()

				sv.procTable.Add(fullName, proc)
				oldProj.SetState(name, false)

				pList = append(pList, proc)
			}

			return oldProj, pList
		}
	}

	return oldProj, nil
}

// 参数toDo是一个占位符
// 0x0表示将要停止进程
// 0x1表示将要启动进程
// 0x2表示将要重启进程
// 0x3表示查看进程状态
func (sv *Supervisor) BatchDo(toDo ActionCtl, opt *ProcfileOption, procs []string) []*ProcInfo {
	var doFn func(string) *Process
	var doMany func(string) []*Process

	proj, _ := sv.UpdateApp(true, opt)
	if proj == nil {
		sv.logger.Errorf("Cannot find project in work directory %s", opt.WorkDir)
		return nil
	}

	switch toDo {
	case ActionStop:
		doFn = sv.Stop
		doMany = sv.StopAll
	case ActionStart:
		doFn = sv.Start
		doMany = sv.StartAll
	case ActionRestart:
		doFn = sv.Restart
		doMany = sv.RestartAll
	case ActionStatus:
		doFn = sv.Status
		doMany = sv.StatusAll
	}

	var pInfo = make([]*ProcInfo, 0)
	if slices.Contains(procs, "*") {
		completed := doMany("*")

		for _, p := range completed {
			pInfo = append(pInfo, &ProcInfo{
				Pid:     p.Pid,
				Name:    p.FullName,
				StartAt: p.StartAt.UnixMilli(),
				StopAt:  p.StartAt.UnixMilli(),
				Status:  p.State,
			})
		}
	} else {
		for _, name := range procs {
			p := doFn(name)
			if p != nil {
				pInfo = append(pInfo, &ProcInfo{
					Pid:     p.Pid,
					Name:    p.FullName,
					StartAt: p.StartAt.UnixMilli(),
					StopAt:  p.StartAt.UnixMilli(),
					Status:  p.State,
				})
			}
		}
	}

	return pInfo
}
