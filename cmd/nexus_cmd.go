package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// 默认配置
var (
	DefaultCtrlHost    = "127.0.0.1"
	DefaultCtrlPort    = "8090"
	DefaultCtrlTimeout = 5 // seconds
	DefaultConfigFile  = "nexus.yaml"
)

type GlobalProgramContext struct {
	GlobalProgram         CmdProgram
	GlobalProgramStopFunc context.CancelFunc
	GlobalStopFunc        context.CancelFunc
	GlobalCleanupDone     chan error
}

// ProgramConfig 保存业务启动时的各项配置
type ProgramConfig struct {
	MySQLDSN string
	BBoltDB  string
	Host     string
	Port     string
	Mode     string
}

type NexusCmd struct {
	cmd *cobra.Command
}

// setNestedValue 将键 key（用点号分隔）对应的 value 设置到 m 中
func setNestedValue(m map[string]interface{}, key string, value interface{}) {
	parts := strings.Split(key, ".")
	last := len(parts) - 1
	current := m
	for i, part := range parts {
		if i == last {
			current[part] = value
		} else {
			if next, exists := current[part]; exists {
				if nextMap, ok := next.(map[string]interface{}); ok {
					current = nextMap
				} else {
					nextMap := make(map[string]interface{})
					current[part] = nextMap
					current = nextMap
				}
			} else {
				nextMap := make(map[string]interface{})
				current[part] = nextMap
				current = nextMap
			}
		}
	}
}

// NewNexusCmd 创建基于 cobra 的命令行接口，其中包括 start、stop、restart、status 以及 install 子命令
func NewNexusCmd(program CmdProgram) *NexusCmd {
	cmd := &cobra.Command{
		Use:   "nexus",
		Short: "Nexus server",
	}

	// 直接在命令创建后定义 PersistentFlags
	pflags := cmd.PersistentFlags()
	pflags.StringP("config", "c", "", "config file")
	pflags.String("ctrlport", DefaultCtrlPort, "control port")
	pflags.String("ctrlhost", DefaultCtrlHost, "control host")
	pflags.Int("ctrltimeout", DefaultCtrlTimeout, "control timeout")
	pflags.StringArrayP("env", "e", nil, "Override config items, format KEY=VALUE (can be set multiple times)")

	// 接着设置 PersistentPreRun 仅做绑定和配置加载
	cmd.PersistentPreRun = func(cmd *cobra.Command, args []string) {
		viper.BindPFlag("nexus.ctrlhost", pflags.Lookup("ctrlhost"))
		viper.BindPFlag("nexus.ctrlport", pflags.Lookup("ctrlport"))
		viper.BindPFlag("nexus.ctrltimeout", pflags.Lookup("ctrltimeout"))

		configFile := cmd.Flag("config").Value.String()
		if configFile == "" {
			fmt.Printf("No configuration file specified; attempting to use default configuration file \"%s\".\n", DefaultConfigFile)
			configFile = DefaultConfigFile
			if _, err := os.Stat(configFile); os.IsNotExist(err) {
				fmt.Printf("Default config file %s not found, program may not work as expected.\n", configFile)
				configFile = ""
			}
		}
		if configFile != "" {
			if _, err := os.Stat(configFile); os.IsNotExist(err) {
				fmt.Printf("Config file %s not found, program is exiting.\n", configFile)
				os.Exit(1)
			}
			fmt.Printf("Using config file: %s\n", configFile)
			viper.SetConfigFile(configFile)
			if err := viper.ReadInConfig(); err == nil {
				fmt.Printf("Reading config file: %s successfully.\n", viper.ConfigFileUsed())
			} else {
				fmt.Printf("Error reading config file: %v\n", err)
			}
		}
		pid := os.Getpid()
		viper.Set("nexus.pid", pid)
		viper.Set("nexus.config", configFile)

		envItems, _ := cmd.Flags().GetStringArray("env")
		if len(envItems) > 0 {
			// 先取原有的 nexus.environment，如果返回 nil 则新建一个 map
			origEnv := viper.Get("nexus.environment")
			var origConfig map[string]interface{}
			if origEnv == nil {
				origConfig = make(map[string]interface{})
			} else {
				// 尝试断言为 map
				if m, ok := origEnv.(map[string]interface{}); ok {
					origConfig = m
				} else {
					origConfig = make(map[string]interface{})
				}
			}
			// 对每个 -e 参数进行处理，支持多层级键，如 database.mysql=xxx
			for _, item := range envItems {
				parts := strings.SplitN(item, "=", 2)
				if len(parts) != 2 {
					fmt.Printf("Invalid env override format: %s, expected KEY=VALUE\n", item)
					continue
				}
				key := strings.TrimSpace(parts[0])
				value := strings.TrimSpace(parts[1])
				// 使用辅助函数支持多层级键（例如 "database.mysql"）
				setNestedValue(origConfig, key, value)
			}
			// 将更新后的 map 写回 viper
			viper.Set("nexus.environment", origConfig)
		}
	}

	nexusCmd := &NexusCmd{
		cmd: cmd,
	}

	startCmd := &cobra.Command{
		Use:   "start",
		Short: "Start the program and control server",
		Run: func(cmd *cobra.Command, args []string) {
			ctrlhost := viper.GetString("nexus.ctrlhost")
			ctrlport := viper.GetString("nexus.ctrlport")
			ctrltimeout := viper.GetInt("nexus.ctrltimeout")
			ncs := newNexusCmdServer(ctrlhost, ctrlport, ctrltimeout, program, viper.GetStringMap("nexus.environment"))
			ncs.start()
		},
	}
	stopCmd := &cobra.Command{
		Use:   "stop",
		Short: "Stop the program and control server",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("Stopping the program and control server...")
			sendControlCommand("stop")
		},
	}
	restartCmd := &cobra.Command{
		Use:   "restart",
		Short: "Restart the program (control server remains running)",
		Run: func(cmd *cobra.Command, args []string) {
			if err := sendControlCommand("restart"); err != nil {
				fmt.Printf("Error restarting program: %v\n", err)
			}
		},
	}
	statusCmd := &cobra.Command{
		Use:   "status",
		Short: "Get the program and control server status",
		Run: func(cmd *cobra.Command, args []string) {
			if err := sendControlCommand("status"); err != nil {
				fmt.Printf("Error getting status: %v\n", err)
			}
		},
	}

	nexusCmd.cmd.AddCommand(startCmd, stopCmd, restartCmd, statusCmd)
	return nexusCmd
}

func (n *NexusCmd) Execute() error {
	return n.cmd.Execute()
}

// sendControlCommand 作为客户端连接 ctrlport 并发送指定命令
func sendControlCommand(command string) error {
	ctrlhost := viper.GetString("nexus.ctrlhost")
	ctrlport := viper.GetString("nexus.ctrlport")
	ctrltimeout := viper.GetInt("nexus.ctrltimeout")
	url := fmt.Sprintf("http://%s:%s/control/%s", ctrlhost, ctrlport, command)
	client := http.Client{Timeout: time.Duration(ctrltimeout) * time.Second}
	var resp *http.Response
	var err error
	if command == "restart" {
		// restart 命令需要发送 JSON 格式的环境变量
		env := viper.GetStringMap("nexus.environment")
		var envData []byte
		envData, err = json.Marshal(env)
		if err != nil {
			return fmt.Errorf("marshal env error: %v", err)
		}
		resp, err = client.Post(url, "application/json", bytes.NewBuffer(envData))
	} else {
		resp, err = client.Get(url)
	}
	if err != nil {
		return fmt.Errorf("cannot connect to control server: %v", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	fmt.Println(string(body))
	return nil
}
