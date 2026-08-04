package capabilityv1

import (
	"bytes"
	"crypto/subtle"
	"errors"
	"fmt"
	"strings"

	"google.golang.org/protobuf/proto"
)

var (
	// ErrStartReplayConflict is returned when an operation UUID is replayed
	// with a different business/root request or principal binding. The final
	// request digest is deliberately not part of this fence because it binds a
	// short-lived grant that may be renewed for the same business request.
	ErrStartReplayConflict = errors.New("start operation replay conflict")
	// ErrOperationReceiptImmutable fences a replay from changing a durable
	// operation state, result, error, or revision that has already been
	// recorded. A control-grant refresh never changes this receipt.
	ErrOperationReceiptImmutable = errors.New("operation receipt is immutable")
	// ErrRootGrantRenewal is returned when a replacement root grant changes an
	// authenticated business or authorization field. Issuance time, expiry,
	// deadline, and transport chain may change on renewal.
	ErrRootGrantRenewal = errors.New("root grant renewal mismatch")
	// ErrControlGrantRefresh is returned when a StartOperation response carries
	// a malformed, unexpected, or incorrectly bound control-grant set.
	ErrControlGrantRefresh = errors.New("control grant refresh rejected")
)

// StartReplayKey is the durable business/idempotency fence for StartOperation.
// RootRequestDigest is the grant-independent canonical business digest. The
// final request digest is intentionally absent: it changes when message-server
// renews an otherwise equivalent short-lived root delegation.
type StartReplayKey struct {
	OperationID       string
	CapabilityID      string
	Operation         string
	OwnerID           string
	AccountGeneration int64
	RootRequestDigest []byte
}

// OperationReplayKey is an explicit alias for callers that name the fence
// independently of the StartOperation RPC.
type OperationReplayKey = StartReplayKey

// Validate checks the shape of a durable replay key.
func (k StartReplayKey) Validate() error {
	if err := ValidateOperationID(k.OperationID); err != nil {
		return fmt.Errorf("%w: operation_id: %v", ErrStartReplayConflict, err)
	}
	if err := validateReplayToken("capability_id", k.CapabilityID); err != nil {
		return fmt.Errorf("%w: %v", ErrStartReplayConflict, err)
	}
	if err := validateReplayToken("operation", k.Operation); err != nil {
		return fmt.Errorf("%w: %v", ErrStartReplayConflict, err)
	}
	if err := validateReplayToken("owner_id", k.OwnerID); err != nil {
		return fmt.Errorf("%w: %v", ErrStartReplayConflict, err)
	}
	if k.AccountGeneration <= 0 {
		return fmt.Errorf("%w: account_generation must be positive", ErrStartReplayConflict)
	}
	if err := ValidateSHA256Digest(k.RootRequestDigest); err != nil {
		return fmt.Errorf("%w: root_request_digest: %v", ErrStartReplayConflict, err)
	}
	return nil
}

// ValidateStartReplayKey applies the durable idempotency fence. Equal keys
// allow a replay and a control-grant refresh; every other change is a
// permanent conflict, including a different root/business digest.
func ValidateStartReplayKey(existing, requested StartReplayKey) error {
	if err := existing.Validate(); err != nil {
		return err
	}
	if err := requested.Validate(); err != nil {
		return err
	}
	if existing.OperationID != requested.OperationID ||
		existing.CapabilityID != requested.CapabilityID ||
		existing.Operation != requested.Operation ||
		existing.OwnerID != requested.OwnerID ||
		existing.AccountGeneration != requested.AccountGeneration ||
		subtle.ConstantTimeCompare(existing.RootRequestDigest, requested.RootRequestDigest) != 1 {
		return ErrStartReplayConflict
	}
	return nil
}

// ValidateOperationReplayKey is the operation-oriented spelling of
// ValidateStartReplayKey.
func ValidateOperationReplayKey(existing, requested OperationReplayKey) error {
	return ValidateStartReplayKey(existing, requested)
}

// OperationReplayReceipt is the durable business receipt returned by a
// StartOperation replay. Control grants are deliberately not represented:
// replacing them is an authorization refresh, not a receipt mutation.
type OperationReplayReceipt struct {
	Key      StartReplayKey
	State    OperationState
	Result   []byte
	Error    *CapabilityError
	Revision int64
}

// ValidateStartReplayReceipt confirms that a replay returns the exact durable
// business receipt. It permits a separate control-grant refresh while fencing
// terminal (and in-flight) state/result/error/revision mutation.
func ValidateStartReplayReceipt(existing, replay OperationReplayReceipt) error {
	if err := ValidateStartReplayKey(existing.Key, replay.Key); err != nil {
		return err
	}
	if existing.State != replay.State ||
		!bytes.Equal(existing.Result, replay.Result) ||
		existing.Revision != replay.Revision ||
		!proto.Equal(existing.Error, replay.Error) {
		return ErrOperationReceiptImmutable
	}
	return nil
}

// ValidateOperationReplayReceipt is the operation-oriented spelling of
// ValidateStartReplayReceipt.
func ValidateOperationReplayReceipt(existing, replay OperationReplayReceipt) error {
	return ValidateStartReplayReceipt(existing, replay)
}

// ValidateRootGrantRenewal verifies that a replacement root delegation
// describes the exact same business and authorization boundary. Both claims
// must be grant_kind=root and carry no Product child fields. A retry may have a new
// transport chain and new issued/expiry/deadline timestamps, but it may not
// change owner, generation, scopes, target descriptor, catalog/schema, entry
// boundary, hop budget, or RootRequestDigest. Signature and current-time
// validity must still be checked separately with VerifyAgentRootBinding or
// VerifyRootBinding; this helper never mints or authenticates a grant.
func ValidateRootGrantRenewal(previous, replacement GrantClaims) error {
	if err := ValidateGrantClaims(previous, DefaultGrantTTL); err != nil {
		return fmt.Errorf("%w: previous grant: %v", ErrRootGrantRenewal, err)
	}
	if err := ValidateGrantClaims(replacement, DefaultGrantTTL); err != nil {
		return fmt.Errorf("%w: replacement grant: %v", ErrRootGrantRenewal, err)
	}
	// Renewal is only for the message-server root delegation. A Product child
	// (or a root carrying Product-only claims) cannot be substituted into this
	// ledger even when its owner, scopes, and business digest happen to match.
	if previous.GrantKind != GrantKindRoot || replacement.GrantKind != GrantKindRoot ||
		previous.ChildOperationID != "" || replacement.ChildOperationID != "" ||
		previous.ProductTargetKind != "" || replacement.ProductTargetKind != "" {
		return ErrRootGrantRenewal
	}
	if previous.RootOperationID != replacement.RootOperationID ||
		previous.OwnerID != replacement.OwnerID ||
		previous.AccountGeneration != replacement.AccountGeneration ||
		!sameStringSlice(previous.Scopes, replacement.Scopes) ||
		previous.RootCapabilityID != replacement.RootCapabilityID ||
		previous.RootOperation != replacement.RootOperation ||
		previous.EntryRoute != replacement.EntryRoute ||
		previous.EntryHop != replacement.EntryHop ||
		previous.MaxHop != replacement.MaxHop ||
		previous.MaxRouteLength != replacement.MaxRouteLength ||
		subtle.ConstantTimeCompare(previous.RootRequestDigest, replacement.RootRequestDigest) != 1 ||
		subtle.ConstantTimeCompare(previous.CatalogDigest, replacement.CatalogDigest) != 1 ||
		subtle.ConstantTimeCompare(previous.SchemaDigest, replacement.SchemaDigest) != 1 {
		return ErrRootGrantRenewal
	}
	return nil
}

// ControlGrantEnvelopeBinding describes the exact operation and principal for
// a complete refresh set. RequiredActions must match the response one-for-one;
// this prevents a server from silently dropping or widening a lifecycle
// control permission during a replay.
type ControlGrantEnvelopeBinding struct {
	CallContext       *CallContext
	OwnerID           string
	AccountGeneration int64
	OperationID       string
	RequiredActions   []string
}

// OperationControlGrantEnvelopeBinding is an explicit long-form alias for
// callers that prefer the generated message name in their type declarations.
type OperationControlGrantEnvelopeBinding = ControlGrantEnvelopeBinding

// VerifyControlGrantEnvelopeSet validates and authenticates every control
// grant returned by a Product StartOperation replay. It selects the strict
// Agent or Product boundary from the actual CallContext route, never from an
// entry-route prefix supplied by the caller. The verifier only needs the
// Ed25519 public key; Agent processes never receive the signing private key.
func (c GrantCodec) VerifyControlGrantEnvelopeSet(envelopes []*OperationControlGrantEnvelope, key []byte, binding ControlGrantEnvelopeBinding) (map[string]OperationControlGrant, error) {
	if err := validateControlGrantEnvelopeBinding(binding); err != nil {
		return nil, err
	}
	c = c.normalized()
	if err := ValidateOperationControlGrantEnvelopes(envelopes, c.Now().UnixMilli()); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrControlGrantRefresh, err)
	}
	if len(envelopes) != len(binding.RequiredActions) {
		return nil, fmt.Errorf("%w: response actions do not match required actions", ErrControlGrantRefresh)
	}
	required := make(map[string]struct{}, len(binding.RequiredActions))
	for _, action := range binding.RequiredActions {
		if !isControlAction(action) {
			return nil, fmt.Errorf("%w: invalid required action %q", ErrControlGrantRefresh, action)
		}
		if _, exists := required[action]; exists {
			return nil, fmt.Errorf("%w: duplicate required action %q", ErrControlGrantRefresh, action)
		}
		required[action] = struct{}{}
	}
	claimsByAction := make(map[string]OperationControlGrant, len(envelopes))
	for _, envelope := range envelopes {
		if _, ok := required[envelope.Action]; !ok {
			return nil, fmt.Errorf("%w: unexpected action %q", ErrControlGrantRefresh, envelope.Action)
		}
		controlBinding := OperationControlGrantBinding{
			CallContext:       binding.CallContext,
			OwnerID:           binding.OwnerID,
			AccountGeneration: binding.AccountGeneration,
			OperationID:       binding.OperationID,
			ControlAction:     envelope.Action,
			ControlScope:      "operation:control:" + envelope.Action,
		}
		var claims OperationControlGrant
		var err error
		switch binding.CallContext.Route {
		case NodeMessage + RouteSeparator + NodeAgent:
			claims, err = c.VerifyOperationControlGrant(envelope.Grant, key, controlBinding)
		case NodeMessage + RouteSeparator + NodeAgent + RouteSeparator + NodeProduct,
			NodeAgent + RouteSeparator + NodeProduct:
			claims, err = c.VerifyProductOperationControlGrant(envelope.Grant, key, controlBinding)
		default:
			return nil, fmt.Errorf("%w: unsupported control boundary route %q", ErrControlGrantRefresh, binding.CallContext.Route)
		}
		if err != nil {
			return nil, fmt.Errorf("%w: %s grant: %v", ErrControlGrantRefresh, envelope.Action, err)
		}
		if claims.ExpiresAtUnixMs != envelope.ExpiresAtUnixMs {
			return nil, fmt.Errorf("%w: %s expiry metadata mismatch", ErrControlGrantRefresh, envelope.Action)
		}
		claimsByAction[envelope.Action] = claims
	}
	return claimsByAction, nil
}

// VerifyOperationControlGrantEnvelopes is the conventional exported spelling
// for VerifyControlGrantEnvelopeSet.
func (c GrantCodec) VerifyOperationControlGrantEnvelopes(envelopes []*OperationControlGrantEnvelope, key []byte, binding OperationControlGrantEnvelopeBinding) (map[string]OperationControlGrant, error) {
	return c.VerifyControlGrantEnvelopeSet(envelopes, key, binding)
}

func validateReplayToken(name, value string) error {
	if strings.TrimSpace(value) == "" || strings.TrimSpace(value) != value || len(value) > 256 {
		return fmt.Errorf("%s is invalid", name)
	}
	for _, r := range value {
		if r < 0x20 || r == '→' {
			return fmt.Errorf("%s contains a forbidden delimiter", name)
		}
	}
	return nil
}

func validateControlGrantEnvelopeBinding(binding ControlGrantEnvelopeBinding) error {
	if binding.CallContext == nil || binding.OwnerID == "" || binding.AccountGeneration <= 0 || binding.OperationID == "" || len(binding.RequiredActions) == 0 {
		return fmt.Errorf("%w: complete binding is required", ErrControlGrantRefresh)
	}
	if err := ValidateStrictCallContext(binding.CallContext); err != nil {
		return fmt.Errorf("%w: call context: %v", ErrControlGrantRefresh, err)
	}
	if err := ValidateOperationID(binding.OperationID); err != nil {
		return fmt.Errorf("%w: operation_id: %v", ErrControlGrantRefresh, err)
	}
	if err := validateReplayToken("owner_id", binding.OwnerID); err != nil {
		return fmt.Errorf("%w: %v", ErrControlGrantRefresh, err)
	}
	return nil
}

func sameStringSlice(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}
	return true
}
