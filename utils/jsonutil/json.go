package jsonutil

import (
	"encoding/json"
	"io"
	"net/http"
	"os"

	"github.com/go-playground/validator/v10"
)

func ValidateModel[T any](model *T) error {
	validator := validator.New()
	return validator.Struct(model)
}

func ParseJson[T any](data []byte) (*T, error) {
	var result T
	err := json.Unmarshal(data, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func ParseJsonAndValidate[T any](data []byte) (*T, error) {
	result, err := ParseJson[T](data)
	if err != nil {
		return nil, err
	}
	err = ValidateModel(result)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func ParseJsonFromReader[T any](r io.Reader) (*T, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	return ParseJson[T](data)
}

func ParseJsonAndValidateFromReader[T any](r io.Reader) (*T, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	return ParseJsonAndValidate[T](data)
}

func ParseJsonFromRequest[T any](r *http.Request) (*T, error) {
	return ParseJsonFromReader[T](r.Body)
}

func ParseJsonAndValidateFromRequest[T any](r *http.Request) (*T, error) {
	return ParseJsonAndValidateFromReader[T](r.Body)
}

func ParseJsonFromFile[T any](filePath string) (*T, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}
	return ParseJson[T](data)
}

func ParseJsonAndValidateFromFile[T any](filePath string) (*T, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}
	return ParseJsonAndValidate[T](data)
}
