package capabilityv1

import (
	"bytes"
	"encoding/hex"
	"testing"
)

func TestRootRequestDigestAvoidsGrantCycle(t *testing.T) {
	input := map[string]interface{}{"message": "hello"}
	schema := bytes.Repeat([]byte{0x10}, 32)
	grant := bytes.Repeat([]byte{0x42}, 32)
	root, err := ComputeRootRequestDigest(1, "agent.chat.v1", "1.0.0", schema, "chat", 0, input, nil)
	if err != nil {
		t.Fatal(err)
	}
	withoutGrant, err := computeRequestDigest(1, "agent.chat.v1", "1.0.0", schema, "chat", 0, input, nil, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(root, withoutGrant) {
		t.Fatal("root digest does not use the documented grant-independent preimage")
	}
	final, err := ComputeRequestDigest(1, "agent.chat.v1", "1.0.0", schema, "chat", 0, input, nil, grant)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(root, final) {
		t.Fatal("final request digest did not bind the signed grant")
	}
}

func TestRequestDigestRequiresSHA256Inputs(t *testing.T) {
	schema := bytes.Repeat([]byte{1}, 32)
	grant := bytes.Repeat([]byte{2}, 32)
	input := map[string]interface{}{"ok": true}
	if _, err := ComputeRequestDigest(1, "agent.chat.v1", "1.0.0", []byte("short"), "chat", 0, input, nil, grant); err == nil {
		t.Fatal("short schema digest accepted")
	}
	if _, err := ComputeRequestDigest(1, "agent.chat.v1", "1.0.0", schema, "chat", 0, input, nil, []byte("short")); err == nil {
		t.Fatal("short grant digest accepted")
	}
	if _, err := ComputeRequestDigest(1, "agent.chat.v1", "1.0.0", schema, "chat", 0, input, [][]byte{[]byte("short")}, grant); err == nil {
		t.Fatal("short attachment digest accepted")
	}
	if _, err := ComputeRootRequestDigest(1, "agent.chat.v1", "1.0.0", []byte("short"), "chat", 0, input, nil); err == nil {
		t.Fatal("root digest accepted short schema digest")
	}
}

func TestComputeRequestDigest(t *testing.T) {
	tests := []struct {
		name                  string
		protocolMajor         int32
		capabilityID          string
		capabilityVersion     string
		schemaDigest          []byte
		operation             string
		expectedRevision      int64
		businessInput         map[string]interface{}
		attachmentRefDigests  [][]byte
		permissionGrantDigest []byte
		wantDeterministic     bool
	}{
		{
			name:              "simple input",
			protocolMajor:     1,
			capabilityID:      "agent.chat.v1",
			capabilityVersion: "1.0.0",
			schemaDigest:      bytes.Repeat([]byte{0x11}, 32),
			operation:         "send_message",
			expectedRevision:  0,
			businessInput: map[string]interface{}{
				"message": "hello",
			},
			attachmentRefDigests:  nil,
			permissionGrantDigest: bytes.Repeat([]byte{0x12}, 32),
			wantDeterministic:     true,
		},
		{
			name:              "complex input",
			protocolMajor:     1,
			capabilityID:      "agent.chat.v1",
			capabilityVersion: "1.0.0",
			schemaDigest:      bytes.Repeat([]byte{0x11}, 32),
			operation:         "send_message",
			expectedRevision:  5,
			businessInput: map[string]interface{}{
				"message": "hello",
				"context": map[string]interface{}{
					"thread_id": "abc123",
					"priority":  1,
				},
				"attachments": []interface{}{"ref1", "ref2"},
			},
			attachmentRefDigests:  [][]byte{bytes.Repeat([]byte{0x13}, 32), bytes.Repeat([]byte{0x14}, 32)},
			permissionGrantDigest: bytes.Repeat([]byte{0x12}, 32),
			wantDeterministic:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 计算两次，确保确定性
			digest1, err := ComputeRequestDigest(
				tt.protocolMajor,
				tt.capabilityID,
				tt.capabilityVersion,
				tt.schemaDigest,
				tt.operation,
				tt.expectedRevision,
				tt.businessInput,
				tt.attachmentRefDigests,
				tt.permissionGrantDigest,
			)
			if err != nil {
				t.Fatalf("ComputeRequestDigest() error = %v", err)
			}

			digest2, err := ComputeRequestDigest(
				tt.protocolMajor,
				tt.capabilityID,
				tt.capabilityVersion,
				tt.schemaDigest,
				tt.operation,
				tt.expectedRevision,
				tt.businessInput,
				tt.attachmentRefDigests,
				tt.permissionGrantDigest,
			)
			if err != nil {
				t.Fatalf("ComputeRequestDigest() second call error = %v", err)
			}

			if tt.wantDeterministic {
				if hex.EncodeToString(digest1) != hex.EncodeToString(digest2) {
					t.Errorf("Digest not deterministic: %x vs %x", digest1, digest2)
				}
			}

			t.Logf("Digest: %x", digest1)
		})
	}
}

func TestCanonicalJSON(t *testing.T) {
	tests := []struct {
		name  string
		input interface{}
		want  string
	}{
		{
			name:  "null",
			input: nil,
			want:  "null",
		},
		{
			name:  "bool true",
			input: true,
			want:  "true",
		},
		{
			name:  "bool false",
			input: false,
			want:  "false",
		},
		{
			name:  "integer",
			input: 42,
			want:  "42",
		},
		{
			name:  "string",
			input: "hello",
			want:  `"hello"`,
		},
		{
			name:  "empty array",
			input: []interface{}{},
			want:  "[]",
		},
		{
			name:  "simple array",
			input: []interface{}{1, 2, 3},
			want:  "[1,2,3]",
		},
		{
			name:  "empty object",
			input: map[string]interface{}{},
			want:  "{}",
		},
		{
			name: "simple object",
			input: map[string]interface{}{
				"a": 1,
				"b": 2,
			},
			want: `{"a":1,"b":2}`,
		},
		{
			name: "object with sorted keys",
			input: map[string]interface{}{
				"z": 1,
				"a": 2,
				"m": 3,
			},
			want: `{"a":2,"m":3,"z":1}`,
		},
		{
			name: "nested object",
			input: map[string]interface{}{
				"outer": map[string]interface{}{
					"b": 2,
					"a": 1,
				},
			},
			want: `{"outer":{"a":1,"b":2}}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := toCanonicalJSON(tt.input)
			if err != nil {
				t.Fatalf("toCanonicalJSON() error = %v", err)
			}
			if got != tt.want {
				t.Errorf("toCanonicalJSON() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestSerializeAndParseBusinessInput(t *testing.T) {
	tests := []struct {
		name  string
		input map[string]interface{}
	}{
		{
			name: "simple input",
			input: map[string]interface{}{
				"message": "hello",
				"count":   42,
			},
		},
		{
			name: "nested input",
			input: map[string]interface{}{
				"message": "hello",
				"context": map[string]interface{}{
					"thread_id": "abc123",
					"priority":  1,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Serialize
			canonical, err := SerializeBusinessInput(tt.input)
			if err != nil {
				t.Fatalf("SerializeBusinessInput() error = %v", err)
			}

			t.Logf("Canonical JSON: %s", string(canonical))

			// Parse back
			parsed, err := ParseBusinessInput(canonical)
			if err != nil {
				t.Fatalf("ParseBusinessInput() error = %v", err)
			}

			// Re-serialize to check consistency
			canonical2, err := SerializeBusinessInput(parsed)
			if err != nil {
				t.Fatalf("SerializeBusinessInput() second call error = %v", err)
			}

			if string(canonical) != string(canonical2) {
				t.Errorf("Canonical JSON not consistent: %s vs %s", string(canonical), string(canonical2))
			}
		})
	}
}

func TestValidateDigest(t *testing.T) {
	digest1 := []byte{0x01, 0x02, 0x03, 0x04}
	digest2 := []byte{0x01, 0x02, 0x03, 0x04}
	digest3 := []byte{0x01, 0x02, 0x03, 0x05}
	digest4 := []byte{0x01, 0x02, 0x03}

	tests := []struct {
		name     string
		actual   []byte
		expected []byte
		wantErr  bool
	}{
		{
			name:     "matching digests",
			actual:   digest1,
			expected: digest2,
			wantErr:  false,
		},
		{
			name:     "different digests",
			actual:   digest1,
			expected: digest3,
			wantErr:  true,
		},
		{
			name:     "different lengths",
			actual:   digest1,
			expected: digest4,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateDigest(tt.actual, tt.expected)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateDigest() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestDigestDeterminismWithKeyOrder(t *testing.T) {
	// 测试不同的键顺序产生相同的 digest
	input1 := map[string]interface{}{
		"a": 1,
		"b": 2,
		"c": 3,
	}

	input2 := map[string]interface{}{
		"c": 3,
		"a": 1,
		"b": 2,
	}

	input3 := map[string]interface{}{
		"b": 2,
		"c": 3,
		"a": 1,
	}

	schema := bytes.Repeat([]byte{0x21}, 32)
	grant := bytes.Repeat([]byte{0x22}, 32)
	digest1, err := ComputeRequestDigest(1, "test", "1.0.0", schema, "op", 0, input1, nil, grant)
	if err != nil {
		t.Fatalf("ComputeRequestDigest(input1) error = %v", err)
	}

	digest2, err := ComputeRequestDigest(1, "test", "1.0.0", schema, "op", 0, input2, nil, grant)
	if err != nil {
		t.Fatalf("ComputeRequestDigest(input2) error = %v", err)
	}

	digest3, err := ComputeRequestDigest(1, "test", "1.0.0", schema, "op", 0, input3, nil, grant)
	if err != nil {
		t.Fatalf("ComputeRequestDigest(input3) error = %v", err)
	}

	if hex.EncodeToString(digest1) != hex.EncodeToString(digest2) {
		t.Errorf("Different key order produced different digests: %x vs %x", digest1, digest2)
	}

	if hex.EncodeToString(digest1) != hex.EncodeToString(digest3) {
		t.Errorf("Different key order produced different digests: %x vs %x", digest1, digest3)
	}

	t.Logf("All three inputs produced the same digest: %x", digest1)
}
