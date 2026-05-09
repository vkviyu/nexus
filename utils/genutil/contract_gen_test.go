package genutil

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGenerateContract(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "contract_gen_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	contractJSON := `{
  "name": "GetUser",
  "description": "Get user by ID",
  "request": {
    "meta": {
      "method": "GET",
      "path": "/users/{userId}",
      "baseURL": "https://api.example.com"
    },
    "example": {
      "pathParams": {
        "userId": "user123"
      },
      "query": {
        "includeProfile": true,
        "fields": "name,email"
      }
    },
    "structs": {
      "GetUserQuery": "query"
    }
  },
  "response": {
    "example": {
      "id": 1,
      "name": "John",
      "profile": {
        "age": 30,
        "city": "Beijing"
      }
    },
    "structs": {
      "UserProfile": "profile"
    }
  }
}`

	jsonPath := filepath.Join(tmpDir, "get_user.contract.json")
	if err := os.WriteFile(jsonPath, []byte(contractJSON), 0644); err != nil {
		t.Fatalf("failed to write test json: %v", err)
	}

	files, err := GenerateContract(ContractGenConfig{
		JSONPath:    jsonPath,
		PackageName: "client",
	})
	if err != nil {
		t.Fatalf("GenerateContract failed: %v", err)
	}

	if len(files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(files))
	}

	var content string
	for name, c := range files {
		if name != "get_user_contract.go" {
			t.Errorf("expected filename 'get_user_contract.go', got '%s'", name)
		}
		content = c
	}

	checks := []string{
		"package client",
		"GetUserQuery struct",
		"IncludeProfile bool",
		"GetUserResponse struct",
		"UserProfile struct",
		"var GetUser = func(",
		"userId string",
		"query *GetUserQuery",
	}

	for _, check := range checks {
		if !strings.Contains(content, check) {
			t.Errorf("expected to contain '%s'", check)
		}
	}
}

func TestGenerateContract_InlineStruct(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "contract_gen_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	contractJSON := `{
  "name": "CreateOrder",
  "request": {
    "meta": {"method": "POST", "path": "/orders"},
    "example": {
      "body": {
        "customer": {"name": "Alice", "phone": "123"},
        "items": [{"productId": "P001", "quantity": 2}]
      }
    },
    "structs": {}
  },
  "response": {"example": {"orderId": "ORD001"}}
}`

	jsonPath := filepath.Join(tmpDir, "create_order.contract.json")
	if err := os.WriteFile(jsonPath, []byte(contractJSON), 0644); err != nil {
		t.Fatalf("failed to write test json: %v", err)
	}

	files, err := GenerateContract(ContractGenConfig{
		JSONPath:    jsonPath,
		PackageName: "api",
	})
	if err != nil {
		t.Fatalf("GenerateContract failed: %v", err)
	}

	content := files["create_order_contract.go"]

	// Should contain inline struct syntax
	if !strings.Contains(content, "struct {") {
		t.Error("expected inline struct syntax")
	}
}

func TestGenerateContractsFromDir(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "contract_gen_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	contracts := map[string]string{
		"get_user.contract.json": `{
  "name": "GetUser",
  "request": {"meta": {"method": "GET", "path": "/users/{id}"}}
}`,
		"create_user.contract.json": `{
  "name": "CreateUser",
  "request": {
    "meta": {"method": "POST", "path": "/users"},
    "example": {"body": {"name": "test"}}
  }
}`,
		"config.json":       `{"key": "value"}`,
		"other.schema.json": `{"$schema": "..."}`,
	}

	for name, content := range contracts {
		path := filepath.Join(tmpDir, name)
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatalf("failed to write %s: %v", name, err)
		}
	}

	files, err := GenerateContractsFromDir(tmpDir, "client")
	if err != nil {
		t.Fatalf("GenerateContractsFromDir failed: %v", err)
	}

	if len(files) != 2 {
		t.Errorf("expected 2 files, got %d", len(files))
	}

	expectedFiles := []string{"get_user_contract.go", "create_user_contract.go"}
	for _, expected := range expectedFiles {
		if _, ok := files[expected]; !ok {
			t.Errorf("expected file '%s' not found", expected)
		}
	}
}

func TestGenerateContract_StructsControl(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "contract_gen_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	contractJSON := `{
  "name": "TestStructs",
  "request": {
    "meta": {"method": "POST", "path": "/test"},
    "example": {
      "body": {
        "declared": {"a": 1},
        "notDeclared": {"b": 2}
      }
    },
    "structs": {
      "DeclaredType": "body.declared"
    }
  },
  "response": {"example": {"ok": true}}
}`

	jsonPath := filepath.Join(tmpDir, "test.contract.json")
	if err := os.WriteFile(jsonPath, []byte(contractJSON), 0644); err != nil {
		t.Fatalf("failed to write test json: %v", err)
	}

	files, err := GenerateContract(ContractGenConfig{JSONPath: jsonPath})
	if err != nil {
		t.Fatalf("GenerateContract failed: %v", err)
	}

	content := files["test_structs_contract.go"]

	// DeclaredType should be independent struct
	if !strings.Contains(content, "DeclaredType struct {") {
		t.Error("DeclaredType should be an independent struct")
	}

	// notDeclared should exist as inline struct
	if !strings.Contains(content, "NotDeclared struct {") {
		t.Error("notDeclared should exist as inline struct")
	}
}

func TestGenerateContract_CompositeStruct(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "contract_gen_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	contractJSON := `{
  "name": "TestComposite",
  "request": {
    "meta": {"method": "POST", "path": "/test/{id}"},
    "example": {
      "query": {"year": 2024, "month": "January"},
      "body": {"name": "test"}
    },
    "structs": {
      "CompositeParams": {
        "ID": "pathParams.id",
        "Year": "query.year",
        "Month": "query.month"
      }
    }
  },
  "response": {"example": {"ok": true}}
}`

	jsonPath := filepath.Join(tmpDir, "test.contract.json")
	if err := os.WriteFile(jsonPath, []byte(contractJSON), 0644); err != nil {
		t.Fatalf("failed to write test json: %v", err)
	}

	files, err := GenerateContract(ContractGenConfig{JSONPath: jsonPath})
	if err != nil {
		t.Fatalf("GenerateContract failed: %v", err)
	}

	content := files["test_composite_contract.go"]

	// CompositeParams should be generated
	if !strings.Contains(content, "CompositeParams struct {") {
		t.Error("CompositeParams should be generated")
	}

	// Should have composite fields
	checks := []string{"ID string", "Year int", "Month string"}
	for _, check := range checks {
		if !strings.Contains(content, check) {
			t.Errorf("expected to contain '%s'", check)
		}
	}
}

func TestGenerateContract_StructRefWithFieldName(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "contract_gen_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	contractJSON := `{
  "name": "TestStructRef",
  "request": {
    "meta": {"method": "POST", "path": "/test"},
    "example": {
      "body": {
        "teacher": {"name": "Alice", "age": 30},
        "students": [{"name": "Bob", "score": 95}]
      }
    },
    "structs": {
      "TeacherInfo": "body.teacher",
      "StudentItem": "body.students",
      "ClassInput": {
        "@structs.TeacherInfo[Teacher]": "",
        "@structs.StudentItem": "",
        "Year": "path.year"
      }
    }
  },
  "response": {"example": {"ok": true}}
}`

	jsonPath := filepath.Join(tmpDir, "test.contract.json")
	if err := os.WriteFile(jsonPath, []byte(contractJSON), 0644); err != nil {
		t.Fatalf("failed to write test json: %v", err)
	}

	files, err := GenerateContract(ContractGenConfig{JSONPath: jsonPath})
	if err != nil {
		t.Fatalf("GenerateContract failed: %v", err)
	}

	content := files["test_struct_ref_contract.go"]

	// ClassInput should have custom field name "Teacher" with type "TeacherInfo"
	if !strings.Contains(content, "Teacher TeacherInfo") {
		t.Error("expected 'Teacher TeacherInfo' (custom field name)")
	}

	// StudentItem should use struct name as field name (default)
	if !strings.Contains(content, "StudentItem StudentItem") {
		t.Error("expected 'StudentItem StudentItem' (default field name)")
	}
}

func TestGenerateContract_CustomFuncParams(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "contract_gen_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	contractJSON := `{
  "name": "TestCustomFunc",
  "request": {
    "meta": {"method": "POST", "path": "/classes/{schoolId}"},
    "example": {
      "query": {"year": 2024, "semester": "spring"},
      "body": {"name": "test"}
    },
    "structs": {
      "ClassParams": {
        "SchoolID": "pathParams.schoolId",
        "Year": "query.year",
        "Semester": "query.semester"
      },
      "ClassBody": "body"
    },
    "func": {
      "params": [
        {"name": "params", "type": "ClassParams"},
        {"name": "body", "type": "ClassBody"}
      ]
    }
  },
  "response": {"example": {"ok": true}}
}`

	jsonPath := filepath.Join(tmpDir, "test.contract.json")
	if err := os.WriteFile(jsonPath, []byte(contractJSON), 0644); err != nil {
		t.Fatalf("failed to write test json: %v", err)
	}

	files, err := GenerateContract(ContractGenConfig{JSONPath: jsonPath})
	if err != nil {
		t.Fatalf("GenerateContract failed: %v", err)
	}

	content := files["test_custom_func_contract.go"]

	// Should have custom function parameters
	if !strings.Contains(content, "params *ClassParams") {
		t.Error("expected 'params *ClassParams' in function signature")
	}
	if !strings.Contains(content, "body *ClassBody") {
		t.Error("expected 'body *ClassBody' in function signature")
	}

	// Should use params.SchoolID in URL building
	if !strings.Contains(content, "params.SchoolID") {
		t.Error("expected 'params.SchoolID' in URL expression")
	}
}