package capabilityv1

import (
	"bytes"
	"errors"
	"testing"
)

func TestOperationStateTransitions(t *testing.T) {
	valid := [][2]OperationState{
		{OperationState_OPERATION_STATE_UNSPECIFIED, OperationState_OPERATION_STATE_PENDING},
		{OperationState_OPERATION_STATE_PENDING, OperationState_OPERATION_STATE_RUNNING},
		{OperationState_OPERATION_STATE_PENDING, OperationState_OPERATION_STATE_CANCELLED},
		{OperationState_OPERATION_STATE_PENDING, OperationState_OPERATION_STATE_UNCERTAIN},
		{OperationState_OPERATION_STATE_RUNNING, OperationState_OPERATION_STATE_COMPLETED},
		{OperationState_OPERATION_STATE_RUNNING, OperationState_OPERATION_STATE_FAILED},
		{OperationState_OPERATION_STATE_RUNNING, OperationState_OPERATION_STATE_CANCELLED},
		{OperationState_OPERATION_STATE_RUNNING, OperationState_OPERATION_STATE_UNCERTAIN},
		{OperationState_OPERATION_STATE_UNCERTAIN, OperationState_OPERATION_STATE_COMPLETED},
		{OperationState_OPERATION_STATE_UNCERTAIN, OperationState_OPERATION_STATE_FAILED},
		{OperationState_OPERATION_STATE_UNCERTAIN, OperationState_OPERATION_STATE_CANCELLED},
	}
	for _, pair := range valid {
		if err := ValidateOperationStateTransition(pair[0], pair[1]); err != nil {
			t.Errorf("valid transition %s -> %s rejected: %v", pair[0], pair[1], err)
		}
	}
	invalid := [][2]OperationState{
		{OperationState_OPERATION_STATE_PENDING, OperationState_OPERATION_STATE_COMPLETED},
		{OperationState_OPERATION_STATE_UNCERTAIN, OperationState_OPERATION_STATE_RUNNING},
		{OperationState_OPERATION_STATE_COMPLETED, OperationState_OPERATION_STATE_FAILED},
		{OperationState_OPERATION_STATE_FAILED, OperationState_OPERATION_STATE_RUNNING},
	}
	for _, pair := range invalid {
		if err := ValidateOperationStateTransition(pair[0], pair[1]); err == nil {
			t.Errorf("invalid transition %s -> %s accepted", pair[0], pair[1])
		}
	}
	if err := ValidateOperationStateTransition(OperationState_OPERATION_STATE_COMPLETED, OperationState_OPERATION_STATE_COMPLETED); err != nil {
		t.Fatalf("terminal idempotent no-op rejected: %v", err)
	}
}

func TestOperationRestartReconcileAndTerminalFence(t *testing.T) {
	for _, state := range []OperationState{OperationState_OPERATION_STATE_PENDING, OperationState_OPERATION_STATE_RUNNING} {
		got, err := RecoverOperationStateAfterRestart(state)
		if err != nil || got != OperationState_OPERATION_STATE_UNCERTAIN {
			t.Errorf("restart %s did not fence to uncertain: %s, %v", state, got, err)
		}
	}
	for _, state := range []OperationState{OperationState_OPERATION_STATE_COMPLETED, OperationState_OPERATION_STATE_FAILED, OperationState_OPERATION_STATE_CANCELLED} {
		got, err := RecoverOperationStateAfterRestart(state)
		if err != nil || got != state {
			t.Errorf("terminal restart changed %s: %s, %v", state, got, err)
		}
	}
	if err := ValidateReconcileTransition(OperationState_OPERATION_STATE_UNCERTAIN, OperationState_OPERATION_STATE_COMPLETED); err != nil {
		t.Fatal(err)
	}
	if err := ValidateReconcileTransition(OperationState_OPERATION_STATE_RUNNING, OperationState_OPERATION_STATE_COMPLETED); err == nil {
		t.Fatal("running operation reconciled without uncertain fence")
	}
	digest := bytes.Repeat([]byte{0x55}, 32)
	if err := ValidateTerminalUpdate(OperationState_OPERATION_STATE_COMPLETED, OperationState_OPERATION_STATE_COMPLETED, digest, digest); err != nil {
		t.Fatalf("identical terminal receipt rejected: %v", err)
	}
	if !errors.Is(ValidateTerminalUpdate(OperationState_OPERATION_STATE_COMPLETED, OperationState_OPERATION_STATE_FAILED, digest, digest), ErrTerminalImmutable) {
		t.Fatal("terminal state mutation accepted")
	}
	if !errors.Is(ValidateTerminalUpdate(OperationState_OPERATION_STATE_COMPLETED, OperationState_OPERATION_STATE_COMPLETED, digest, bytes.Repeat([]byte{0x56}, 32)), ErrTerminalImmutable) {
		t.Fatal("terminal digest mutation accepted")
	}
}

func TestOperationTombstoneConflict(t *testing.T) {
	operationID := "018f2f1e-7b5b-7b21-8f2e-4a6f3b1d9c11"
	digest := bytes.Repeat([]byte{0x66}, 32)
	tombstone := OperationTombstone{OperationID: operationID, RequestDigest: digest, State: OperationState_OPERATION_STATE_FAILED, Revision: 8}
	if err := ValidateTombstoneConflict(tombstone, operationID, digest); err != nil {
		t.Fatalf("same tombstone retry rejected: %v", err)
	}
	if !errors.Is(ValidateTombstoneConflict(tombstone, operationID, bytes.Repeat([]byte{0x67}, 32)), ErrTombstoneConflict) {
		t.Fatal("same operation with different digest accepted")
	}
	if !errors.Is(ValidateTombstoneConflict(tombstone, "018f2f1e-7b5b-7b21-8f2e-4a6f3b1d9c12", digest), ErrTombstoneConflict) {
		t.Fatal("different operation ID accepted against tombstone")
	}
}

func TestRootTombstoneConflictAllowsOnlySameBusinessDigest(t *testing.T) {
	operationID := "018f2f1e-7b5b-7b21-8f2e-4a6f3b1d9c11"
	rootDigest := bytes.Repeat([]byte{0x71}, 32)
	tombstone := OperationTombstone{OperationID: operationID, RootRequestDigest: rootDigest, State: OperationState_OPERATION_STATE_COMPLETED, Revision: 9}
	if err := ValidateRootTombstoneConflict(tombstone, operationID, rootDigest); err != nil {
		t.Fatalf("same root digest replay rejected: %v", err)
	}
	if !errors.Is(ValidateRootTombstoneConflict(tombstone, operationID, bytes.Repeat([]byte{0x72}, 32)), ErrTombstoneConflict) {
		t.Fatal("different root digest accepted by tombstone fence")
	}
}

func TestStartReplayTombstoneRetainsFullPrincipalAndDescriptorFence(t *testing.T) {
	operationID := "018f2f1e-7b5b-7b21-8f2e-4a6f3b1d9c11"
	rootDigest := bytes.Repeat([]byte{0x81}, 32)
	key := StartReplayKey{OperationID: operationID, CapabilityID: "product.contacts.v1", Operation: "list", OwnerID: "owner-1", AccountGeneration: 4, RootRequestDigest: rootDigest}
	tombstone := OperationTombstone{OperationID: operationID, CapabilityID: key.CapabilityID, Operation: key.Operation, OwnerID: key.OwnerID, AccountGeneration: key.AccountGeneration, RootRequestDigest: rootDigest, State: OperationState_OPERATION_STATE_COMPLETED, Revision: 10}
	if err := ValidateStartReplayTombstoneConflict(tombstone, key); err != nil {
		t.Fatalf("same full replay key rejected: %v", err)
	}
	for name, mutate := range map[string]func(*StartReplayKey){
		"capability": func(k *StartReplayKey) { k.CapabilityID = "product.rooms.v1" },
		"operation":  func(k *StartReplayKey) { k.Operation = "write" },
		"owner":      func(k *StartReplayKey) { k.OwnerID = "owner-2" },
		"generation": func(k *StartReplayKey) { k.AccountGeneration++ },
		"digest":     func(k *StartReplayKey) { k.RootRequestDigest = bytes.Repeat([]byte{0x82}, 32) },
	} {
		candidate := key
		mutate(&candidate)
		if !errors.Is(ValidateStartReplayTombstoneConflict(tombstone, candidate), ErrTombstoneConflict) {
			t.Fatalf("%s mutation accepted", name)
		}
	}
}
