package example

import (
	"os"
	"strings"
	"testing"

	"github.com/vkviyu/nexus/utils/genutil"
)

func TestGenerateCreateClassContract(t *testing.T) {
	files, err := genutil.GenerateContract(genutil.ContractGenConfig{
		JSONPath:    "create_class.contract.json",
		PackageName: "client",
	})
	if err != nil {
		t.Fatalf("GenerateContract failed: %v", err)
	}

	content := files["create_class_contract.go"]

	// Verify basic structure (with custom func.params)
	checks := []string{
		"package client",
		"CreateClassQuery struct",
		"CreateClassBody struct",
		"CreateClassResponse struct",
		"StudentItem struct",
		"ClassResult struct",
		"ClassDetail struct",
		"ClassParams struct",
		"var CreateClass = func(",
		"params *ClassParams",
		"body *CreateClassBody",
		"*Contract[CreateClassResponse]",
		"params.SchoolID",
	}

	for _, check := range checks {
		if !strings.Contains(content, check) {
			t.Errorf("expected to contain '%s'", check)
		}
	}
	os.WriteFile("create_class_contract_gen.go", []byte(content), 0644)
}

func TestGenerateContractsFromClientDir(t *testing.T) {
	files, err := genutil.GenerateContractsFromDir(".", "example")
	if err != nil {
		t.Fatalf("GenerateContractsFromDir failed: %v", err)
	}

	if len(files) == 0 {
		t.Error("expected at least one contract file")
	}

	// Should find create_class.contract.json
	if _, ok := files["create_class_contract.go"]; !ok {
		t.Error("expected create_class_contract.go to be generated")
	}

	// 保存文件
	for name, content := range files {
		os.WriteFile(name, []byte(content), 0644)
	}
}
