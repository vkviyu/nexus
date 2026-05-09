package domain

import (
	"fmt"
	"reflect"
	"time"

	"github.com/traefik/yaegi/interp"
	"github.com/traefik/yaegi/stdlib"
)

// ============================================================================
// Script Engine - Yaegi Go 解释器封装
// ============================================================================

// ScriptEngine 脚本执行引擎
type ScriptEngine struct {
	timeout time.Duration
}

// NewScriptEngine 创建脚本引擎
func NewScriptEngine() *ScriptEngine {
	return &ScriptEngine{
		timeout: 5 * time.Second, // 默认 5 秒超时
	}
}

// SetTimeout 设置脚本执行超时时间
func (e *ScriptEngine) SetTimeout(d time.Duration) {
	e.timeout = d
}

// ============================================================================
// Script Execution
// ============================================================================

// ExecuteResult 脚本执行结果
type ExecuteResult struct {
	Action string // "next" 或 "terminate"
	Error  error
}

// Execute 执行 Go 脚本
// 脚本中可以访问 ctx 变量（*ScriptContext 类型）
// 脚本应该返回 "next" 继续执行下一个行为，或返回 "terminate" 终止行为链
func (e *ScriptEngine) Execute(scriptCtx *ScriptContext, code string) ExecuteResult {
	resultCh := make(chan ExecuteResult, 1)

	go func() {
		resultCh <- e.executeInternal(scriptCtx, code)
	}()

	select {
	case result := <-resultCh:
		return result
	case <-time.After(e.timeout):
		return ExecuteResult{
			Action: "terminate",
			Error:  fmt.Errorf("script execution timeout after %v", e.timeout),
		}
	}
}

// executeInternal 内部执行逻辑
func (e *ScriptEngine) executeInternal(scriptCtx *ScriptContext, code string) ExecuteResult {
	// 创建新的解释器实例
	i := interp.New(interp.Options{})

	// 注入 Go 标准库（用户可以 import 任意标准库包）
	if err := i.Use(stdlib.Symbols); err != nil {
		return ExecuteResult{Error: fmt.Errorf("failed to load stdlib: %w", err)}
	}

	// 注入自定义 symbols（ScriptContext 类型和 ctx 变量）
	if err := i.Use(e.createCustomSymbols(scriptCtx)); err != nil {
		return ExecuteResult{Error: fmt.Errorf("failed to inject context: %w", err)}
	}

	// 包装用户代码为完整的 Go 程序
	wrappedCode := e.wrapUserCode(code)

	// 执行脚本
	result, err := i.Eval(wrappedCode)
	if err != nil {
		return ExecuteResult{Error: fmt.Errorf("script error: %w", err)}
	}

	// 解析返回值
	action := "next"
	if result.IsValid() && result.Kind() == reflect.String {
		action = result.String()
	}

	return ExecuteResult{Action: action}
}

// createCustomSymbols 创建自定义 symbols 供脚本使用
func (e *ScriptEngine) createCustomSymbols(ctx *ScriptContext) interp.Exports {
	return interp.Exports{
		"nexus/nexus": map[string]reflect.Value{
			// 导出类型
			"ScriptContext":  reflect.ValueOf((*ScriptContext)(nil)),
			"ScriptRequest":  reflect.ValueOf((*ScriptRequest)(nil)),
			"ScriptResponse": reflect.ValueOf((*ScriptResponse)(nil)),
			// 导出 ctx 实例
			"Ctx": reflect.ValueOf(ctx),
		},
	}
}

// wrapUserCode 包装用户代码为可执行的 Go 程序
func (e *ScriptEngine) wrapUserCode(userCode string) string {
	// 用户代码将被包装在一个函数中执行
	// ctx 变量通过 nexus 包导入
	return fmt.Sprintf(`
package main

import (
	"nexus"
)

func main() string {
	ctx := nexus.Ctx
	_ = ctx // 避免未使用警告
	
	// ========== 用户代码开始 ==========
	%s
	// ========== 用户代码结束 ==========
	
	return "next" // 默认继续执行
}
`, userCode)
}

// ============================================================================
// Convenience Methods
// ============================================================================

// ExecuteSimple 执行简单脚本，不关心返回值
func (e *ScriptEngine) ExecuteSimple(scriptCtx *ScriptContext, code string) error {
	result := e.Execute(scriptCtx, code)
	return result.Error
}