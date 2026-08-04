package capabilityv1

import (
	"bytes"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"
)

const (
	testGrantChain = "018f2f1e-7b5b-7b21-8f2e-4a6f3b1d9c10"
	testGrantRoot  = "018f2f1e-7b5b-7b21-8f2e-4a6f3b1d9c11"
)

func testGrantKeys() (ed25519.PublicKey, ed25519.PrivateKey) {
	privateKey := ed25519.NewKeyFromSeed(bytes.Repeat([]byte{0x5a}, ed25519.SeedSize))
	return privateKey.Public().(ed25519.PublicKey), privateKey
}

func testGrantClaims() GrantClaims {
	return GrantClaims{
		ChainID:           testGrantChain,
		RootOperationID:   testGrantRoot,
		GrantKind:         GrantKindRoot,
		EntryRoute:        NodeMessage,
		EntryHop:          1,
		OwnerID:           "owner-123",
		AccountGeneration: 7,
		Scopes:            []string{"contacts:read", "messages:send"},
		RootCapabilityID:  "agent.chat.v1",
		RootOperation:     "chat",
		RootRequestDigest: bytes.Repeat([]byte{0x33}, sha256.Size),
		CatalogDigest:     bytes.Repeat([]byte{0x11}, sha256.Size),
		SchemaDigest:      bytes.Repeat([]byte{0x22}, sha256.Size),
		IssuedAtUnixMs:    1_700_000_000_000,
		ExpiresAtUnixMs:   1_700_000_600_000,
		MaxHop:            3,
		MaxRouteLength:    MaxRouteLength,
	}
}

func testAgentCallContext() *CallContext {
	return &CallContext{
		ChainId:         testGrantChain,
		RootOperationId: testGrantRoot,
		Hop:             1,
		Route:           NodeMessage,
		DeadlineUnixMs:  1_700_000_500_000,
	}
}

func testProductCallContext() *CallContext {
	return &CallContext{
		ChainId:         testGrantChain,
		RootOperationId: testGrantRoot,
		Hop:             3,
		Route:           NodeMessage + RouteSeparator + NodeAgent + RouteSeparator + NodeProduct,
		DeadlineUnixMs:  1_700_000_500_000,
	}
}

func testGrantBinding() GrantBinding {
	claims := testGrantClaims()
	return GrantBinding{
		CallContext:       testAgentCallContext(),
		OwnerID:           claims.OwnerID,
		AccountGeneration: claims.AccountGeneration,
		RootCapabilityID:  claims.RootCapabilityID,
		RootOperation:     claims.RootOperation,
		RootOperationID:   claims.RootOperationID,
		RootRequestDigest: claims.RootRequestDigest,
		CatalogDigest:     claims.CatalogDigest,
		SchemaDigest:      claims.SchemaDigest,
	}
}

func TestCapabilityGrantRoundTripAndDeterminism(t *testing.T) {
	publicKey, privateKey := testGrantKeys()
	codec := GrantCodec{Now: func() time.Time { return time.UnixMilli(1_700_000_100_000) }}
	claims := testGrantClaims()
	grant1, err := codec.Sign(claims, privateKey)
	if err != nil {
		t.Fatal(err)
	}
	grant2, err := codec.Sign(claims, privateKey)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(grant1, grant2) {
		t.Fatal("Ed25519 grant was not deterministic")
	}
	binding := testGrantBinding()
	verified, err := codec.Verify(grant1, publicKey, &binding)
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	if verified.OwnerID != claims.OwnerID || verified.ChainID != claims.ChainID || verified.EntryRoute != NodeMessage {
		t.Fatalf("verified claims mismatch: %#v", verified)
	}
}

func TestCapabilityGrantRejectsUnsafeAccountGeneration(t *testing.T) {
	_, privateKey := testGrantKeys()
	claims := testGrantClaims()
	claims.AccountGeneration = MaxAccountGeneration + 1
	if _, err := (GrantCodec{}).Sign(claims, privateKey); !errors.Is(err, ErrInvalidGrant) {
		t.Fatalf("unsafe account generation was signed: %v", err)
	}
}

func TestCapabilityGrantVerificationRequiresCompleteBinding(t *testing.T) {
	publicKey, privateKey := testGrantKeys()
	now := time.UnixMilli(1_700_000_100_000)
	grant, err := (GrantCodec{Now: func() time.Time { return now }}).Sign(testGrantClaims(), privateKey)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := (GrantCodec{Now: func() time.Time { return now }}).Verify(grant, publicKey, nil); !errors.Is(err, ErrGrantBinding) {
		t.Fatalf("nil binding accepted: %v", err)
	}
	partial := &GrantBinding{OwnerID: testGrantClaims().OwnerID}
	if _, err := (GrantCodec{Now: func() time.Time { return now }}).Verify(grant, publicKey, partial); !errors.Is(err, ErrGrantBinding) {
		t.Fatalf("partial binding accepted: %v", err)
	}
	if _, err := VerifyCapabilityGrant(grant, publicKey, now, nil); !errors.Is(err, ErrGrantBinding) {
		t.Fatalf("functional verifier accepted nil binding: %v", err)
	}
}

func TestCapabilityGrantVerifierCannotMint(t *testing.T) {
	publicKey, privateKey := testGrantKeys()
	if _, err := (GrantCodec{}).Sign(testGrantClaims(), publicKey); !errors.Is(err, ErrGrantKey) {
		t.Fatalf("public verifier key minted a grant: %v", err)
	}
	grant, err := (GrantCodec{}).Sign(testGrantClaims(), privateKey)
	if err != nil {
		t.Fatal(err)
	}
	binding := testGrantBinding()
	if _, err := (GrantCodec{Now: func() time.Time { return time.UnixMilli(1_700_000_100_000) }}).Verify(grant, privateKey, &binding); !errors.Is(err, ErrGrantKey) {
		t.Fatalf("private signing key accepted as verifier material: %v", err)
	}
}

func TestGrantKeyRawAndPEMCodecs(t *testing.T) {
	publicKey, privateKey := testGrantKeys()
	privatePEM, err := MarshalGrantPrivateKeyPEM(privateKey)
	if err != nil {
		t.Fatal(err)
	}
	publicPEM, err := MarshalGrantPublicKeyPEM(publicKey)
	if err != nil {
		t.Fatal(err)
	}
	parsedPrivate, err := ParseGrantPrivateKey(privatePEM)
	if err != nil || !bytes.Equal(parsedPrivate, privateKey) {
		t.Fatalf("private PEM round trip failed: %v", err)
	}
	parsedPublic, err := ParseGrantPublicKey(publicPEM)
	if err != nil || !bytes.Equal(parsedPublic, publicKey) {
		t.Fatalf("public PEM round trip failed: %v", err)
	}
	for _, malformed := range [][]byte{bytes.Repeat([]byte{1}, 31), bytes.Repeat([]byte{1}, 33), []byte("-----BEGIN PUBLIC KEY-----\ninvalid\n-----END PUBLIC KEY-----\n")} {
		if _, err := ParseGrantPrivateKey(malformed); !errors.Is(err, ErrGrantKey) {
			t.Errorf("malformed private key accepted: %q", malformed)
		}
		if _, err := ParseGrantPublicKey(malformed); !errors.Is(err, ErrGrantKey) {
			t.Errorf("malformed public key accepted: %q", malformed)
		}
	}
}

func TestCapabilityGrantExpiryAndClockSkew(t *testing.T) {
	publicKey, privateKey := testGrantKeys()
	claims := testGrantClaims()
	codec := GrantCodec{Now: func() time.Time { return time.UnixMilli(claims.IssuedAtUnixMs) }}
	grant, err := codec.Sign(claims, privateKey)
	if err != nil {
		t.Fatal(err)
	}
	codec.Now = func() time.Time { return time.UnixMilli(claims.ExpiresAtUnixMs) }
	if _, err := codec.verifySigned(grant, publicKey); !errors.Is(err, ErrGrantExpired) {
		t.Fatalf("expiry error = %v", err)
	}
	codec.Now = func() time.Time {
		return time.UnixMilli(claims.IssuedAtUnixMs - DefaultGrantClockSkew.Milliseconds() - 1)
	}
	if _, err := codec.verifySigned(grant, publicKey); !errors.Is(err, ErrGrantNotYetValid) {
		t.Fatalf("not-yet-valid error = %v", err)
	}
}

func TestCapabilityGrantExactBindingAndTamperRejection(t *testing.T) {
	publicKey, privateKey := testGrantKeys()
	codec := GrantCodec{Now: func() time.Time { return time.UnixMilli(1_700_000_100_000) }}
	grant, err := codec.Sign(testGrantClaims(), privateKey)
	if err != nil {
		t.Fatal(err)
	}
	for name, binding := range map[string]GrantBinding{
		"empty":            {},
		"wrong owner":      func() GrantBinding { b := testGrantBinding(); b.OwnerID = "other"; return b }(),
		"wrong operation":  func() GrantBinding { b := testGrantBinding(); b.RootOperation = "send"; return b }(),
		"wrong capability": func() GrantBinding { b := testGrantBinding(); b.RootCapabilityID = "product.rooms.v1"; return b }(),
		"wrong operation id": func() GrantBinding {
			b := testGrantBinding()
			b.RootOperationID = "018f2f1e-7b5b-7b21-8f2e-4a6f3b1d9c12"
			return b
		}(),
		"wrong request digest": func() GrantBinding {
			b := testGrantBinding()
			b.RootRequestDigest = bytes.Repeat([]byte{0x44}, sha256.Size)
			return b
		}(),
		"wrong chain": func() GrantBinding {
			b := testGrantBinding()
			b.CallContext = cloneCallContext(b.CallContext)
			b.CallContext.ChainId = "018f2f1e-7b5b-7b21-8f2e-4a6f3b1d9c12"
			return b
		}(),
		"route reset": func() GrantBinding {
			b := testGrantBinding()
			b.CallContext = &CallContext{ChainId: testGrantChain, RootOperationId: testGrantRoot, Hop: 1, Route: NodeAgent, DeadlineUnixMs: 1_700_000_500_000}
			return b
		}(),
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := codec.Verify(grant, publicKey, &binding); !errors.Is(err, ErrGrantBinding) {
				t.Fatalf("binding error = %v", err)
			}
		})
	}

	tampered := append([]byte(nil), grant...)
	tampered[len(tampered)-1] ^= 1
	binding := testGrantBinding()
	if _, err := codec.Verify(tampered, publicKey, &binding); !errors.Is(err, ErrInvalidGrant) {
		t.Fatalf("tampered grant error = %v", err)
	}
	otherPrivate := ed25519.NewKeyFromSeed(bytes.Repeat([]byte{0x33}, ed25519.SeedSize))
	otherPublic := otherPrivate.Public().(ed25519.PublicKey)
	if _, err := codec.Verify(grant, otherPublic, &binding); !errors.Is(err, ErrInvalidGrant) {
		t.Fatalf("wrong public key error = %v", err)
	}
	for _, malformed := range [][]byte{nil, []byte("grant-v1.bad"), []byte("grant-v2.a.b"), []byte("grant-v1.a.b.c"), append(grant, '=')} {
		if _, err := codec.Verify(malformed, publicKey, &binding); err == nil {
			t.Errorf("malformed grant %q accepted", malformed)
		}
	}
}

func TestCapabilityGrantClaimsValidation(t *testing.T) {
	_, privateKey := testGrantKeys()
	base := testGrantClaims()
	cases := map[string]func(*GrantClaims){
		"missing chain":        func(c *GrantClaims) { c.ChainID = "" },
		"negative generation":  func(c *GrantClaims) { c.AccountGeneration = -1 },
		"scope not sorted":     func(c *GrantClaims) { c.Scopes = []string{"messages:send", "contacts:read"} },
		"scope duplicate":      func(c *GrantClaims) { c.Scopes = []string{"contacts:read", "contacts:read"} },
		"delimiter":            func(c *GrantClaims) { c.RootOperation = "send→part" },
		"negative max hop":     func(c *GrantClaims) { c.MaxHop = -1 },
		"ttl too long":         func(c *GrantClaims) { c.ExpiresAtUnixMs = c.IssuedAtUnixMs + DefaultGrantTTL.Milliseconds() + 1 },
		"bad catalog digest":   func(c *GrantClaims) { c.CatalogDigest = []byte("short") },
		"bad request digest":   func(c *GrantClaims) { c.RootRequestDigest = []byte("short") },
		"bad root operation":   func(c *GrantClaims) { c.RootOperationID = "not-a-uuid" },
		"nested entry route":   func(c *GrantClaims) { c.EntryRoute = "ms→agent"; c.EntryHop = 2 },
		"entry exceeds limits": func(c *GrantClaims) { c.MaxRouteLength = 1 },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			claims := base
			mutate(&claims)
			if _, err := (GrantCodec{}).Sign(claims, privateKey); err == nil {
				t.Fatal("invalid claims accepted")
			}
		})
	}
	if _, err := (GrantCodec{}).Sign(base, privateKey[:len(privateKey)-1]); !errors.Is(err, ErrGrantKey) {
		t.Fatalf("invalid private key error = %v", err)
	}
}

func TestCapabilityGrantUnknownFieldsAndInvalidNumbers(t *testing.T) {
	publicKey, privateKey := testGrantKeys()
	claims := testGrantClaims()
	payload, err := marshalGrantClaims(claims)
	if err != nil {
		t.Fatal(err)
	}
	decoder := json.NewDecoder(strings.NewReader(string(payload)))
	decoder.UseNumber()
	var object map[string]any
	if err := decoder.Decode(&object); err != nil {
		t.Fatal(err)
	}
	object["extra"] = true
	payload, err = Canonicalize(object)
	if err != nil {
		t.Fatal(err)
	}
	signature := ed25519.Sign(privateKey, payload)
	grant := []byte(GrantPrefix + "." + base64.RawURLEncoding.EncodeToString(payload) + "." + base64.RawURLEncoding.EncodeToString(signature))
	codec := GrantCodec{Now: func() time.Time { return time.UnixMilli(claims.IssuedAtUnixMs) }}
	binding := testGrantBinding()
	if _, err := codec.Verify(grant, publicKey, &binding); err == nil {
		t.Fatal("unknown grant field accepted")
	}

	validPayload, err := marshalGrantClaims(claims)
	if err != nil {
		t.Fatal(err)
	}
	invalidPayload := bytes.Replace(validPayload, []byte(`"account_generation":7`), []byte(`"account_generation":1.5`), 1)
	signature = ed25519.Sign(privateKey, invalidPayload)
	invalidGrant := []byte(GrantPrefix + "." + base64.RawURLEncoding.EncodeToString(invalidPayload) + "." + base64.RawURLEncoding.EncodeToString(signature))
	if _, err := codec.Verify(invalidGrant, publicKey, &binding); err == nil {
		t.Fatal("non-integral account generation accepted")
	}
}

func TestCapabilityGrantNestedDelegationUsesAuthenticatedRoutePrefix(t *testing.T) {
	publicKey, privateKey := testGrantKeys()
	claims := testGrantClaims()
	codec := GrantCodec{Now: func() time.Time { return time.UnixMilli(1_700_000_100_000) }}
	grant, err := codec.Sign(claims, privateKey)
	if err != nil {
		t.Fatal(err)
	}
	binding := RootGrantBinding{
		CallContext:       testProductCallContext(),
		RootOperationID:   claims.RootOperationID,
		OwnerID:           claims.OwnerID,
		AccountGeneration: claims.AccountGeneration,
		RootRequestDigest: claims.RootRequestDigest,
		RequiredScopes:    []string{"messages:send"},
	}
	verified, err := VerifyRootBinding(grant, publicKey, time.UnixMilli(1_700_000_100_000), binding)
	if err != nil {
		t.Fatalf("nested root binding rejected: %v", err)
	}
	if verified.RootCapabilityID != "agent.chat.v1" || verified.RootOperation != "chat" {
		t.Fatalf("root claims changed during nested verification: %#v", verified)
	}

	for name, mutate := range map[string]func(*RootGrantBinding){
		"empty":     func(b *RootGrantBinding) { *b = RootGrantBinding{} },
		"truncated": func(b *RootGrantBinding) { b.CallContext = testAgentCallContext() },
		"reset to agent": func(b *RootGrantBinding) {
			b.CallContext = &CallContext{ChainId: testGrantChain, RootOperationId: testGrantRoot, Hop: 1, Route: NodeAgent, DeadlineUnixMs: 1_700_000_700_000}
		},
		"wrong chain": func(b *RootGrantBinding) {
			b.CallContext = cloneCallContext(b.CallContext)
			b.CallContext.ChainId = "018f2f1e-7b5b-7b21-8f2e-4a6f3b1d9c12"
		},
		"missing scope": func(b *RootGrantBinding) { b.RequiredScopes = []string{"rooms:write"} },
	} {
		t.Run(name, func(t *testing.T) {
			candidate := binding
			candidate.CallContext = cloneCallContext(binding.CallContext)
			mutate(&candidate)
			if _, err := VerifyRootBinding(grant, publicKey, time.UnixMilli(1_700_000_100_000), candidate); !errors.Is(err, ErrGrantBinding) {
				t.Fatalf("invalid nested binding accepted: %v", err)
			}
		})
	}
	deadlineTooLate := binding
	deadlineTooLate.CallContext = cloneCallContext(binding.CallContext)
	deadlineTooLate.CallContext.DeadlineUnixMs = claims.ExpiresAtUnixMs + 1
	if _, err := VerifyRootBinding(grant, publicKey, time.UnixMilli(1_700_000_100_000), deadlineTooLate); !errors.Is(err, ErrGrantBinding) {
		t.Fatalf("call deadline beyond grant expiry accepted: %v", err)
	}
}

func TestAgentRootBindingRequiresExactCallContext(t *testing.T) {
	publicKey, privateKey := testGrantKeys()
	claims := testGrantClaims()
	grant, err := (GrantCodec{Now: func() time.Time { return time.UnixMilli(1_700_000_100_000) }}).Sign(claims, privateKey)
	if err != nil {
		t.Fatal(err)
	}
	call := &CallContext{ChainId: testGrantChain, RootOperationId: testGrantRoot, Hop: 2, Route: NodeMessage + RouteSeparator + NodeAgent, DeadlineUnixMs: 1_700_000_500_000}
	binding := AgentGrantBinding{CallContext: call, RootOperationID: testGrantRoot, RootCapabilityID: claims.RootCapabilityID, RootOperation: claims.RootOperation, OwnerID: claims.OwnerID, AccountGeneration: claims.AccountGeneration, RootRequestDigest: claims.RootRequestDigest, CatalogDigest: claims.CatalogDigest, SchemaDigest: claims.SchemaDigest, RequiredScopes: []string{"contacts:read"}}
	if _, err := VerifyAgentRootBinding(grant, publicKey, time.UnixMilli(1_700_000_100_000), binding); err != nil {
		t.Fatalf("exact Agent binding rejected: %v", err)
	}
	for _, route := range []string{"ms", "agent", "ms→agent→product"} {
		candidate := binding
		candidate.CallContext = cloneCallContext(call)
		candidate.CallContext.Route = route
		candidate.CallContext.Hop = int32(len(strings.Split(route, RouteSeparator)))
		if _, err := VerifyAgentRootBinding(grant, publicKey, time.UnixMilli(1_700_000_100_000), candidate); !errors.Is(err, ErrGrantBinding) {
			t.Errorf("route %q accepted at Agent boundary: %v", route, err)
		}
	}
	wrongDescriptor := binding
	wrongDescriptor.RootOperation = "send"
	if _, err := VerifyAgentQueryGrant(grant, publicKey, time.UnixMilli(1_700_000_100_000), wrongDescriptor); !errors.Is(err, ErrGrantBinding) {
		t.Fatalf("Agent Query descriptor mismatch accepted: %v", err)
	}
}

func TestOperationControlGrantDomainAndExactBinding(t *testing.T) {
	publicKey, privateKey := testGrantKeys()
	now := time.UnixMilli(1_700_000_100_000)
	claims := OperationControlGrant{
		ChainID:           testGrantChain,
		OwnerID:           "owner-123",
		AccountGeneration: 7,
		OperationID:       testGrantRoot,
		ControlAction:     "cancel",
		ControlScope:      "operation:control:cancel",
		EntryRoute:        NodeMessage,
		EntryHop:          1,
		DeadlineUnixMs:    1_700_000_150_000,
		IssuedAtUnixMs:    1_700_000_100_000,
		ExpiresAtUnixMs:   1_700_000_160_000,
	}
	codec := GrantCodec{Now: func() time.Time { return now }}
	grant, err := codec.SignOperationControlGrant(claims, privateKey)
	if err != nil {
		t.Fatal(err)
	}
	ctx := &CallContext{ChainId: testGrantChain, RootOperationId: testGrantRoot, Hop: 2, Route: NodeMessage + RouteSeparator + NodeAgent, DeadlineUnixMs: claims.DeadlineUnixMs}
	binding := OperationControlGrantBinding{CallContext: ctx, OwnerID: claims.OwnerID, AccountGeneration: claims.AccountGeneration, OperationID: claims.OperationID, ControlAction: claims.ControlAction, ControlScope: claims.ControlScope}
	verified, err := codec.VerifyOperationControlGrant(grant, publicKey, binding)
	if err != nil || verified.OperationID != claims.OperationID {
		t.Fatalf("control grant verification failed: %#v, %v", verified, err)
	}
	codec.Now = func() time.Time { return time.UnixMilli(claims.ExpiresAtUnixMs) }
	if _, err := codec.VerifyOperationControlGrant(grant, publicKey, binding); !errors.Is(err, ErrControlGrantExpired) {
		t.Fatalf("expired control grant accepted: %v", err)
	}
	codec.Now = func() time.Time { return now }
	stripped := binding
	stripped.CallContext = cloneCallContext(ctx)
	stripped.CallContext.Route = NodeMessage
	stripped.CallContext.Hop = 1
	if _, err := codec.VerifyOperationControlGrant(grant, publicKey, stripped); !errors.Is(err, ErrGrantBinding) {
		t.Fatalf("entry context accepted instead of actual Agent route: %v", err)
	}
	rootBinding := testGrantBinding()
	if _, err := codec.Verify(grant, publicKey, &rootBinding); !errors.Is(err, ErrInvalidGrant) {
		t.Fatalf("control grant accepted as root grant: %v", err)
	}
	rootGrant, err := codec.Sign(testGrantClaims(), privateKey)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := codec.VerifyOperationControlGrant(rootGrant, publicKey, binding); err == nil {
		t.Fatal("root grant accepted as control grant")
	}
	for name, mutate := range map[string]func(*OperationControlGrantBinding){
		"owner":      func(b *OperationControlGrantBinding) { b.OwnerID = "other" },
		"generation": func(b *OperationControlGrantBinding) { b.AccountGeneration = 8 },
		"operation":  func(b *OperationControlGrantBinding) { b.OperationID = "018f2f1e-7b5b-7b21-8f2e-4a6f3b1d9c12" },
		"action":     func(b *OperationControlGrantBinding) { b.ControlAction = "get" },
		"scope":      func(b *OperationControlGrantBinding) { b.ControlScope = "operation:control:get" },
	} {
		t.Run(name, func(t *testing.T) {
			candidate := binding
			mutate(&candidate)
			if _, err := codec.VerifyOperationControlGrant(grant, publicKey, candidate); !errors.Is(err, ErrGrantBinding) {
				t.Fatalf("cross-boundary binding accepted: %v", err)
			}
		})
	}
}

func TestOperationControlGrantActionsExpiryAndShortTTL(t *testing.T) {
	_, privateKey := testGrantKeys()
	claims := OperationControlGrant{ChainID: testGrantChain, OwnerID: "owner-123", AccountGeneration: 7, OperationID: testGrantRoot, ControlAction: "watch", ControlScope: "operation:control:watch", EntryRoute: NodeAgent, EntryHop: 1, DeadlineUnixMs: 1_700_000_150_000, IssuedAtUnixMs: 1_700_000_100_000, ExpiresAtUnixMs: 1_700_000_160_000}
	if _, err := (GrantCodec{Now: func() time.Time { return time.UnixMilli(claims.IssuedAtUnixMs) }}).SignOperationControlGrant(claims, privateKey); err != nil {
		t.Fatalf("short control grant should be signed: %v", err)
	}
	for _, action := range []string{"get", "watch", "cancel", "reconcile"} {
		candidate := claims
		candidate.ControlAction = action
		candidate.ControlScope = "operation:control:" + action
		if _, err := (GrantCodec{}).SignOperationControlGrant(candidate, privateKey); err != nil {
			t.Errorf("action %s rejected: %v", action, err)
		}
	}
	for _, action := range []string{"send", "", "CANCEL"} {
		candidate := claims
		candidate.ControlAction = action
		candidate.ControlScope = "operation:control:" + action
		if _, err := (GrantCodec{}).SignOperationControlGrant(candidate, privateKey); err == nil {
			t.Errorf("invalid action %q accepted", action)
		}
	}
	long := claims
	long.ExpiresAtUnixMs = long.IssuedAtUnixMs + DefaultControlGrantTTL.Milliseconds() + 1
	if _, err := (GrantCodec{}).SignOperationControlGrant(long, privateKey); err == nil {
		t.Fatal("long control TTL accepted")
	}
}

func TestProductOperationControlGrantUsesActualProductRoute(t *testing.T) {
	publicKey, privateKey := testGrantKeys()
	claims := OperationControlGrant{ChainID: testGrantChain, OwnerID: "owner-123", AccountGeneration: 7, OperationID: testGrantRoot, ControlAction: "get", ControlScope: "operation:control:get", EntryRoute: NodeMessage, EntryHop: 1, DeadlineUnixMs: 1_700_000_150_000, IssuedAtUnixMs: 1_700_000_100_000, ExpiresAtUnixMs: 1_700_000_160_000}
	now := time.UnixMilli(1_700_000_100_000)
	grant, err := (GrantCodec{Now: func() time.Time { return now }}).SignOperationControlGrant(claims, privateKey)
	if err != nil {
		t.Fatal(err)
	}
	ctx := &CallContext{ChainId: testGrantChain, RootOperationId: testGrantRoot, Hop: 3, Route: NodeMessage + RouteSeparator + NodeAgent + RouteSeparator + NodeProduct, DeadlineUnixMs: claims.DeadlineUnixMs}
	binding := OperationControlGrantBinding{CallContext: ctx, OwnerID: claims.OwnerID, AccountGeneration: claims.AccountGeneration, OperationID: claims.OperationID, ControlAction: claims.ControlAction, ControlScope: claims.ControlScope}
	if _, err := (GrantCodec{Now: func() time.Time { return now }}).VerifyProductOperationControlGrant(grant, publicKey, binding); err != nil {
		t.Fatalf("product control route rejected: %v", err)
	}
}
