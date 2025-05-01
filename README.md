# Nexus

Nexus 是一个 Go 语言编写的轻量级服务框架，提供了命令行控制、HTTP/WebSocket 服务、数据库操作等常用功能，帮助开发者快速构建微服务应用。

## 特性

- 🚀 命令行控制：支持 start/stop/restart/status 等命令
- 🌐 HTTP/WebSocket 服务：内置 HTTP 服务器和 WebSocket 支持
- 💾 数据库支持：支持 MySQL (GORM)、BBoltDB、BadgerDB 等数据库
- 📝 日志管理：支持滚动日志记录
- 🛠 工具集：提供 JSON 处理、HTTP 响应等实用工具

## 安装

```bash
go get github.com/vkviyu/nexus
```

## 快速开始

### 1. 创建主程序

```go
package main

import (
    "context"
    "fmt"
    "net/http"

    "github.com/vkviyu/nexus/cmd"
    "github.com/vkviyu/nexus/database/gormdb"
    "github.com/vkviyu/nexus/transport/server/handler"
    "github.com/vkviyu/nexus/utils/logutil"
)

func program(stopctx context.Context, env map[string]interface{}, cleanUpDone chan error) {
    // 从环境变量中获取配置
    mysqldsn := env["mysql"].(string)
    host := env["host"].(string)
    port := env["port"].(string)

    // 创建数据库连接
    db, err := gormdb.OpenWithDSN(mysqldsn, nil)
    if err != nil {
        panic(err)
    }

    // 创建处理器映射
    handlerMap := handler.HandlerMap{
        "/api/hello": http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            fmt.Fprintf(w, "Hello, Nexus!")
        }),
    }

    // 创建 HTTP 服务器
    mux := handler.GetServeMux(handlerMap)
    server := &http.Server{
        Addr:    host + ":" + port,
        Handler: mux,
    }

    // 启动服务器
    go server.ListenAndServe()

    // 等待停止信号
    <-stopctx.Done()
    server.Shutdown(context.Background())
    cleanUpDone <- nil
}

func main() {
    // 创建日志记录器
    logger, err := logutil.NewRotateLogger(nil)
    if err != nil {
        panic(err)
    }

    // 创建并执行命令
    serverCmd := cmd.NewNexusCmd(program)
    if err := serverCmd.Execute(); err != nil {
        logger.Errorf("serverCmd.Execute() error: %v", err)
        panic(err)
    }
}
```

### 2. 运行程序

```bash
# 启动服务
nexus start -e mysql="user:pass@tcp(localhost:3306)/dbname"

# 查看状态
nexus status

# 重启服务（支持所有配置选项，包括配置文件和环境变量）
nexus restart -c custom.yaml -e mysql="new-dsn" -e host="new-host"

# 停止服务
nexus stop
```

## 配置说明

### 命令行参数

- `--config` 或 `-c`: 配置文件路径
- `--ctrlport`: 控制端口 (默认: 8090)
- `--ctrlhost`: 控制主机 (默认: 127.0.0.1)
- `--ctrltimeout`: 控制超时时间 (默认: 5 秒)
- `--env` 或 `-e`: 环境变量覆盖，格式为 KEY=VALUE (可多次使用)

### 配置文件

Nexus 默认使用 `nexus.yaml` 作为配置文件，支持 YAML 格式。配置文件分为两个主要部分：

1. **框架配置**：以 `nexus` 为根节点的配置，用于控制框架本身的行为
2. **应用配置**：以 `environment` 为节点的配置，用于存储用户自定义的应用配置

#### 框架配置示例

```yaml
nexus:
  ctrlhost: "127.0.0.1" # 控制服务器主机
  ctrlport: "8090" # 控制服务器端口
  ctrltimeout: 5 # 控制超时时间（秒）
```

#### 应用配置示例

**所有在 `environment` 节点下的配置都是完全自定义的**，您可以自由定义任何配置项。以下是一个示例：

```yaml
nexus:
  environment:
    # 数据库配置
    mysql: "user:pass@tcp(localhost:3306)/dbname?charset=utf8mb4&parseTime=True&loc=Local"
    bbolt: "data.db"

    # 服务器配置
    host: "0.0.0.0"
    port: "5000"
    mode: "dev" # 或 "prod"


    # 添加更多自定义的配置项......
```

#### 配置覆盖

您可以通过命令行参数覆盖配置文件中的任何配置项。配置的优先级从高到低为：

1. 命令行 `-e` 参数
2. 配置文件中的配置
3. 默认配置

覆盖配置的示例：

```bash
# 覆盖单个配置项
nexus start -e mysql="new-user:new-pass@tcp(new-host:3306)/new-db"

# 覆盖多个配置项
nexus start -e mysql="new-dsn" -e host="new-host" -e port="8080"

# 覆盖嵌套配置项（使用点号分隔）
nexus start -e "database.mysql.host=localhost" -e "database.mysql.port=3306"

# restart 命令同样支持所有配置覆盖选项
nexus restart -c custom.yaml -e mysql="new-dsn" -e host="new-host"
```

注意：

- **所有在 `environment` 节点下的配置都是完全自定义的**，您可以：
  - 添加任意名称的配置项
  - 使用任意层级的配置结构
  - 使用字符串、数字、布尔值、数组、对象等所有 YAML 支持的数据类型
  - 根据实际需求修改或删除任何配置项
- 框架不会对配置项做任何限制，您可以根据应用需求自由组织配置结构
- 配置项的结构和命名完全由您决定，上面的示例仅供参考

## 模块说明

### cmd

提供命令行控制功能，包括：

- 服务启动/停止/重启
- 状态查询
- 配置管理

### transport

提供网络传输相关功能：

- HTTP 服务器
- WebSocket 支持
- 客户端工具

### database

支持多种数据库：

- MySQL (通过 GORM)
- BBoltDB
- BadgerDB

### utils

提供实用工具：

- 日志管理
- JSON 处理
- HTTP 响应处理

## 示例

### WebSocket 服务

```go
// 创建 WebSocket 端点
endpoint := websocket.NewEndpoint("/ws", websocket.WithAuthFunc(
    func(r *http.Request) (authResult bool, id string) {
        // 实现认证逻辑
        return true, "user123"
    },
))

// 添加到处理器映射
handlerMap := handler.HandlerMap{
    "/ws": endpoint,
}
```

### 数据库操作

```go
// MySQL 操作
db, err := gormdb.OpenWithDSN("user:pass@tcp(localhost:3306)/dbname", nil)
if err != nil {
    panic(err)
}

// BBoltDB 操作
bboltDB, err := bboltdb.Open("data.db", 0600, nil)
if err != nil {
    panic(err)
}
```

## 许可证

MIT License
