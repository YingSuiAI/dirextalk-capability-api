package capabilityv1

import (
	"bytes"
	"crypto/subtle"
	"errors"
	"fmt"
)

var (
	ErrInvalidOperationTransition = errors.New("invalid operation state transition")
	ErrTerminalImmutable          = errors.New("terminal operation is immutable")
	ErrTombstoneConflict          = errors.New("operation tombstone conflict")
)

// OperationTombstone is the durable terminal receipt retained after an
// operation is compacted. Its embedded replay identity prevents re-executing
// the same UUID with a different descriptor, principal, generation, or
// business input after compaction.
type OperationTombstone struct {
	OperationID       string
	CapabilityID      string
	Operation         string
	OwnerID           string
	AccountGeneration int64
	// RequestDigest is retained for callers that store the final request
	// digest. New Start replay ledgers should use RootRequestDigest instead so
	// a renewed short-lived grant does not create a business conflict.
	RequestDigest []byte
	// RootRequestDigest is the grant-independent business/idempotency fence.
	RootRequestDigest []byte
	State             OperationState
	Revision          int64
}

func IsTerminalOperationState(state OperationState) bool {
	switch state {
	case OperationState_OPERATION_STATE_COMPLETED,
		OperationState_OPERATION_STATE_FAILED,
		OperationState_OPERATION_STATE_CANCELLED:
		return true
	default:
		return false
	}
}

// ValidateOperationStateTransition enforces the durable operation state
// machine. Repeating an identical terminal state is an idempotent no-op;
// changing a terminal state is never allowed.
func ValidateOperationStateTransition(from, to OperationState) error {
	if from == to && IsTerminalOperationState(from) {
		return nil
	}
	if IsTerminalOperationState(from) {
		return fmt.Errorf("%w: %s cannot transition to %s", ErrTerminalImmutable, from, to)
	}
	var allowed bool
	switch from {
	case OperationState_OPERATION_STATE_UNSPECIFIED:
		allowed = to == OperationState_OPERATION_STATE_PENDING
	case OperationState_OPERATION_STATE_PENDING:
		allowed = to == OperationState_OPERATION_STATE_RUNNING || to == OperationState_OPERATION_STATE_CANCELLED || to == OperationState_OPERATION_STATE_UNCERTAIN
	case OperationState_OPERATION_STATE_RUNNING:
		allowed = to == OperationState_OPERATION_STATE_COMPLETED || to == OperationState_OPERATION_STATE_FAILED || to == OperationState_OPERATION_STATE_CANCELLED || to == OperationState_OPERATION_STATE_UNCERTAIN
	case OperationState_OPERATION_STATE_UNCERTAIN:
		allowed = to == OperationState_OPERATION_STATE_COMPLETED || to == OperationState_OPERATION_STATE_FAILED || to == OperationState_OPERATION_STATE_CANCELLED
	}
	if !allowed {
		return fmt.Errorf("%w: %s -> %s", ErrInvalidOperationTransition, from, to)
	}
	return nil
}

// RecoverOperationStateAfterRestart fences in-flight work. Pending/running
// operations cannot be assumed to have had no side effect after a process
// restart, so they become uncertain and must be reconciled.
func RecoverOperationStateAfterRestart(state OperationState) (OperationState, error) {
	switch state {
	case OperationState_OPERATION_STATE_PENDING, OperationState_OPERATION_STATE_RUNNING:
		return OperationState_OPERATION_STATE_UNCERTAIN, nil
	case OperationState_OPERATION_STATE_UNSPECIFIED,
		OperationState_OPERATION_STATE_COMPLETED,
		OperationState_OPERATION_STATE_FAILED,
		OperationState_OPERATION_STATE_CANCELLED,
		OperationState_OPERATION_STATE_UNCERTAIN:
		return state, nil
	default:
		return OperationState_OPERATION_STATE_UNSPECIFIED, fmt.Errorf("%w: unknown state %d", ErrInvalidOperationTransition, state)
	}
}

// ValidateReconcileTransition limits reconciliation to an uncertain operation
// and a terminal result; it cannot restart or skip the side-effect fence.
func ValidateReconcileTransition(from, to OperationState) error {
	if from != OperationState_OPERATION_STATE_UNCERTAIN {
		return fmt.Errorf("%w: reconcile requires uncertain state", ErrInvalidOperationTransition)
	}
	if !IsTerminalOperationState(to) {
		return fmt.Errorf("%w: reconcile result must be terminal", ErrInvalidOperationTransition)
	}
	return nil
}

// ValidateTerminalUpdate enforces terminal immutability while allowing a
// duplicate receipt with the exact same final state and request digest.
func ValidateTerminalUpdate(previousState, nextState OperationState, previousDigest, nextDigest []byte) error {
	if !IsTerminalOperationState(previousState) {
		return ValidateOperationStateTransition(previousState, nextState)
	}
	if previousState != nextState || ValidateRequestDigest(previousDigest) != nil || ValidateRequestDigest(nextDigest) != nil || !bytes.Equal(previousDigest, nextDigest) {
		return ErrTerminalImmutable
	}
	return nil
}

// ValidateTombstoneConflict validates a retry against a compacted terminal
// receipt. Same ID + same digest is an idempotent hit; same ID + new digest is
// a permanent conflict and must never re-enter execution.
func ValidateTombstoneConflict(tombstone OperationTombstone, operationID string, requestDigest []byte) error {
	if err := ValidateOperationID(tombstone.OperationID); err != nil {
		return fmt.Errorf("%w: invalid operation ID", ErrTombstoneConflict)
	}
	if err := ValidateOperationID(operationID); err != nil {
		return fmt.Errorf("%w: invalid operation ID", ErrTombstoneConflict)
	}
	if !IsTerminalOperationState(tombstone.State) || ValidateRequestDigest(tombstone.RequestDigest) != nil || ValidateRequestDigest(requestDigest) != nil {
		return fmt.Errorf("%w: malformed tombstone or retry", ErrTombstoneConflict)
	}
	if tombstone.OperationID != operationID || !bytes.Equal(tombstone.RequestDigest, requestDigest) {
		return ErrTombstoneConflict
	}
	return nil
}

// ValidateRootTombstoneConflict applies the v1 replay fence to a compacted
// receipt. It compares the operation UUID and RootRequestDigest only; a later
// Start replay may carry a different final grant digest and still receive the
// immutable terminal receipt.
func ValidateRootTombstoneConflict(tombstone OperationTombstone, operationID string, rootRequestDigest []byte) error {
	if err := ValidateOperationID(tombstone.OperationID); err != nil {
		return fmt.Errorf("%w: invalid operation ID", ErrTombstoneConflict)
	}
	if err := ValidateOperationID(operationID); err != nil {
		return fmt.Errorf("%w: invalid operation ID", ErrTombstoneConflict)
	}
	if !IsTerminalOperationState(tombstone.State) || ValidateRequestDigest(tombstone.RootRequestDigest) != nil || ValidateRequestDigest(rootRequestDigest) != nil {
		return fmt.Errorf("%w: malformed root tombstone or retry", ErrTombstoneConflict)
	}
	if tombstone.OperationID != operationID || subtle.ConstantTimeCompare(tombstone.RootRequestDigest, rootRequestDigest) != 1 {
		return ErrTombstoneConflict
	}
	return nil
}

// ValidateStartReplayTombstoneConflict applies the complete StartReplayKey
// fence to a compacted terminal receipt. A UUID and root digest alone are not
// sufficient: capability/operation and authenticated principal changes must
// remain permanent conflicts after compaction.
func ValidateStartReplayTombstoneConflict(tombstone OperationTombstone, requested StartReplayKey) error {
	if err := ValidateOperationID(tombstone.OperationID); err != nil {
		return fmt.Errorf("%w: invalid operation ID", ErrTombstoneConflict)
	}
	if err := requested.Validate(); err != nil {
		return fmt.Errorf("%w: invalid replay key: %v", ErrTombstoneConflict, err)
	}
	if !IsTerminalOperationState(tombstone.State) || ValidateRequestDigest(tombstone.RootRequestDigest) != nil {
		return fmt.Errorf("%w: malformed replay tombstone", ErrTombstoneConflict)
	}
	stored := StartReplayKey{
		OperationID:       tombstone.OperationID,
		CapabilityID:      tombstone.CapabilityID,
		Operation:         tombstone.Operation,
		OwnerID:           tombstone.OwnerID,
		AccountGeneration: tombstone.AccountGeneration,
		RootRequestDigest: tombstone.RootRequestDigest,
	}
	if err := stored.Validate(); err != nil {
		return fmt.Errorf("%w: tombstone replay key is incomplete: %v", ErrTombstoneConflict, err)
	}
	if err := ValidateStartReplayKey(stored, requested); err != nil {
		return ErrTombstoneConflict
	}
	return nil
}

// ValidateOperationReplayTombstoneConflict is the operation-oriented alias.
func ValidateOperationReplayTombstoneConflict(tombstone OperationTombstone, requested OperationReplayKey) error {
	return ValidateStartReplayTombstoneConflict(tombstone, requested)
}
