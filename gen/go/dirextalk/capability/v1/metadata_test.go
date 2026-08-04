package capabilityv1

import (
	"bytes"
	"encoding/base64"
	"errors"
	"strings"
	"testing"
)

func testMetadataToken() string {
	return base64.RawURLEncoding.EncodeToString(bytes.Repeat([]byte{0x42}, CapabilityTokenBytes))
}

const testMetadataInstance = "018f2f1e-7b5b-7b21-8f2e-4a6f3b1d9c11"

func TestCapabilityMetadataRoundTrip(t *testing.T) {
	token := testMetadataToken()
	values, err := FormatCapabilityMetadata(token, testMetadataInstance, 9)
	if err != nil {
		t.Fatal(err)
	}
	got, err := ParseCapabilityMetadata(map[string][]string{
		CapabilityAuthorizationMetadata: {values[CapabilityAuthorizationMetadata]},
		CapabilityInstanceMetadata:      {values[CapabilityInstanceMetadata]},
		CapabilityGenerationMetadata:    {values[CapabilityGenerationMetadata]},
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.Token != token || got.InstanceID != testMetadataInstance || got.AccountGeneration != 9 {
		t.Fatalf("metadata mismatch: %#v", got)
	}
	raw, err := EncodeCapabilityToken(bytes.Repeat([]byte{0x42}, CapabilityTokenBytes))
	if err != nil || raw != token {
		t.Fatalf("raw token encoding = %q, %v", raw, err)
	}
}

func TestCapabilityTokenStrictParser(t *testing.T) {
	validToken := testMetadataToken()
	valid, err := FormatCapabilityToken(validToken)
	if err != nil || valid != "DTX-Capability-Token "+validToken {
		t.Fatalf("format = %q, %v", valid, err)
	}
	for _, value := range []string{
		"",
		"DTX-Agent-Token " + validToken,
		"DTX-Capability-Token",
		"DTX-Capability-Token  " + validToken,
		"DTX-Capability-Token " + validToken + "=",
		"DTX-Capability-Token " + validToken[:42],
		"DTX-Capability-Token " + validToken[:42] + "!",
	} {
		if _, err := ParseCapabilityToken(value); err == nil {
			t.Errorf("invalid token header %q accepted", value)
		}
	}
	if _, err := FormatCapabilityToken(strings.Repeat("a", CapabilityTokenLength)); err == nil {
		t.Error("non-base64 token accepted")
	}
	if _, err := EncodeCapabilityToken(make([]byte, CapabilityTokenBytes-1)); err == nil {
		t.Error("short raw token accepted")
	}
}

func TestCapabilityMetadataRejectsAmbiguousValues(t *testing.T) {
	base := map[string][]string{
		CapabilityAuthorizationMetadata: {"DTX-Capability-Token " + testMetadataToken()},
		CapabilityInstanceMetadata:      {testMetadataInstance},
		CapabilityGenerationMetadata:    {"1"},
	}
	cases := map[string]func(map[string][]string){
		"duplicate authorization": func(v map[string][]string) {
			v[CapabilityAuthorizationMetadata] = []string{v[CapabilityAuthorizationMetadata][0], v[CapabilityAuthorizationMetadata][0]}
		},
		"missing instance":     func(v map[string][]string) { delete(v, CapabilityInstanceMetadata) },
		"duplicate generation": func(v map[string][]string) { v[CapabilityGenerationMetadata] = []string{"1", "1"} },
		"leading zero":         func(v map[string][]string) { v[CapabilityGenerationMetadata] = []string{"01"} },
		"plus sign":            func(v map[string][]string) { v[CapabilityGenerationMetadata] = []string{"+1"} },
		"zero":                 func(v map[string][]string) { v[CapabilityGenerationMetadata] = []string{"0"} },
		"negative":             func(v map[string][]string) { v[CapabilityGenerationMetadata] = []string{"-1"} },
		"non uuid instance":    func(v map[string][]string) { v[CapabilityInstanceMetadata] = []string{"instance"} },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			values := make(map[string][]string, len(base))
			for key, items := range base {
				values[key] = append([]string(nil), items...)
			}
			mutate(values)
			if _, err := ParseCapabilityMetadata(values); !errors.Is(err, ErrInvalidCapabilityMetadata) {
				t.Fatalf("error = %v", err)
			}
		})
	}
}
