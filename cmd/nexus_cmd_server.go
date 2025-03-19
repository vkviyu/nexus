package cmd

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/spf13/viper"
)

// CmdProgram 表示业务程序
type CmdProgram func(stopctx context.Context, env map[string]interface{}, cleanupDone chan error)

// programStopContext 保存程序停止相关的上下文和取消方法
type programStopContext struct {
	stopctx context.Context
	cancel  context.CancelFunc
}

// ServerStopResponse 用于 /control/stop 接口，表示停止程序和关闭 HTTP 控制服务器
type ServerStopResponse struct {
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}

// ProgramRestartResponse 用于 /control/restart 接口，仅重启业务程序
type ProgramRestartResponse struct {
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}

// ServerStatusResponse 用于 /control/status 接口
type ServerStatusResponse struct {
	Status      string                 `json:"status"`
	CtrlHost    string                 `json:"ctrlhost"`
	CtrlPort    string                 `json:"ctrlport"`
	CtrlTimeout int                    `json:"ctrltimeout"`
	Config      string                 `json:"config"`
	Pid         int                    `json:"pid"`
	Env         map[string]interface{} `json:"environment"`
}

// nexusCmdServer 中既包含业务程序，也包含 HTTP 控制服务器，用于启动和停止服务
type nexusCmdServer struct {
	ctrlhost           string
	ctrlport           string
	ctrltimeout        int // in seconds
	program            CmdProgram
	programStopContext programStopContext
	env                map[string]interface{}
	cleanupDone        chan error
}

func newNexusCmdServer(ctrlHost, ctrlPort string, ctrltimeout int, program CmdProgram, env map[string]interface{}) *nexusCmdServer {
	programStopCtx, cancel := context.WithCancel(context.Background())
	cleanupDone := make(chan error)
	return &nexusCmdServer{
		ctrlhost:    ctrlHost,
		ctrlport:    ctrlPort,
		ctrltimeout: ctrltimeout,
		program:     program,
		programStopContext: programStopContext{
			stopctx: programStopCtx,
			cancel:  cancel,
		},
		env:         env,
		cleanupDone: cleanupDone,
	}
}

func (n *nexusCmdServer) start() {
	// 启动业务程序
	go n.program(n.programStopContext.stopctx, n.env, n.cleanupDone)
	mux := http.NewServeMux()
	server := &http.Server{
		Addr:    n.ctrlhost + ":" + n.ctrlport,
		Handler: mux,
	}

	// /control/stop 接口用于停止程序和关闭控制服务器
	mux.HandleFunc("/control/stop", func(w http.ResponseWriter, r *http.Request) {
		n.programStopContext.cancel()
		select {
		case <-n.cleanupDone:
			ctx, cancel := context.WithTimeout(context.Background(), time.Duration(n.ctrltimeout)*time.Second)
			defer cancel()
			if err := server.Shutdown(ctx); err != nil {
				resp := ServerStopResponse{
					Success: false,
					Error:   "Program stopped but server did not shut down: " + err.Error(),
				}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
				json.NewEncoder(w).Encode(resp)
				return
			}
			resp := ServerStopResponse{Success: true}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
		case <-time.After(time.Duration(n.ctrltimeout) * time.Second):
			resp := ServerStopResponse{
				Success: false,
				Error:   "Program stop timeout",
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(resp)
		}
	})

	// /control/restart 接口仅用于重启业务程序，不重启控制服务器
	mux.HandleFunc("/control/restart", func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			resp := ProgramRestartResponse{
				Success: false,
				Error:   err.Error(),
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(resp)
			return
		}
		env := make(map[string]interface{})
		err = json.Unmarshal(body, &env)
		if err != nil {
			resp := ProgramRestartResponse{
				Success: false,
				Error:   err.Error(),
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(resp)
			return
		}
		// 将未传入的新环境变量补充上原始值
		for k, v := range n.env {
			if _, exists := env[k]; !exists {
				env[k] = v
			}
		}
		n.programStopContext.cancel()
		select {
		case <-n.cleanupDone:
			// 重置上下文和程序状态，重启业务程序
			n.programStopContext.stopctx, n.programStopContext.cancel = context.WithCancel(context.Background())
			n.env = env
			go n.program(n.programStopContext.stopctx, n.env, n.cleanupDone)
			resp := ProgramRestartResponse{Success: true}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
		case <-time.After(time.Duration(n.ctrltimeout) * time.Second):
			resp := ProgramRestartResponse{
				Success: false,
				Error:   "Program stop timeout",
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(resp)
		}
	})

	// /control/status 接口返回当前服务器状态
	mux.HandleFunc("/control/status", func(w http.ResponseWriter, r *http.Request) {
		configFile := viper.ConfigFileUsed()
		pid := viper.GetInt("nexus.pid")
		resp := ServerStatusResponse{
			Status:      "running",
			CtrlHost:    n.ctrlhost,
			CtrlPort:    n.ctrlport,
			CtrlTimeout: n.ctrltimeout,
			Config:      configFile,
			Pid:         pid,
			Env:         n.env,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})

	server.ListenAndServe()
}
