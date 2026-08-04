package capabilityv1

import (
	"bytes"
	"encoding/hex"
	"errors"
	"math"
	"testing"
)

func TestRFC8785AppendixBGolden(t *testing.T) {
	input := []byte(`{"numbers":[333333333.33333329,1E+30,4.50,2e-3,0.000000000000000000000000001],"literals":[null,true,false],"string":"€$\\u000f\\nA'B\\\"\\\\\\\"/"}`)
	want := `{"literals":[null,true,false],"numbers":[333333333.3333333,1e+30,4.5,0.002,1e-27],"string":"€$\\u000f\\nA'B\\\"\\\\\\\"/"}`
	got, err := CanonicalizeJSON(input)
	if err != nil {
		t.Fatalf("CanonicalizeJSON() error = %v", err)
	}
	if string(got) != want {
		t.Fatalf("RFC 8785 golden mismatch\n got: %s\nwant: %s", got, want)
	}
	canonicalAgain, err := CanonicalizeJSON(got)
	if err != nil || !bytes.Equal(canonicalAgain, got) {
		t.Fatalf("canonical output is not idempotent: %q (%v)", canonicalAgain, err)
	}
}

func TestCanonicalNumberThresholds(t *testing.T) {
	tests := map[string]string{
		"1e-7":  "1e-7",
		"1e-6":  "0.000001",
		"1e20":  "100000000000000000000",
		"1e21":  "1e+21",
		"-0":    "0",
		"-1.25": "-1.25",
	}
	for input, want := range tests {
		got, err := CanonicalizeJSON([]byte(input))
		if err != nil {
			t.Errorf("%s: unexpected error: %v", input, err)
			continue
		}
		if string(got) != want {
			t.Errorf("%s: got %s, want %s", input, got, want)
		}
	}
}

func TestCanonicalStringEscaping(t *testing.T) {
	input := "quote:\" slash:\\ control:\x00\x08\x1f separators:\u2028\u2029"
	got, err := Canonicalize(input)
	if err != nil {
		t.Fatal(err)
	}
	want := `"quote:\" slash:\\ control:\u0000\b\u001f separators:` + "\u2028\u2029" + `"`
	if string(got) != want {
		t.Fatalf("string escaping got %q, want %q", got, want)
	}
}

func TestCanonicalIntegerSafetyBoundaries(t *testing.T) {
	for _, value := range []int64{-(1<<53 - 1), 1<<53 - 1, -(1 << 53), 1 << 53, 100000000000000000} {
		if _, err := Canonicalize(value); err != nil {
			t.Errorf("exact IEEE-754 integer %d rejected: %v", value, err)
		}
	}
	for _, value := range []int64{-(1<<53 + 1), 1<<53 + 1, 100000000000000001} {
		if _, err := Canonicalize(value); err == nil {
			t.Errorf("inexact IEEE-754 integer %d accepted", value)
		}
	}
	for _, value := range []string{"9007199254740991", "-9007199254740991", "9007199254740992", "-9007199254740992", "100000000000000000"} {
		if _, err := CanonicalizeJSON([]byte(value)); err != nil {
			t.Errorf("safe JSON integer %s rejected: %v", value, err)
		}
	}
	for _, value := range []string{"9007199254740993", "-9007199254740993", "100000000000000001"} {
		if _, err := CanonicalizeJSON([]byte(value)); err == nil {
			t.Errorf("unsafe JSON integer %s accepted", value)
		}
	}
}

func TestCanonicalObjectSortsUTF16CodeUnits(t *testing.T) {
	// U+10000 encodes as D800 DC00 in UTF-16, so it sorts before U+E000.
	input := map[string]interface{}{"\ue000": 2, "\U00010000": 1}
	got, err := Canonicalize(input)
	if err != nil {
		t.Fatal(err)
	}
	want := `{"𐀀":1,"":2}`
	if string(got) != want {
		t.Fatalf("got %s, want %s", got, want)
	}
}

func TestCanonicalJSONRejectsUnsafeInput(t *testing.T) {
	tests := [][]byte{
		[]byte(`{"duplicate":1,"duplicate":2}`),
		[]byte(`"\ud800"`),
		[]byte(`"\udc00"`),
		[]byte(`NaN`),
		[]byte(`1 2`),
	}
	for _, input := range tests {
		if _, err := CanonicalizeJSON(input); err == nil {
			t.Errorf("CanonicalizeJSON(%q) unexpectedly succeeded", input)
		}
	}
	cycle := map[string]interface{}{}
	cycle["self"] = cycle
	if _, err := Canonicalize(cycle); err == nil {
		t.Error("Canonicalize(cyclic map) unexpectedly succeeded")
	}
	if _, err := Canonicalize(math.NaN()); err == nil {
		t.Error("Canonicalize(NaN) unexpectedly succeeded")
	}
}

func TestRequestDigestGolden(t *testing.T) {
	digest, err := ComputeRequestDigest(
		1,
		"product.echo.v1",
		"1.0.0",
		bytes.Repeat([]byte{0x31}, 32),
		"echo",
		0,
		map[string]interface{}{"message": "test"},
		nil,
		bytes.Repeat([]byte{0x32}, 32),
	)
	if err != nil {
		t.Fatal(err)
	}
	const want = "af6d8d327ac1b794b6aabd640ad4f0e3d2930dff09bf39eec85b9a2ab0e1c422"
	if got := hex.EncodeToString(digest); got != want {
		t.Fatalf("digest = %s, want %s", got, want)
	}
	if err := ValidateRequestDigest(digest); err != nil {
		t.Fatal(err)
	}
}

func TestOperationAndRequestValidation(t *testing.T) {
	validID := "018f2f1e-7b5b-7b21-8f2e-4a6f3b1d9c11"
	if err := ValidateOperationID(validID); err != nil {
		t.Fatalf("valid operation ID rejected: %v", err)
	}
	for _, invalid := range []string{"", "OP-1", "00000000-0000-0000-0000-000000000000", "018f2f1e-7b5b-7b21-8f2e-4a6f3b1d9c1"} {
		if err := ValidateOperationID(invalid); err == nil {
			t.Errorf("invalid operation ID %q accepted", invalid)
		}
	}
	ctx := NewCallContext(validID, validID, 2_000_000_000_000)
	ctx.ChainId = validID
	requestJSON := []byte(`{"message":"hello"}`)
	requestDigest := make([]byte, 32)
	req := &StartOperationRequest{
		CallContext: ctx,
		Permission: &PermissionContext{
			AuthenticatedOwnerId: "owner-1",
			GrantedScopes:        []string{"capability:execute"},
			CapabilityGrant:      []byte("signed-grant"),
			AccountGeneration:    1,
		},
		OperationId:   validID,
		CapabilityId:  "product.echo.v1",
		Operation:     "echo",
		RequestJson:   requestJSON,
		RequestDigest: requestDigest,
	}
	if err := ValidateOperationRequest(req); err != nil {
		t.Fatalf("valid request rejected: %v", err)
	}
	if err := ValidateRootOperationRequest(req); err != nil {
		t.Fatalf("valid root request rejected: %v", err)
	}
	req.OperationId = "018f2f1e-7b5b-7b21-8f2e-4a6f3b1d9c12"
	if err := ValidateOperationRequest(req); err != nil {
		t.Fatalf("valid child operation rejected: %v", err)
	}
	if err := ValidateRootOperationRequest(req); err == nil {
		t.Fatal("root request with a different operation_id was accepted")
	}
	req.OperationId = validID
	req.RequestJson = []byte(`{ "message": "hello" }`)
	if err := ValidateOperationRequest(req); err == nil {
		t.Error("non-canonical request JSON accepted")
	}
	req.RequestJson = requestJSON
	req.ExpectedRevision = -1
	if err := ValidateOperationRequest(req); err == nil {
		t.Error("negative expected revision accepted")
	}
	req.ExpectedRevision = 0
	req.Permission = nil
	if err := ValidateOperationRequest(req); err == nil {
		t.Error("missing operation permission accepted")
	}
	query := &QueryRequest{
		CallContext:  ctx,
		Permission:   &PermissionContext{AuthenticatedOwnerId: "owner-1", GrantedScopes: []string{"capability:execute"}, CapabilityGrant: []byte("signed-grant"), AccountGeneration: 1},
		CapabilityId: "product.echo.v1",
		OperationId:  "echo",
		RequestJson:  requestJSON,
	}
	if err := ValidateQueryRequest(query); err != nil {
		t.Fatalf("valid query rejected: %v", err)
	}
	query.Permission = nil
	if err := ValidateQueryRequest(query); err == nil {
		t.Error("missing query permission accepted")
	}
	cycleErr := DetectCycle("ms→agent→ms")
	if !errors.Is(cycleErr, ErrCycleDetected) {
		t.Fatalf("cycle sentinel is not wrapped correctly: %v", cycleErr)
	}
}

func TestCallTopologyAndBoundaryReset(t *testing.T) {
	const id = "018f2f1e-7b5b-7b21-8f2e-4a6f3b1d9c11"
	base := NewCallContext(id, id, 2_000_000_000_000)
	if base.Route != "" || base.Hop != 0 {
		t.Fatalf("new call context did not reset route: %#v", base)
	}

	// Sender appends its node, receiver validates and appends its own node.
	msSent, err := AppendCallNode(base, NodeMessage)
	if err != nil {
		t.Fatalf("append ms: %v", err)
	}
	if err := ValidateAgentCallPath(msSent); err != nil {
		t.Fatalf("validate ms→agent: %v", err)
	}
	agentReceived, err := ValidateAndAdvanceAgentCallContext(msSent)
	if err != nil || agentReceived.Route != "ms→agent" || agentReceived.Hop != 2 {
		t.Fatalf("advance to agent: %#v, %v", agentReceived, err)
	}
	agentSent, err := AppendCallNode(agentReceived, NodeAgent)
	if err != nil || agentSent.Route != agentReceived.Route || agentSent.Hop != agentReceived.Hop {
		t.Fatalf("sender re-append changed route: %#v, %v", agentSent, err)
	}
	if err := ValidateProductCallPath(agentReceived); err != nil {
		t.Fatalf("validate agent→product: %v", err)
	}
	productReceived, err := ValidateAndAdvanceProductCallContext(agentReceived)
	if err != nil || productReceived.Route != "ms→agent→product" || productReceived.Hop != 3 {
		t.Fatalf("advance to product: %#v, %v", productReceived, err)
	}
	if _, err := ValidateAndAdvanceCallContext(productReceived, NodeMessage); err == nil {
		t.Fatal("product terminal route was allowed to forward")
	}

	if err := ValidateAgentCallPath(base); err == nil {
		t.Fatal("empty route was accepted at Agent peer boundary")
	}
	if err := ValidateProductCallPath(msSent); err == nil {
		t.Fatal("wrong peer route was accepted at Product boundary")
	}
	reset := NewCallContext(id, id, 2_000_000_000_000)
	if reset.Route != "" || reset.Hop != 0 {
		t.Fatalf("reset context retained old route: %#v", reset)
	}
}

func FuzzCanonicalizeJSON(f *testing.F) {
	for _, seed := range []string{`{}`, `{"a":1,"b":[true,null]}`, `1e-7`, `"hello"`, `{"\ud834\udd1e":1}`} {
		f.Add([]byte(seed))
	}
	f.Fuzz(func(t *testing.T, input []byte) {
		got, err := CanonicalizeJSON(input)
		if err != nil {
			return
		}
		again, err := CanonicalizeJSON(got)
		if err != nil {
			t.Fatalf("canonical output cannot be parsed: %v", err)
		}
		if !bytes.Equal(got, again) {
			t.Fatalf("canonicalization is not idempotent: %q != %q", got, again)
		}
	})
}

func FuzzValidateOperationID(f *testing.F) {
	for _, seed := range []string{"018f2f1e-7b5b-7b21-8f2e-4a6f3b1d9c11", "", "not-an-id"} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, input string) {
		_ = ValidateOperationID(input)
	})
}
