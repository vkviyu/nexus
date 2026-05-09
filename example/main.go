package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"

	"github.com/vkviyu/nexus/cmd"
	"github.com/vkviyu/nexus/transport/server/websocket"
	"github.com/vkviyu/nexus/utils/logutil"
)

type (
	SignalPayload struct {
		Type string          `json:"type"` // "offer", "answer"
		From string          `json:"from"` // 消息来源（由服务器填充）
		To   string          `json:"to"`   // 目标设备 ID
		Data json.RawMessage `json:"data"` // 具体的 SessionDescription
	}
)

func program(stopctx context.Context, config ServerConfig, cleanUpDone chan error) {
	host := config.Host
	port := config.Port

	authFunc := func(r *http.Request) (bool, string) {
		id := r.URL.Query().Get("id")
		if id == "" {
			return false, ""
		}

		return true, id
	}

	ep := websocket.NewEndpoint("/ws",
		websocket.WithAuthFunc(authFunc),
		websocket.WithReadErrorFunc(func(err error) {
			log.Printf("连接读取错误：%v", err)
		}),
	)

	go func() {
		msgChan := ep.GetMsgChan()

		for endpointMsg := range msgChan {
			fromID := "unknown"

			if len(endpointMsg.ConnIds) > 0 {
				fromID = endpointMsg.ConnIds[0]
			}

			var signal SignalPayload

			if err := json.Unmarshal(endpointMsg.Message.Message, &signal); err != nil {
				log.Printf("JSON 解析失败（来自 %s）：%v", fromID, err)
				continue
			}

			if signal.To == "" {
				log.Printf("消息没有目标 To（来自 %s）", fromID)
				continue
			}

			// 填充 From 字段（告诉接收方消息来源）
			signal.From = fromID

			// 重新序列化消息
			forwardPayload, err := json.Marshal(signal)
			if err != nil {
				log.Printf("序列化失败（来自 %s）：%v", fromID, err)
				continue
			}

			// 构造转发消息
			forwardMsg := &websocket.Message{
				MessageType: endpointMsg.Message.MessageType, // 保持原有类型
				Message:     forwardPayload,                  // 带有 From 的消息
				ConnIds:     []string{signal.To},             // 设置目标设备 ID
			}

			if err := ep.SendMessage(forwardMsg); err != nil {
				// ep.SendMessage 会返回 MessageSendError
				log.Printf("[Error] 转发失败 %s -> %s: %v", fromID, signal.To, err)
			} else {
				log.Printf("[Trace] 转发: %s -> %s (Type: %s)", fromID, signal.To, signal.Type)
			}

		}
	}()

	// -------------------------------------------------------
	// 3. 启动 HTTP 服务
	// -------------------------------------------------------

	// 将 Endpoint 注册到 http 处理链
	// 因为你的 Endpoint 实现了 ServeHTTP
	http.Handle("/ws", ep)

	addr := host + ":" + port
	log.Println(">>> FluxDesk 信令服务器启动在 " + addr + " <<<")
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatal(err)
	}
}

func main() {
	// 创建日志记录器
	logger, err := logutil.NewRotateLogger(nil)
	if err != nil {
		panic(err)
	}
	serverCmd := cmd.NewNexusCmd[ServerConfig](program)
	if err := serverCmd.Execute(); err != nil {
		logger.Errorf("serverCmd.Execute() error: %v", err)
		panic(err)
	}
}
