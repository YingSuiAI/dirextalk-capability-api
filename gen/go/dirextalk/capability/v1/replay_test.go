package capabilityv1

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"testing"
	"time"
)

func replayKeyFromClaims(claims GrantClaims) StartReplayKey {
	return StartReplayKey{
		OperationID:       claims.RootOperationID,
		CapabilityID:      claims.RootCapabilityID,
		Operation:         claims.RootOperation,
		OwnerID:           claims.OwnerID,
		AccountGeneration: claims.AccountGeneration,
		RootRequestDigest: append([]byte(nil), claims.RootRequestDigest...),
	}
}

func TestStartReplayKeyUsesRootDigestNotFinalGrantDigest(t *testing.T) {
	claims := testGrantClaims()
	existing := replayKeyFromClaims(claims)
	requested := existing
	if err := ValidateStartReplayKey(existing, requested); err != nil {
		t.Fatalf("same root replay rejected: %v", err)
	}
	requested.RootRequestDigest = bytes.Repeat([]byte{0x44}, sha256.Size)
	if !errors.Is(ValidateStartReplayKey(existing, requested), ErrStartReplayConflict) {
		t.Fatal("different root request digest accepted")
	}
	requested = existing
	requested.OwnerID = "other-owner"
	if !errors.Is(ValidateOperationReplayKey(existing, requested), ErrStartReplayConflict) {
		t.Fatal("different owner accepted for root replay")
	}
}

func TestStartReplayKeyRejectsUnsafeAccountGeneration(t *testing.T) {
	valid := replayKeyFromClaims(testGrantClaims())
	valid.AccountGeneration = MaxAccountGeneration
	if err := valid.Validate(); err != nil {
		t.Fatalf("maximum safe account generation rejected: %v", err)
	}

	unsafe := valid
	unsafe.AccountGeneration = MaxAccountGeneration + 1
	if !errors.Is(unsafe.Validate(), ErrStartReplayConflict) {
		t.Fatal("unsafe account generation accepted by replay key validation")
	}
	if !errors.Is(ValidateStartReplayKey(valid, unsafe), ErrStartReplayConflict) {
		t.Fatal("unsafe account generation crossed replay comparison fence")
	}
}

func TestStartReplayReceiptRemainsImmutableWhileControlsRefresh(t *testing.T) {
	claims := testGrantClaims()
	key := replayKeyFromClaims(claims)
	existing := OperationReplayReceipt{
		Key: key, State: OperationState_OPERATION_STATE_COMPLETED,
		Result: []byte(`{"ok":true}`), Revision: 8,
	}
	if err := ValidateStartReplayReceipt(existing, existing); err != nil {
		t.Fatalf("identical terminal receipt rejected: %v", err)
	}
	changed := existing
	changed.Result = []byte(`{"ok":false}`)
	if !errors.Is(ValidateStartReplayReceipt(existing, changed), ErrOperationReceiptImmutable) {
		t.Fatal("terminal receipt result changed during control refresh")
	}
	changed = existing
	changed.State = OperationState_OPERATION_STATE_FAILED
	if !errors.Is(ValidateOperationReplayReceipt(existing, changed), ErrOperationReceiptImmutable) {
		t.Fatal("terminal receipt state changed during control refresh")
	}
	changed = existing
	changed.Key.RootRequestDigest = bytes.Repeat([]byte{0x44}, sha256.Size)
	if !errors.Is(ValidateStartReplayReceipt(existing, changed), ErrStartReplayConflict) {
		t.Fatal("different root digest crossed receipt fence")
	}
}

func TestRootGrantRenewalKeepsExactAuthorizationBoundary(t *testing.T) {
	previous := testGrantClaims()
	previous.EntryDeadlineUnixMs = previous.ExpiresAtUnixMs
	replacement := previous
	replacement.ChainID = "018f2f1e-7b5b-7b21-8f2e-4a6f3b1d9c12"
	replacement.IssuedAtUnixMs += time.Minute.Milliseconds()
	replacement.ExpiresAtUnixMs += time.Minute.Milliseconds()
	replacement.EntryDeadlineUnixMs += time.Minute.Milliseconds()
	if err := ValidateRootGrantRenewal(previous, replacement); err != nil {
		t.Fatalf("equivalent renewed root grant rejected: %v", err)
	}
	_, privateKey := testGrantKeys()
	codec := GrantCodec{Now: func() time.Time { return time.UnixMilli(previous.IssuedAtUnixMs) }}
	previousGrant, err := codec.Sign(previous, privateKey)
	if err != nil {
		t.Fatalf("sign previous root grant: %v", err)
	}
	replacementGrant, err := codec.Sign(replacement, privateKey)
	if err != nil {
		t.Fatalf("sign replacement root grant: %v", err)
	}
	if bytes.Equal(previousGrant, replacementGrant) {
		t.Fatal("renewed root grant unexpectedly reused identical signature/time payload")
	}
	grantDigest1 := sha256.Sum256(previousGrant)
	grantDigest2 := sha256.Sum256(replacementGrant)
	requestDigest1, err := ComputeRequestDigest(1, previous.RootCapabilityID, "1.0.0", previous.SchemaDigest, previous.RootOperation, 0, map[string]interface{}{"message": "hello"}, nil, grantDigest1[:])
	if err != nil {
		t.Fatal(err)
	}
	requestDigest2, err := ComputeRequestDigest(1, replacement.RootCapabilityID, "1.0.0", replacement.SchemaDigest, replacement.RootOperation, 0, map[string]interface{}{"message": "hello"}, nil, grantDigest2[:])
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(requestDigest1, requestDigest2) {
		t.Fatal("final request digest did not remain bound to the renewed grant")
	}
	for name, mutate := range map[string]func(*GrantClaims){
		"root digest": func(c *GrantClaims) { c.RootRequestDigest = bytes.Repeat([]byte{0x44}, sha256.Size) },
		"owner":       func(c *GrantClaims) { c.OwnerID = "other-owner" },
		"generation":  func(c *GrantClaims) { c.AccountGeneration++ },
		"operation":   func(c *GrantClaims) { c.RootOperation = "other" },
		"scopes":      func(c *GrantClaims) { c.Scopes = []string{"contacts:read"} },
		"schema":      func(c *GrantClaims) { c.SchemaDigest = bytes.Repeat([]byte{0x44}, sha256.Size) },
		"product kind": func(c *GrantClaims) {
			c.GrantKind = GrantKindProductChild
			c.ProductTargetKind = ProductTargetStart
			c.ChildOperationID = "018f2f1e-7b5b-7b21-8f2e-4a6f3b1d9c12"
		},
	} {
		t.Run(name, func(t *testing.T) {
			candidate := replacement
			mutate(&candidate)
			if !errors.Is(ValidateRootGrantRenewal(previous, candidate), ErrRootGrantRenewal) {
				t.Fatal("changed root authorization accepted")
			}
		})
	}
	productPrevious := previous
	productPrevious.GrantKind = GrantKindProductChild
	productPrevious.ProductTargetKind = ProductTargetStart
	productPrevious.ChildOperationID = "018f2f1e-7b5b-7b21-8f2e-4a6f3b1d9c12"
	productReplacement := productPrevious
	productReplacement.IssuedAtUnixMs += time.Minute.Milliseconds()
	productReplacement.ExpiresAtUnixMs += time.Minute.Milliseconds()
	productReplacement.EntryDeadlineUnixMs += time.Minute.Milliseconds()
	if !errors.Is(ValidateRootGrantRenewal(productPrevious, productReplacement), ErrRootGrantRenewal) {
		t.Fatal("Product child grant substituted into root renewal ledger")
	}
}

func TestVerifyControlGrantEnvelopeSetBindsRefreshExactly(t *testing.T) {
	publicKey, privateKey := testGrantKeys()
	now := time.UnixMilli(1_700_000_100_000)
	codec := GrantCodec{Now: func() time.Time { return now }}
	callContext := &CallContext{
		ChainId: testGrantChain, RootOperationId: testGrantRoot,
		Hop: 2, Route: NodeMessage + RouteSeparator + NodeAgent,
		DeadlineUnixMs: now.Add(time.Minute).UnixMilli(),
	}
	envelopes := make([]*OperationControlGrantEnvelope, 0, 2)
	for _, action := range []string{"get", "watch"} {
		claims := OperationControlGrant{
			ChainID: testGrantChain, OwnerID: "owner-123", AccountGeneration: 7,
			OperationID: testGrantRoot, ControlAction: action,
			ControlScope: "operation:control:" + action,
			EntryRoute:   NodeMessage, EntryHop: 1,
			DeadlineUnixMs: now.Add(time.Minute).UnixMilli(),
			IssuedAtUnixMs: now.UnixMilli(), ExpiresAtUnixMs: now.Add(90 * time.Second).UnixMilli(),
		}
		grant, err := codec.SignOperationControlGrant(claims, privateKey)
		if err != nil {
			t.Fatalf("sign %s grant: %v", action, err)
		}
		envelopes = append(envelopes, &OperationControlGrantEnvelope{Action: action, Grant: grant, ExpiresAtUnixMs: claims.ExpiresAtUnixMs})
	}
	binding := ControlGrantEnvelopeBinding{
		CallContext: callContext, OwnerID: "owner-123", AccountGeneration: 7,
		OperationID: testGrantRoot, RequiredActions: []string{"get", "watch"},
	}
	claimsByAction, err := codec.VerifyControlGrantEnvelopeSet(envelopes, publicKey, binding)
	if err != nil || len(claimsByAction) != 2 {
		t.Fatalf("valid exact control refresh rejected: claims=%#v err=%v", claimsByAction, err)
	}
	if claimsByAction["watch"].OwnerID != binding.OwnerID || claimsByAction["watch"].AccountGeneration != binding.AccountGeneration || claimsByAction["watch"].OperationID != binding.OperationID {
		t.Fatal("control refresh claims lost owner/generation/operation binding")
	}
	productBinding := binding
	productBinding.CallContext = &CallContext{
		ChainId: testGrantChain, RootOperationId: testGrantRoot,
		Hop: 3, Route: NodeMessage + RouteSeparator + NodeAgent + RouteSeparator + NodeProduct,
		DeadlineUnixMs: callContext.DeadlineUnixMs,
	}
	if _, err := codec.VerifyControlGrantEnvelopeSet(envelopes, publicKey, productBinding); err != nil {
		t.Fatalf("same exact controls rejected at Product boundary: %v", err)
	}
	if _, err := codec.VerifyOperationControlGrantEnvelopes(envelopes[:1], publicKey, binding); !errors.Is(err, ErrControlGrantRefresh) {
		t.Fatal("missing control action accepted")
	}
	mutated := append([]*OperationControlGrantEnvelope(nil), envelopes...)
	mutated[0] = &OperationControlGrantEnvelope{Action: "get", Grant: envelopes[0].Grant, ExpiresAtUnixMs: envelopes[0].ExpiresAtUnixMs + 1}
	if _, err := codec.VerifyControlGrantEnvelopeSet(mutated, publicKey, binding); !errors.Is(err, ErrControlGrantRefresh) {
		t.Fatal("tampered control expiry metadata accepted")
	}
	wrongOperation := append([]*OperationControlGrantEnvelope(nil), envelopes...)
	wrongClaims := OperationControlGrant{
		ChainID: testGrantChain, OwnerID: "owner-123", AccountGeneration: 7,
		OperationID: "018f2f1e-7b5b-7b21-8f2e-4a6f3b1d9c12", ControlAction: "get",
		ControlScope: "operation:control:get", EntryRoute: NodeMessage, EntryHop: 1,
		DeadlineUnixMs: now.Add(time.Minute).UnixMilli(), IssuedAtUnixMs: now.UnixMilli(), ExpiresAtUnixMs: now.Add(90 * time.Second).UnixMilli(),
	}
	wrongGrant, err := codec.SignOperationControlGrant(wrongClaims, privateKey)
	if err != nil {
		t.Fatal(err)
	}
	wrongOperation[0] = &OperationControlGrantEnvelope{Action: "get", Grant: wrongGrant, ExpiresAtUnixMs: wrongClaims.ExpiresAtUnixMs}
	if _, err := codec.VerifyControlGrantEnvelopeSet(wrongOperation, publicKey, binding); !errors.Is(err, ErrControlGrantRefresh) {
		t.Fatal("wrong operation control grant accepted")
	}
}
