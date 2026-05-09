# Nexus Web Workstation 架构设计文档

## 1. 项目愿景

Nexus Web Workstation 是一个以 **Contract 为核心** 的全功能 Web 开发工作站，融合 HTTP 客户端、代理服务器、Mock 服务器、代码生成器于一体，提供从 API 设计、测试、调试到代码生成的完整工作流。

### 1.1 核心理念

```

Contract First, Code Follows

协议优先，代码随行

```

-**Contract 即真理**：所有请求/响应都以 Contract 为标准

-**双向验证**：客户端和服务器都遵循同一 Contract 定义

-**协议即文档**：.contract.json 既是配置也是文档

-**文档即代码**：从 Contract 自动生成类型安全的代码

### 1.2 目标用户

- 后端开发者：API 开发、测试、Mock
- 前端开发者：无后端依赖的并行开发
- 测试工程师：接口测试、契约验证
- 架构师：API 设计、协议管理

---

## 2. 核心概念

### 2.1 Contract

Contract 是系统的核心抽象，定义了一个 API 的完整契约：

```

Contract = 请求定义 + 响应定义 + 类型约束 + 元数据

```

```json

{

"name": "CreateUser",

"request": {

"meta": { "method": "POST", "path": "/users" },

"example": { "body": { "name": "test" } },

"structs": { "CreateUserBody": "body" }

  },

"response": {

"example": { "id": 1, "name": "test" },

"structs": { "UserResult": "." }

  }

}

```

### 2.2 Server Domain（服务器域）

服务器域是一组相关服务器的逻辑集合：

```

ServerDomain

├── ProductionServer (https://api.example.com)

├── StagingServer (https://staging.example.com)

├── LocalMockServer (:8080)

└── LocalProxyServer (:8081)

```

### 2.3 Connection Mode（连接模式）

每个 Contract 可以配置不同的连接模式：

| 模式 | 描述 | 场景 |

|-----|------|------|

| Direct | 直连目标服务器 | 正常请求 |

| Proxy | 通过代理转发 | 流量记录 |

| Mock | 使用本地 Mock | 无后端测试 |

| Passthrough | 代理透传 | 生产环境调试 |

### 2.4 Link（链路）

Link 定义了 Contract 与 Server 之间的连接关系：

```

Link = Contract + Server + Mode + Options

```

---

## 3. 系统架构

### 3.1 整体架构图

```

┌─────────────────────────────────────────────────────────────────────┐

│                         Nexus Workstation                           │

├─────────────────────────────────────────────────────────────────────┤

│  ┌───────────────────────────────────────────────────────────────┐  │

│  │                      GUI Layer (Fyne)                         │  │

│  │  ┌─────────┐ ┌─────────┐ ┌─────────┐ ┌─────────┐ ┌─────────┐ │  │

│  │  │ Client  │ │ Server  │ │Contract │ │  Log    │ │ Config  │ │  │

│  │  │  View   │ │  View   │ │ Manager │ │ Viewer  │ │  Panel  │ │  │

│  │  └────┬────┘ └────┬────┘ └────┬────┘ └────┬────┘ └────┬────┘ │  │

│  └───────┼───────────┼───────────┼───────────┼───────────┼──────┘  │

│          │           │           │           │           │         │

│  ┌───────┴───────────┴───────────┴───────────┴───────────┴──────┐  │

│  │                    Core Service Layer                         │  │

│  │  ┌──────────────┐ ┌──────────────┐ ┌──────────────────────┐  │  │

│  │  │   Request    │ │    Server    │ │      Contract        │  │  │

│  │  │   Executor   │ │   Manager    │ │      Registry        │  │  │

│  │  └──────────────┘ └──────────────┘ └──────────────────────┘  │  │

│  │  ┌──────────────┐ ┌──────────────┐ ┌──────────────────────┐  │  │

│  │  │  WebSocket   │ │     Code     │ │      Traffic         │  │  │

│  │  │    Hub       │ │   Generator  │ │      Recorder        │  │  │

│  │  └──────────────┘ └──────────────┘ └──────────────────────┘  │  │

│  └──────────────────────────────────────────────────────────────┘  │

│                                                                     │

│  ┌──────────────────────────────────────────────────────────────┐  │

│  │                   Server Runtime Layer                        │  │

│  │  ┌──────────────┐ ┌──────────────┐ ┌──────────────────────┐  │  │

│  │  │    HTTP      │ │    Proxy     │ │     WebSocket        │  │  │

│  │  │   Server     │ │   Server     │ │      Server          │  │  │

│  │  │   (Mock)     │ │  (Record)    │ │     (Realtime)       │  │  │

│  │  └──────────────┘ └──────────────┘ └──────────────────────┘  │  │

│  └──────────────────────────────────────────────────────────────┘  │

│                                                                     │

│  ┌──────────────────────────────────────────────────────────────┐  │

│  │                    Storage Layer                              │  │

│  │  ┌──────────────┐ ┌──────────────┐ ┌──────────────────────┐  │  │

│  │  │   Contract   │ │   Traffic    │ │      Config          │  │  │

│  │  │    Store     │ │    Store     │ │      Store           │  │  │

│  │  │  (JSON/DB)   │ │  (SQLite)    │ │     (YAML)           │  │  │

│  │  └──────────────┘ └──────────────┘ └──────────────────────┘  │  │

│  └──────────────────────────────────────────────────────────────┘  │

└─────────────────────────────────────────────────────────────────────┘

```

### 3.2 数据流架构

```

                                  ┌─────────────┐

                                  │  Contract   │

                                  │   Store     │

                                  └──────┬──────┘

                                         │

          ┌──────────────────────────────┼──────────────────────────────┐

          │                              │                              │

          ▼                              ▼                              ▼

┌─────────────────┐           ┌─────────────────┐           ┌─────────────────┐

│  Client Mode    │           │   Proxy Mode    │           │   Mock Mode     │

│                 │           │                 │           │                 │

│  GUI ──► HTTP   │           │  GUI ──► Proxy  │           │  GUI ──► Mock   │

│         Client  │           │        Server   │           │        Server   │

└────────┬────────┘           └────────┬────────┘           └────────┬────────┘

         │                             │                             │

         │ Request                     │ Request                     │ Request

         ▼                             ▼                             ▼

┌─────────────────┐           ┌─────────────────┐           ┌─────────────────┐

│  Target Server  │           │  Target Server  │           │  Contract Data  │

│  (External)     │◄──────────│  (External)     │           │  (Local)        │

└────────┬────────┘           └────────┬────────┘           └────────┬────────┘

         │                             │                             │

         │ Response                    │ Response + Record           │ Response

         ▼                             ▼                             ▼

┌─────────────────────────────────────────────────────────────────────────────┐

│                           Response Handler                                   │

│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐  ┌─────────────────────┐ │

│  │  Validate   │  │   Record    │  │   Display   │  │  Generate Contract  │ │

│  │  Contract   │  │   Traffic   │  │   Result    │  │  (Optional)         │ │

│  └─────────────┘  └─────────────┘  └─────────────┘  └─────────────────────┘ │

└─────────────────────────────────────────────────────────────────────────────┘

         │

         ▼

┌─────────────────┐

│   WebSocket     │──────► Real-time Log Viewer

│   Broadcast     │

└─────────────────┘

```

---

## 4. 模块拆分

### 4.1 目录结构

```

gui/

├── main.go                     # GUI 入口

├── ARCHITECTURE.md             # 本文档

│

├── app/                        # 应用核心

│   ├── app.go                  # 应用实例

│   ├── state.go                # 全局状态管理

│   └── events.go               # 事件总线

│

├── views/                      # UI 视图层

│   ├── layout.go               # 主布局

│   ├── client/                 # 客户端视图

│   │   ├── client_view.go      # 客户端主视图

│   │   ├── request_bar.go      # 请求栏组件

│   │   ├── request_tabs.go     # 请求配置 Tabs

│   │   ├── response_panel.go   # 响应面板

│   │   └── kv_editor.go        # 键值对编辑器

│   │

│   ├── server/                 # 服务器视图

│   │   ├── server_view.go      # 服务器主视图

│   │   ├── server_list.go      # 服务器列表

│   │   ├── route_panel.go      # 路由配置面板

│   │   └── mock_editor.go      # Mock 规则编辑器

│   │

│   ├── contract/               # Contract 管理视图

│   │   ├── contract_view.go    # Contract 主视图

│   │   ├── contract_list.go    # Contract 列表

│   │   ├── contract_editor.go  # Contract 编辑器

│   │   └── link_config.go      # 连接配置面板

│   │

│   ├── log/                    # 日志视图

│   │   ├── log_view.go         # 日志主视图

│   │   ├── traffic_list.go     # 流量列表

│   │   └── detail_panel.go     # 详情面板

│   │

│   └── common/                 # 通用组件

│       ├── json_viewer.go      # JSON 查看器

│       ├── json_editor.go      # JSON 编辑器

│       ├── status_bar.go       # 状态栏

│       └── toolbar.go          # 工具栏

│

├── services/                   # 核心服务层

│   ├── executor/               # 请求执行器

│   │   ├── executor.go         # 执行器接口

│   │   ├── http_executor.go    # HTTP 执行

│   │   └── result.go           # 执行结果

│   │

│   ├── server/                 # 服务器管理

│   │   ├── manager.go          # 服务器管理器

│   │   ├── http_server.go      # HTTP 服务器

│   │   ├── proxy_server.go     # 代理服务器

│   │   └── websocket_server.go # WebSocket 服务器

│   │

│   ├── contract/               # Contract 服务

│   │   ├── registry.go         # Contract 注册表

│   │   ├── loader.go           # Contract 加载器

│   │   ├── validator.go        # Contract 验证器

│   │   └── generator.go        # Contract 生成器

│   │

│   ├── traffic/                # 流量记录

│   │   ├── recorder.go         # 流量记录器

│   │   ├── storage.go          # 流量存储

│   │   └── analyzer.go         # 流量分析器

│   │

│   ├── codegen/                # 代码生成

│   │   ├── generator.go        # 生成器接口

│   │   ├── go_generator.go     # Go 代码生成

│   │   └── template.go         # 代码模板

│   │

│   └── websocket/              # WebSocket 服务

│       ├── hub.go              # WebSocket Hub

│       ├── client.go           # WebSocket 客户端

│       └── message.go          # 消息定义

│

├── models/                     # 数据模型

│   ├── contract.go             # Contract 模型

│   ├── server.go               # Server 模型

│   ├── link.go                 # Link 模型

│   ├── traffic.go              # Traffic 模型

│   └── config.go               # Config 模型

│

├── store/                      # 存储层

│   ├── contract_store.go       # Contract 存储

│   ├── traffic_store.go        # 流量存储

│   ├── config_store.go         # 配置存储

│   └── db.go                   # 数据库封装

│

└── utils/                      # 工具函数

    ├── http.go                 # HTTP 工具

    ├── json.go                 # JSON 工具

    └── validator.go            # 验证工具

```

### 4.2 模块职责

#### 4.2.1 GUI Layer (views/)

| 模块 | 职责 |

|-----|------|

| `client/` | HTTP 客户端界面，发送请求、查看响应 |

| `server/` | 本地服务器管理，启停、路由、Mock |

| `contract/` | Contract CRUD、连接配置、代码生成 |

| `log/` | 流量日志查看、搜索、导出 |

| `common/` | 可复用 UI 组件 |

#### 4.2.2 Core Service Layer (services/)

| 模块 | 职责 |

|-----|------|

| `executor/` | 执行 HTTP 请求，处理响应 |

| `server/` | 管理本地 HTTP/Proxy/WebSocket 服务器 |

| `contract/` | Contract 注册、加载、验证、生成 |

| `traffic/` | 记录、存储、分析 HTTP 流量 |

| `codegen/` | 从 Contract 生成代码 |

| `websocket/` | WebSocket 实时通信 |

#### 4.2.3 Storage Layer (store/)

| 模块 | 职责 |

|-----|------|

| `contract_store` | Contract JSON 文件管理 |

| `traffic_store` | 流量记录持久化 (SQLite) |

| `config_store` | 用户配置管理 (YAML) |

---

## 5. 核心模块设计

### 5.1 Contract Registry

```go

// Contract 注册表 - 管理所有 Contract 的中心

typeContractRegistrystruct {

contractsmap[string]*Contract// name -> contract

linksmap[string]*Link// contractName -> link

watchers  []ContractWatcher// 变更监听器

}


typeContractstruct {

Namestring

Descriptionstring

RequestRequestSpec

ResponseResponseSpec

Structsmap[string]any

Mutablemap[string]any

}


typeLinkstruct {

ContractNamestring

ServerIDstring

ModeConnectionMode// Direct, Proxy, Mock

OptionsLinkOptions

}


typeConnectionModestring

const (

ModeDirectConnectionMode="direct"

ModeProxyConnectionMode="proxy"

ModeMockConnectionMode="mock"

)

```

### 5.2 Server Manager

```go

// Server 管理器 - 管理所有本地服务器

typeServerManagerstruct {

serversmap[string]Server// id -> server

}


typeServerinterface {

ID() string

Type() ServerType

Start() error

Stop() error

Status() ServerStatus

Address() string

}


typeServerTypestring

const (

TypeHTTPServerType="http"// Mock 服务器

TypeProxyServerType="proxy"// 代理服务器

TypeWebSocketServerType="websocket"// WebSocket 服务器

)


// HTTP Mock Server

typeHTTPServerstruct {

idstring

addrstring

router*Router

registry*ContractRegistry

}


// Proxy Server - 透明代理，记录流量

typeProxyServerstruct {

idstring

addrstring

targetstring// 目标服务器

recorder*TrafficRecorder

}

```

### 5.3 Request Executor

```go

// 请求执行器

typeExecutorstruct {

client*http.Client

registry*ContractRegistry

recorder*TrafficRecorder

hub*WebSocketHub

}


typeExecuteRequeststruct {

Contract*Contract

Link*Link

Overridesmap[string]any// 运行时覆盖的参数

}


typeExecuteResultstruct {

Request*http.Request

Response*http.Response

Body        []byte

Durationtime.Duration

Errorerror

Validation*ValidationResult

}


func (e *Executor) Execute(reqExecuteRequest) (*ExecuteResult, error) {

// 1. 根据 Link 决定目标

// 2. 构建 HTTP 请求

// 3. 发送请求

// 4. 验证响应是否符合 Contract

// 5. 记录流量

// 6. 通过 WebSocket 广播

// 7. 返回结果

}

```

### 5.4 Traffic Recorder

```go

// 流量记录器

typeTrafficRecorderstruct {

store*TrafficStore

hub*WebSocketHub

}


typeTrafficRecordstruct {

IDstring

Timestamptime.Time

ContractNamestring

RequestRequestRecord

ResponseResponseRecord

Durationtime.Duration

SourceTrafficSource// Client, Proxy

Validation*ValidationResult

}


typeRequestRecordstruct {

Methodstring

URLstring

Headersmap[string][]string

Body    []byte

}


typeResponseRecordstruct {

StatusCodeint

Headersmap[string][]string

Body       []byte

}

```

### 5.5 WebSocket Hub

```go

// WebSocket 消息中心

typeWebSocketHubstruct {

clientsmap[*Client]bool

broadcastchanMessage

registerchan*Client

unregisterchan*Client

}


typeMessagestruct {

TypeMessageType

Payloadany

}


typeMessageTypestring

const (

MsgTrafficMessageType="traffic"// 流量记录

MsgServerStatusMessageType="server_status"// 服务器状态

MsgValidationMessageType="validation"// 验证结果

MsgLogMessageType="log"// 日志

)

```

### 5.6 Contract Generator

```go

// Contract 生成器 - 从响应生成 Contract

typeContractGeneratorstruct {

registry*ContractRegistry

}


typeGenerateOptionsstruct {

Namestring

Methodstring

Pathstring

RequestBody    []byte

ResponseBody   []byte

ResponseStatusint

InferTypesbool// 是否推断类型

}


func (g *ContractGenerator) Generate(optsGenerateOptions) (*Contract, error) {

// 1. 解析请求体，推断 request.example

// 2. 解析响应体，推断 response.example

// 3. 推断 structs 定义

// 4. 生成 Contract

}


func (g *ContractGenerator) GenerateCode(contract*Contract, langstring) (string, error) {

// 复用 utils/genutil 的代码生成逻辑

}

```

---

## 6. 数据流详解

### 6.1 客户端发送请求流程

```

用户操作                     系统处理

────────────────────────────────────────────────────

1. 选择 Contract

        │

        ▼

2. 配置参数 (URL/Headers/Body)

        │

        ▼

3. 点击 Send

        │

        ├─────────────────────────────────────────┐

        ▼                                         │

4. Executor.Execute()                             │

        │                                         │

        ├─── 检查 Link 配置                        │

        │         │                               │

        │         ├─ Direct → 直接请求目标         │

        │         ├─ Proxy  → 经过代理服务器       │

        │         └─ Mock   → 请求本地 Mock        │

        │                                         │

        ▼                                         │

5. 发送 HTTP 请求                                  │

        │                                         │

        ▼                                         │

6. 接收响应                                        │

        │                                         │

        ├─── Contract.Validate(response)          │

        │         │                               │

        │         └─ 验证响应是否符合 Contract      │

        │                                         │

        ├─── TrafficRecorder.Record()             │

        │         │                               │

        │         └─ 记录到 SQLite                 │

        │                                         │

        ├─── WebSocketHub.Broadcast()             │

        │         │                               │

        │         └─ 实时推送到日志面板             │

        │                                         │

        ▼                                         │

7. 返回结果到 UI                                   │

        │                                         │

        └─────────────────────────────────────────┘

```

### 6.2 代理服务器流量记录流程

```

外部请求                      Proxy Server                     目标服务器

──────────────────────────────────────────────────────────────────────────

    │

    │  HTTP Request

    ▼

┌─────────────────────┐

│  Proxy Server       │

│  (localhost:8081)   │

├─────────────────────┤

│ 1. 接收请求          │

│ 2. 记录请求信息      │─────────────────────────────┐

│ 3. 转发到目标        │                             │

└──────────┬──────────┘                             │

           │                                        │

           │  Forward Request                       │

           ▼                                        │

    ┌─────────────────────┐                         │

    │  Target Server      │                         │

    │  (api.example.com)  │                         │

    └──────────┬──────────┘                         │

               │                                    │

               │  Response                          │

               ▼                                    │

┌─────────────────────┐                             │

│  Proxy Server       │                             │

├─────────────────────┤                             │

│ 4. 接收响应          │                             │

│ 5. 记录响应信息      │─────────────────────────────┤

│ 6. 匹配 Contract     │                             │

│ 7. 可选：生成 Contract│                             ▼

│ 8. 返回给客户端      │               ┌─────────────────────────┐

└──────────┬──────────┘               │  Traffic Recorder       │

           │                          ├─────────────────────────┤

           │  Response                │  - 保存到 SQLite         │

           ▼                          │  - 广播到 WebSocket      │

      原始客户端                        │  - 更新 UI               │

                                      └─────────────────────────┘

```

### 6.3 响应自动生成 Contract 流程

```

                  ┌─────────────────────────────────────────┐

                  │  TrafficRecord (请求/响应对)             │

                  └────────────────┬────────────────────────┘

                                   │

                                   ▼

                  ┌─────────────────────────────────────────┐

                  │  ContractGenerator.Generate()           │

                  ├─────────────────────────────────────────┤

                  │  1. 解析 Request URL → meta.path        │

                  │  2. 解析 Request Method → meta.method   │

                  │  3. 解析 Request Headers → example.headers│

                  │  4. 解析 Request Body → example.body    │

                  │  5. 解析 Response Body → response.example│

                  │  6. 推断类型 → structs                   │

                  └────────────────┬────────────────────────┘

                                   │

                                   ▼

                  ┌─────────────────────────────────────────┐

                  │  Contract JSON                          │

                  │  {                                      │

                  │    "name": "AutoGenerated_xxx",         │

                  │    "request": { ... },                  │

                  │    "response": { ... }                  │

                  │  }                                      │

                  └────────────────┬────────────────────────┘

                                   │

                    ┌──────────────┼──────────────┐

                    ▼              ▼              ▼

              保存为文件      注册到 Registry    生成代码

```

---

## 7. 接口设计

### 7.1 内部事件总线

```go

// 事件类型

typeEventTypestring

const (

EventContractLoadedEventType="contract.loaded"

EventContractUpdatedEventType="contract.updated"

EventServerStartedEventType="server.started"

EventServerStoppedEventType="server.stopped"

EventTrafficRecordedEventType="traffic.recorded"

EventRequestSentEventType="request.sent"

EventResponseReceivedEventType="response.received"

)


// 事件总线

typeEventBusstruct {

subscribersmap[EventType][]func(Event)

}


func (b *EventBus) Publish(eventEvent)

func (b *EventBus) Subscribe(eventTypeEventType, handlerfunc(Event))

```

### 7.2 WebSocket 消息协议

```json

// 客户端 -> 服务器

{

"type": "subscribe",

"topics": ["traffic", "server_status", "log"]

}


// 服务器 -> 客户端 (流量记录)

{

"type": "traffic",

"payload": {

"id": "uuid",

"timestamp": "2024-01-01T00:00:00Z",

"contract": "CreateUser",

"request": { ... },

"response": { ... },

"duration": 123,

"validation": { "valid": true }

  }

}


// 服务器 -> 客户端 (服务器状态)

{

"type": "server_status",

"payload": {

"id": "mock-server",

"status": "running",

"address": ":8080"

  }

}

```

---

## 8. 技术选型

| 层级 | 技术 | 理由 |

|-----|------|------|

| GUI | Fyne | 纯 Go、跨平台、GPU 加速 |

| HTTP Client | net/http | 标准库，无依赖 |

| HTTP Server | net/http | 标准库，灵活 |

| Proxy | httputil.ReverseProxy | 标准库 |

| WebSocket | gorilla/websocket | 成熟稳定 |

| Storage | SQLite (modernc.org/sqlite) | 纯 Go，无 CGO |

| Config | YAML | 人类可读 |

| JSON | encoding/json | 标准库 |

---

## 9. 实现路径

### Phase 1: 基础客户端 (当前)

- [X] 基础 UI 布局
- [ ] HTTP 请求发送
- [ ] 响应显示
- [ ] Contract 加载

### Phase 2: Mock 服务器

- [ ] HTTP Server 启停
- [ ] Contract 路由注册
- [ ] Mock 响应返回
- [ ] 请求验证

### Phase 3: 代理服务器

- [ ] Proxy Server 实现
- [ ] 流量记录
- [ ] 流量查看 UI
- [ ] 流量导出

### Phase 4: Contract 生成

- [ ] 从响应生成 Contract
- [ ] Contract 编辑器
- [ ] 代码生成集成

### Phase 5: WebSocket 实时通信

- [ ] WebSocket Server
- [ ] 实时日志推送
- [ ] 实时状态同步

### Phase 6: 高级功能

- [ ] 环境变量支持
- [ ] 请求脚本 (pre-request, test)
- [ ] 批量测试
- [ ] 导出/导入

---

## 10. 扩展性考虑

### 10.1 插件机制 (Future)

```go

typePlugininterface {

Name() string

Init(app*App) error


// 钩子

OnBeforeRequest(req*http.Request) error

OnAfterResponse(resp*http.Response) error

OnContractGenerated(contract*Contract) error

}

```

### 10.2 多语言代码生成

```go

typeCodeGeneratorinterface {

Language() string

Generate(contract*Contract) (string, error)

}


// 注册

registry.RegisterGenerator("go", &GoGenerator{})

registry.RegisterGenerator("typescript", &TSGenerator{})

registry.RegisterGenerator("python", &PythonGenerator{})

```

### 10.3 数据库连接 (Future)

```go

typeDatabaseConnectorinterface {

Connect(dsnstring) error

Query(sqlstring) ([]map[string]any, error)

Close() error

}


// Mock 服务器可连接真实数据库返回数据

mockServer.SetDatabaseConnector(connector)

```

---

## 11. 安全考虑

| 风险 | 措施 |

|-----|------|

| 敏感数据泄露 | Headers/Body 中的敏感字段支持遮罩 |

| 代理滥用 | 代理服务器仅监听 localhost |

| 流量存储过大 | 定期清理、大小限制 |

| 配置安全 | 支持环境变量引用敏感值 |

---

## 12. 总结

Nexus Web Workstation 是一个以 Contract 为核心的全功能 API 开发工具，通过统一的 Contract 定义连接客户端、服务器、代理、代码生成等多个模块，实现了：

1.**Contract First** - 所有功能围绕 Contract 展开

2.**双向验证** - 请求和响应都可以验证

3.**非侵入记录** - 代理模式透明记录流量

4.**实时协作** - WebSocket 实现实时日志

5.**协议即代码** - 自动生成类型安全代码

这不仅是一个 HTTP 客户端，而是一个完整的 Web API 开发工作站。
