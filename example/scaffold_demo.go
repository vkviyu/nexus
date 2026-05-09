//go:build ignore

// 这是一个演示脚手架 API 的示例脚本
// 运行方式: go run scaffold_demo.go
package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/vkviyu/nexus/utils/genutil"
)

func main() {
	// 创建输出目录
	outputDir := "./scaffold_output"
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		panic(err)
	}

	// 使用脚手架 API 生成项目框架
	files, err := genutil.GenerateProgram(genutil.ProgramConfig{
		PackageName: "main",
		StructName:  "ServerConfig",
		YAMLPath:    "./nexus.yaml",
		WithLogger:  true,
	})
	if err != nil {
		panic(err)
	}

	// 写入生成的文件
	for filename, content := range files {
		outputPath := filepath.Join(outputDir, filename)
		if err := os.WriteFile(outputPath, []byte(content), 0644); err != nil {
			panic(err)
		}
		fmt.Printf("✅ 已生成: %s\n", outputPath)
	}

	fmt.Println("\n🎉 脚手架生成完成！")
	fmt.Println("生成的文件位于:", outputDir)
	fmt.Println("\n📝 生成的 main.go 内容预览:")
	fmt.Println("─────────────────────────────")
	fmt.Println(files["main.go"])
}