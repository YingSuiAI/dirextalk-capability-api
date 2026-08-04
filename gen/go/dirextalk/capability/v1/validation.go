package capabilityv1

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
)

const MaxOperationControlGrantEnvelopes = 4

// ValidateOperationControlGrantEnvelopes validates optional control grants
// returned by Product StartOperation responses. The list is bounded, actions
// are unique, grants remain opaque but domain-separated, and expiry is short
// and in the future relative to nowUnixMs.
func ValidateOperationControlGrantEnvelopes(envelopes []*OperationControlGrantEnvelope, nowUnixMs int64) error {
	if len(envelopes) == 0 {
		return nil
	}
	if len(envelopes) > MaxOperationControlGrantEnvelopes {
		return fmt.Errorf("control_grants exceeds maximum %d", MaxOperationControlGrantEnvelopes)
	}
	if nowUnixMs <= 0 {
		return errors.New("now_unix_ms must be positive")
	}
	seen := make(map[string]struct{}, len(envelopes))
	for _, envelope := range envelopes {
		if envelope == nil {
			return errors.New("control_grant envelope must not be nil")
		}
		if !isControlAction(envelope.Action) {
			return fmt.Errorf("invalid control grant action %q", envelope.Action)
		}
		if _, exists := seen[envelope.Action]; exists {
			return fmt.Errorf("duplicate control grant action %q", envelope.Action)
		}
		seen[envelope.Action] = struct{}{}
		if len(envelope.Grant) == 0 || len(envelope.Grant) > DefaultGrantMaxSize || !bytes.HasPrefix(envelope.Grant, []byte(ControlGrantPrefix+".")) {
			return fmt.Errorf("control grant for %s is invalid or too large", envelope.Action)
		}
		if envelope.ExpiresAtUnixMs <= nowUnixMs || envelope.ExpiresAtUnixMs-nowUnixMs > DefaultControlGrantTTL.Milliseconds() {
			return fmt.Errorf("control grant for %s has invalid expiry", envelope.Action)
		}
	}
	return nil
}

func isControlAction(action string) bool {
	switch action {
	case "get", "watch", "cancel", "reconcile":
		return true
	default:
		return false
	}
}

// ValidateOperationID validates the canonical UUID representation used for
// idempotency keys. UUID versions are intentionally not restricted; deployments
// may use v4 random IDs or a centrally generated v7 ID.
func ValidateOperationID(operationID string) error {
	if len(operationID) != 36 || strings.ToLower(operationID) != operationID {
		return fmt.Errorf("operation_id must be a lowercase canonical UUID")
	}
	for i, c := range operationID {
		switch i {
		case 8, 13, 18, 23:
			if c != '-' {
				return fmt.Errorf("operation_id has an invalid UUID separator")
			}
		default:
			if c == '-' {
				return fmt.Errorf("operation_id has an invalid UUID separator")
			}
		}
	}
	compact := strings.ReplaceAll(operationID, "-", "")
	decoded, err := hex.DecodeString(compact)
	if err != nil || len(decoded) != 16 {
		return fmt.Errorf("operation_id must be a UUID: %w", err)
	}
	if decoded[0] == 0 && decoded[1] == 0 && decoded[2] == 0 && decoded[3] == 0 &&
		decoded[4] == 0 && decoded[5] == 0 && decoded[6] == 0 && decoded[7] == 0 &&
		decoded[8] == 0 && decoded[9] == 0 && decoded[10] == 0 && decoded[11] == 0 &&
		decoded[12] == 0 && decoded[13] == 0 && decoded[14] == 0 && decoded[15] == 0 {
		return errors.New("operation_id must not be the nil UUID")
	}
	return nil
}

// ValidateOperationRequest validates the common StartOperation shape. It does
// not recompute the request digest because capability version/schema/grant
// inputs are negotiated outside the protobuf request. Services must call
// ComputeRequestDigest with their catalog and VerifyRequestDigest separately.
func ValidateOperationRequest(req *StartOperationRequest) error {
	if req == nil {
		return errors.New("start_operation request is required")
	}
	if err := ValidateOperationID(req.OperationId); err != nil {
		return err
	}
	if strings.TrimSpace(req.CapabilityId) == "" {
		return errors.New("capability_id is required")
	}
	if strings.TrimSpace(req.Operation) == "" {
		return errors.New("operation is required")
	}
	if err := ValidateRequestDigest(req.RequestDigest); err != nil {
		return err
	}
	if req.ExpectedRevision < 0 {
		return errors.New("expected_revision must be non-negative")
	}
	if err := ValidatePermissionContext(req.Permission); err != nil {
		return fmt.Errorf("permission: %w", err)
	}
	canonical, err := CanonicalizeJSON(req.RequestJson)
	if err != nil {
		return fmt.Errorf("request_json: %w", err)
	}
	if string(canonical) != string(req.RequestJson) {
		return errors.New("request_json must be RFC 8785 canonical JSON")
	}
	if err := ValidateStrictCallContext(req.CallContext); err != nil {
		return fmt.Errorf("call_context: %w", err)
	}
	return nil
}

// ValidateRootOperationRequest validates a message-server -> Agent root
// mutation/stream request. Product child operations deliberately use
// ValidateOperationRequest because their operation ID differs from the signed
// root operation ID.
func ValidateRootOperationRequest(req *StartOperationRequest) error {
	if err := ValidateOperationRequest(req); err != nil {
		return err
	}
	if req.CallContext.RootOperationId != req.OperationId {
		return errors.New("root operation_id must equal call_context.root_operation_id")
	}
	return nil
}

// ValidateQueryRequest validates a Query request shape. Query operation_id is a
// capability operation name in the current wire contract, so it is checked as
// a non-empty identifier rather than as an idempotency UUID.
func ValidateQueryRequest(req *QueryRequest) error {
	if req == nil {
		return errors.New("query request is required")
	}
	if strings.TrimSpace(req.CapabilityId) == "" || strings.TrimSpace(req.OperationId) == "" {
		return errors.New("capability_id and operation_id are required")
	}
	if err := ValidatePermissionContext(req.Permission); err != nil {
		return fmt.Errorf("permission: %w", err)
	}
	canonical, err := CanonicalizeJSON(req.RequestJson)
	if err != nil {
		return fmt.Errorf("request_json: %w", err)
	}
	if string(canonical) != string(req.RequestJson) {
		return errors.New("request_json must be RFC 8785 canonical JSON")
	}
	if err := ValidateStrictCallContext(req.CallContext); err != nil {
		return fmt.Errorf("call_context: %w", err)
	}
	return nil
}

// VerifyRequestDigest performs the comparison step after a service has
// recomputed the expected digest from its capability catalog and grant state.
func VerifyRequestDigest(actual, expected []byte) error {
	if err := ValidateRequestDigest(actual); err != nil {
		return fmt.Errorf("actual request digest: %w", err)
	}
	if err := ValidateRequestDigest(expected); err != nil {
		return fmt.Errorf("expected request digest: %w", err)
	}
	return ValidateDigest(actual, expected)
}

// ValidatePermissionContext checks the shape of a server-issued grant. It is
// not a substitute for verifying the grant signature, owner binding, scopes,
// or account generation at the message-server boundary.
func ValidatePermissionContext(permission *PermissionContext) error {
	if permission == nil {
		return errors.New("permission context is required")
	}
	if strings.TrimSpace(permission.AuthenticatedOwnerId) == "" {
		return errors.New("authenticated_owner_id is required")
	}
	if len(permission.CapabilityGrant) == 0 {
		return errors.New("capability_grant is required")
	}
	if permission.AccountGeneration <= 0 {
		return errors.New("account_generation must be positive")
	}
	if len(permission.RootRequestDigest) > 0 {
		if err := ValidateSHA256Digest(permission.RootRequestDigest); err != nil {
			return fmt.Errorf("root_request_digest: %w", err)
		}
	}
	if len(permission.GrantedScopes) == 0 || !isSortedUnique(permission.GrantedScopes) {
		return errors.New("granted_scopes must be non-empty, sorted, and unique")
	}
	return nil
}
