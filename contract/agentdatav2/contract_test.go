package agentdatav2_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
)

type manifest struct {
	Contract string   `json:"contract"`
	Version  int      `json:"version"`
	Vectors  []vector `json:"vectors"`
}

type vector struct {
	File   string `json:"file"`
	Schema string `json:"schema"`
	Valid  bool   `json:"valid"`
	Rule   string `json:"rule,omitempty"`
}

func TestAgentDataPlaneV2Contract(t *testing.T) {
	root := filepath.Join("..", "..")
	contractPath := filepath.Join(root, "api", "openapi", "agent-data-plane-v2.yaml")
	loader := openapi3.NewLoader()
	document, err := loader.LoadFromFile(contractPath)
	if err != nil {
		t.Fatalf("load OpenAPI contract: %v", err)
	}
	if err := document.Validate(context.Background(), openapi3.EnableSchemaFormatValidation()); err != nil {
		t.Fatalf("validate OpenAPI contract: %v", err)
	}

	vectorRoot := filepath.Join(root, "conformance", "agent-data-plane", "v2")
	data, err := os.ReadFile(filepath.Join(vectorRoot, "manifest.json"))
	if err != nil {
		t.Fatalf("read conformance manifest: %v", err)
	}
	var suite manifest
	decodeStrict(t, data, &suite)
	if suite.Contract != "dirextalk-agent-data-plane" || suite.Version != 2 || len(suite.Vectors) == 0 {
		t.Fatalf("invalid conformance manifest identity: %#v", suite)
	}

	for _, item := range suite.Vectors {
		item := item
		t.Run(item.File, func(t *testing.T) {
			schema, ok := document.Components.Schemas[item.Schema]
			if !ok || schema == nil || schema.Value == nil {
				t.Fatalf("unknown component schema %q", item.Schema)
			}
			raw, err := os.ReadFile(filepath.Join(vectorRoot, filepath.FromSlash(item.File)))
			if err != nil {
				t.Fatalf("read vector: %v", err)
			}
			var value any
			decodeStrict(t, raw, &value)
			validationErr := schema.Value.VisitJSON(value, openapi3.MultiErrors(), openapi3.EnableFormatValidation())
			if validationErr == nil {
				validationErr = applySemanticRule(item.Rule, value)
			}
			if item.Valid && validationErr != nil {
				t.Fatalf("valid vector rejected: %v", validationErr)
			}
			if !item.Valid && validationErr == nil {
				t.Fatal("invalid vector was accepted")
			}
		})
	}
}

func decodeStrict(t *testing.T, data []byte, target any) {
	t.Helper()
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		t.Fatalf("decode JSON: %v", err)
	}
	var trailing json.RawMessage
	if err := decoder.Decode(&trailing); err != io.EOF {
		t.Fatalf("JSON has trailing data: %v", err)
	}
}

func applySemanticRule(rule string, value any) error {
	switch rule {
	case "":
		return nil
	case "operation_id_equals_turn_id":
		object, ok := value.(map[string]any)
		if !ok {
			return fmt.Errorf("identity rule requires an object")
		}
		operationID, operationOK := object["operation_id"].(string)
		turnID, turnOK := object["turn_id"].(string)
		if !operationOK || !turnOK || operationID == "" || operationID != turnID {
			return fmt.Errorf("operation_id and turn_id must be equal")
		}
		return nil
	default:
		return fmt.Errorf("unknown semantic rule %q", rule)
	}
}
