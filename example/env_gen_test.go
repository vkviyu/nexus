package main

import (
	"os"
	"testing"

	"github.com/vkviyu/nexus/utils/genutil"
)

func TestGenEnv(t *testing.T) {
	fileStr, err := genutil.GenerateStructFromYAML("./nexus.yaml", "nexus.environment", "main", "ServerConfig")
	if err != nil {
		t.Fatalf("Failed to generate struct: %v", err)
	}
	// 写入文件
	os.WriteFile("env_gen.go", []byte(fileStr), 0644)
}

func TestParseStruct(t *testing.T) {
	// 模拟从配置文件解析出的 map 数据
	envMap := map[string]any{
		"host": "0.0.0.0",
		"port": "8000",
	}

	// 使用泛型 ParseStruct 解析
	config, err := genutil.ParseStruct[ServerConfig](envMap)
	if err != nil {
		t.Fatalf("Failed to parse struct: %v", err)
	}

	// 验证解析结果
	if config.Host != "0.0.0.0" {
		t.Errorf("Expected host '0.0.0.0', got '%s'", config.Host)
	}
	if config.Port != "8000" {
		t.Errorf("Expected port '8000', got '%s'", config.Port)
	}
}

func TestGenerateProgram(t *testing.T) {
	// 测试脚手架生成功能
	files, err := genutil.GenerateProgram(genutil.ProgramConfig{
		PackageName: "main",
		StructName:  "ServerConfig",
		YAMLPath:    "./nexus.yaml",
		WithLogger:  true,
	})
	if err != nil {
		t.Fatalf("Failed to generate program: %v", err)
	}

	// 验证生成了两个文件
	if len(files) != 2 {
		t.Errorf("Expected 2 files, got %d", len(files))
	}

	// 验证 main.go 存在且包含关键内容
	mainContent, ok := files["main.go"]
	if !ok {
		t.Fatal("main.go not generated")
	}
	if !contains(mainContent, "func program(stopctx context.Context") {
		t.Error("main.go missing program function")
	}
	if !contains(mainContent, "func main()") {
		t.Error("main.go missing main function")
	}
	if !contains(mainContent, "cmd.NewNexusCmd[ServerConfig]") {
		t.Error("main.go missing generic NewNexusCmd call")
	}
	if !contains(mainContent, "logutil.NewRotateLogger") {
		t.Error("main.go missing logger initialization")
	}

	// 验证 config_gen.go 存在且包含结构体
	configContent, ok := files["config_gen.go"]
	if !ok {
		t.Fatal("config_gen.go not generated")
	}
	if !contains(configContent, "type ServerConfig struct") {
		t.Error("config_gen.go missing ServerConfig struct")
	}

	t.Logf("Generated main.go:\n%s", mainContent)
	t.Logf("Generated config_gen.go:\n%s", configContent)
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
