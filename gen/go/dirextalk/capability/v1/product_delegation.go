package capabilityv1

import (
	"bytes"
	"crypto/subtle"
	"errors"
	"fmt"
	"strings"
	"time"
)

const DefaultProductDelegationTTL = 2 * time.Minute

var (
	ErrProductDelegationBinding  = errors.New("product delegation binding mismatch")
	ErrProductDelegationExchange = errors.New("product delegation exchange rejected")
)

// ProductGrantBinding is the exact Product boundary binding for the standard
// grant-v1 child delegation signed by message-server. The child grant keeps
// the parent chain/root operation, owner, generation and scopes, while its
// RootCapabilityID/RootOperation/RootRequestDigest are the exact Product
// target descriptor and grant-independent Product request digest.
type ProductGrantBinding struct {
	CallContext       *CallContext
	RootOperationID   string
	ChildOperationID  string
	OwnerID           string
	AccountGeneration int64
	RequiredScopes    []string
	CapabilityID      string
	Operation         string
	TargetKind        ExchangeProductTargetKind
	RootRequestDigest []byte
	CatalogDigest     []byte
	SchemaDigest      []byte
}

// SignProductDelegationGrant is an explicit name for the MS-only child-grant
// signing path. It deliberately delegates to the existing standard grant-v1
// codec so Agent/Product verifier boundaries remain domain-compatible while
// exact Product descriptor binding is enforced by VerifyProductDelegationGrant.
func (c GrantCodec) SignProductDelegationGrant(claims GrantClaims, key []byte) ([]byte, error) {
	c = c.normalized()
	if c.MaxTTL > DefaultProductDelegationTTL {
		c.MaxTTL = DefaultProductDelegationTTL
	}
	claims.GrantKind = GrantKindProductChild
	if claims.ProductTargetKind != ProductTargetQuery && claims.ProductTargetKind != ProductTargetStart {
		return nil, fmt.Errorf("%w: Product target kind is required", ErrProductDelegationBinding)
	}
	if claims.ProductTargetKind == ProductTargetStart {
		if err := ValidateOperationID(claims.ChildOperationID); err != nil {
			return nil, fmt.Errorf("%w: child_operation_id: %v", ErrProductDelegationBinding, err)
		}
	} else if claims.ChildOperationID != "" {
		return nil, fmt.Errorf("%w: Query cannot carry child_operation_id", ErrProductDelegationBinding)
	}
	return c.Sign(claims, key)
}

// ProductDelegationIssue is the only supported input for deriving a Product
// child grant from an already verified Agent parent. Parent identity and
// authorization are copied from ParentClaims; callers can only narrow the
// Product scopes (the child carries exactly RequiredScopes) and target a
// concrete nested operation.
type ProductDelegationIssue struct {
	ParentClaims      GrantClaims
	CallContext       *CallContext
	ChildOperationID  string
	CapabilityID      string
	Operation         string
	RequiredScopes    []string
	TargetKind        ExchangeProductTargetKind
	RootRequestDigest []byte
	CatalogDigest     []byte
	SchemaDigest      []byte
}

// SignProductDelegationFromParent derives and signs the standard grant-v1
// Product child. It refuses arbitrary caller-supplied owner/generation/chain/
// root/scopes fields: those values come from the verified parent and required
// Product scopes must be a subset of the parent grant.
func (c GrantCodec) SignProductDelegationFromParent(issue ProductDelegationIssue, key []byte) ([]byte, error) {
	parent := issue.ParentClaims
	if err := ValidateGrantClaims(parent, DefaultGrantTTL); err != nil {
		return nil, fmt.Errorf("%w: parent claims: %v", ErrProductDelegationBinding, err)
	}
	if parent.GrantKind != GrantKindRoot || parent.ChildOperationID != "" {
		return nil, fmt.Errorf("%w: parent must be a root grant", ErrProductDelegationBinding)
	}
	if parent.EntryRoute != NodeMessage || parent.EntryHop != 1 {
		return nil, fmt.Errorf("%w: parent must enter from ms", ErrProductDelegationBinding)
	}
	if issue.CallContext == nil {
		return nil, fmt.Errorf("%w: call context is required", ErrProductDelegationBinding)
	}
	if err := ValidateStrictCallContext(issue.CallContext); err != nil {
		return nil, fmt.Errorf("%w: call context: %v", ErrProductDelegationBinding, err)
	}
	if issue.CallContext.ChainId != parent.ChainID || issue.CallContext.RootOperationId != parent.RootOperationID {
		return nil, fmt.Errorf("%w: parent chain/root mismatch", ErrProductDelegationBinding)
	}
	if issue.CallContext.Route != NodeMessage+RouteSeparator+NodeAgent+RouteSeparator+NodeProduct && issue.CallContext.Route != NodeAgent+RouteSeparator+NodeProduct {
		return nil, fmt.Errorf("%w: Product route is required", ErrProductDelegationBinding)
	}
	if issue.TargetKind != ExchangeProductTargetKind_EXCHANGE_PRODUCT_TARGET_KIND_QUERY && issue.TargetKind != ExchangeProductTargetKind_EXCHANGE_PRODUCT_TARGET_KIND_START_OPERATION {
		return nil, fmt.Errorf("%w: Product target kind is required", ErrProductDelegationBinding)
	}
	if issue.TargetKind == ExchangeProductTargetKind_EXCHANGE_PRODUCT_TARGET_KIND_START_OPERATION {
		if err := ValidateOperationID(issue.ChildOperationID); err != nil {
			return nil, fmt.Errorf("%w: child_operation_id: %v", ErrProductDelegationBinding, err)
		}
	} else if issue.ChildOperationID != "" {
		return nil, fmt.Errorf("%w: Query cannot carry child_operation_id", ErrProductDelegationBinding)
	}
	if err := validateReplayToken("capability_id", issue.CapabilityID); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrProductDelegationBinding, err)
	}
	if err := validateReplayToken("operation", issue.Operation); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrProductDelegationBinding, err)
	}
	if len(issue.RequiredScopes) == 0 || !isSortedUnique(issue.RequiredScopes) || !scopesSubset(issue.RequiredScopes, parent.Scopes) {
		return nil, fmt.Errorf("%w: Product scopes must be a sorted subset of parent scopes", ErrProductDelegationBinding)
	}
	if len(issue.RootRequestDigest) != 32 || len(issue.CatalogDigest) != 32 || len(issue.SchemaDigest) != 32 {
		return nil, fmt.Errorf("%w: Product digests must be 32 bytes", ErrProductDelegationBinding)
	}
	entryRoute := NodeMessage
	entryHop := int32(1)
	if strings.HasPrefix(issue.CallContext.Route, NodeAgent+RouteSeparator) {
		entryRoute = NodeAgent
	}
	codec := c.normalized()
	if codec.MaxTTL > DefaultProductDelegationTTL {
		codec.MaxTTL = DefaultProductDelegationTTL
	}
	return codec.SignProductDelegationGrant(GrantClaims{
		ChainID:           parent.ChainID,
		RootOperationID:   parent.RootOperationID,
		ChildOperationID:  issue.ChildOperationID,
		GrantKind:         GrantKindProductChild,
		ProductTargetKind: productTargetKindString(issue.TargetKind),
		EntryRoute:        entryRoute,
		EntryHop:          entryHop,
		OwnerID:           parent.OwnerID,
		AccountGeneration: parent.AccountGeneration,
		// Narrow the child authority to exactly the Product operation scopes.
		// Parent-only scopes (for example agent:product:execute) stay at the
		// broker boundary and are never copied into ProductPermission.
		Scopes:              append([]string(nil), issue.RequiredScopes...),
		RootCapabilityID:    issue.CapabilityID,
		RootOperation:       issue.Operation,
		RootRequestDigest:   append([]byte(nil), issue.RootRequestDigest...),
		CatalogDigest:       append([]byte(nil), issue.CatalogDigest...),
		SchemaDigest:        append([]byte(nil), issue.SchemaDigest...),
		MaxHop:              parent.MaxHop,
		MaxRouteLength:      parent.MaxRouteLength,
		EntryDeadlineUnixMs: issue.CallContext.DeadlineUnixMs,
	}, key)
}

// VerifyProductDelegationGrant verifies a standard grant-v1 child delegation
// at the Product boundary. It rejects an Agent-root grant whose descriptor is
// agent.* because Product capability/operation/catalog/schema are bound here.
func VerifyProductDelegationGrant(grant, key []byte, now time.Time, binding ProductGrantBinding) (GrantClaims, error) {
	if binding.CallContext == nil || binding.RootOperationID == "" || binding.OwnerID == "" || binding.AccountGeneration <= 0 || len(binding.RequiredScopes) == 0 || binding.CapabilityID == "" || binding.Operation == "" || len(binding.RootRequestDigest) != 32 || len(binding.CatalogDigest) != 32 || len(binding.SchemaDigest) != 32 {
		return GrantClaims{}, fmt.Errorf("%w: complete binding is required", ErrProductDelegationBinding)
	}
	if binding.TargetKind != ExchangeProductTargetKind_EXCHANGE_PRODUCT_TARGET_KIND_QUERY && binding.TargetKind != ExchangeProductTargetKind_EXCHANGE_PRODUCT_TARGET_KIND_START_OPERATION {
		return GrantClaims{}, fmt.Errorf("%w: Product target kind is required", ErrProductDelegationBinding)
	}
	if binding.TargetKind == ExchangeProductTargetKind_EXCHANGE_PRODUCT_TARGET_KIND_START_OPERATION {
		if err := ValidateOperationID(binding.ChildOperationID); err != nil {
			return GrantClaims{}, fmt.Errorf("%w: child_operation_id: %v", ErrProductDelegationBinding, err)
		}
	} else if binding.ChildOperationID != "" {
		return GrantClaims{}, fmt.Errorf("%w: Query cannot carry child_operation_id", ErrProductDelegationBinding)
	}
	claims, err := (GrantCodec{Now: func() time.Time { return now }}).verifySigned(grant, key)
	if err != nil {
		return GrantClaims{}, err
	}
	if claims.GrantKind != GrantKindProductChild || claims.ProductTargetKind != productTargetKindString(binding.TargetKind) || claims.ChildOperationID != binding.ChildOperationID {
		return GrantClaims{}, fmt.Errorf("%w: child operation or grant kind", ErrProductDelegationBinding)
	}
	if claims.ExpiresAtUnixMs-claims.IssuedAtUnixMs > DefaultProductDelegationTTL.Milliseconds() {
		return GrantClaims{}, fmt.Errorf("%w: Product delegation TTL exceeds %s", ErrProductDelegationBinding, DefaultProductDelegationTTL)
	}
	if err := validateRootBinding(claims, binding.CallContext, binding.RootOperationID, binding.OwnerID, binding.AccountGeneration, binding.RootRequestDigest, binding.RequiredScopes, NodeProduct); err != nil {
		return GrantClaims{}, fmt.Errorf("%w: root: %v", ErrProductDelegationBinding, err)
	}
	if err := validateAgentGrantDescriptor(claims, binding.CapabilityID, binding.Operation, binding.CatalogDigest, binding.SchemaDigest); err != nil {
		return GrantClaims{}, fmt.Errorf("%w: product descriptor: %v", ErrProductDelegationBinding, err)
	}
	return claims, nil
}

// VerifyProductCapabilityGrant is the concise boundary spelling used by
// Product handlers. It is intentionally an alias-like wrapper, not a weaker
// generic root verifier.
func VerifyProductCapabilityGrant(grant, key []byte, now time.Time, binding ProductGrantBinding) (GrantClaims, error) {
	return VerifyProductDelegationGrant(grant, key, now, binding)
}

// VerifyProductDelegationParent verifies the already-authenticated Agent
// parent at the Product broker boundary before a child grant is signed. It
// intentionally binds chain/root operation, owner, generation, parent root
// digest, scopes and the actual advanced Product route, but does not treat the
// parent Agent descriptor as the Product target; the child verifier performs
// that exact descriptor binding after exchange.
func VerifyProductDelegationParent(grant, key []byte, now time.Time, binding RootGrantBinding) (GrantClaims, error) {
	return VerifyRootBinding(grant, key, now, binding)
}

// ValidateProductDelegationExchangeRequest validates the private broker RPC
// before message-server verifies and exchanges the parent grant. It accepts
// the sender-side route (ms→agent or agent) and the receiver-advanced route so
// callers may run it either before or after the Product interceptor advances
// CallContext; both forms are still structurally bounded and cycle-checked.
func ValidateProductDelegationExchangeRequest(req *ExchangeProductDelegationRequest) error {
	if req == nil {
		return fmt.Errorf("%w: request is required", ErrProductDelegationExchange)
	}
	if err := ValidateStrictCallContext(req.CallContext); err != nil {
		return fmt.Errorf("%w: call_context: %v", ErrProductDelegationExchange, err)
	}
	if !isProductDelegationExchangeRoute(req.CallContext.Route, req.CallContext.Hop) {
		return fmt.Errorf("%w: call_context route is not an Agent/Product exchange boundary", ErrProductDelegationExchange)
	}
	switch req.TargetKind {
	case ExchangeProductTargetKind_EXCHANGE_PRODUCT_TARGET_KIND_QUERY:
		if req.ChildOperationId != "" {
			return fmt.Errorf("%w: Query cannot carry child_operation_id", ErrProductDelegationExchange)
		}
		if err := validateReplayToken("operation", req.Operation); err != nil {
			return fmt.Errorf("%w: %v", ErrProductDelegationExchange, err)
		}
	case ExchangeProductTargetKind_EXCHANGE_PRODUCT_TARGET_KIND_START_OPERATION:
		if err := ValidateOperationID(req.ChildOperationId); err != nil {
			return fmt.Errorf("%w: child operation_id: %v", ErrProductDelegationExchange, err)
		}
	default:
		return fmt.Errorf("%w: target_kind must be QUERY or START_OPERATION", ErrProductDelegationExchange)
	}
	if err := validateReplayToken("capability_id", req.CapabilityId); err != nil {
		return fmt.Errorf("%w: %v", ErrProductDelegationExchange, err)
	}
	if err := validateReplayToken("operation", req.Operation); err != nil {
		return fmt.Errorf("%w: %v", ErrProductDelegationExchange, err)
	}
	if req.ExpectedRevision < 0 {
		return fmt.Errorf("%w: expected_revision must be non-negative", ErrProductDelegationExchange)
	}
	canonical, err := CanonicalizeJSON(req.RequestJson)
	if err != nil || !bytes.Equal(canonical, req.RequestJson) {
		return fmt.Errorf("%w: request_json must be RFC 8785 canonical JSON", ErrProductDelegationExchange)
	}
	if err := ValidatePermissionContext(req.ParentPermission); err != nil {
		return fmt.Errorf("%w: parent permission: %v", ErrProductDelegationExchange, err)
	}
	if len(req.ParentPermission.RootRequestDigest) != 32 {
		return fmt.Errorf("%w: parent root_request_digest must be 32 bytes", ErrProductDelegationExchange)
	}
	if !bytes.HasPrefix(req.ParentPermission.CapabilityGrant, []byte(GrantPrefix+".")) {
		return fmt.Errorf("%w: parent grant must be grant-v1", ErrProductDelegationExchange)
	}
	return nil
}

// ValidateProductDelegationExchangeResponse validates the opaque child
// PermissionContext returned by the broker. Signature and exact descriptor
// claims are checked by VerifyProductDelegationGrant at the Product operation
// boundary; this shape check prevents malformed/long-lived response metadata
// from entering the Agent cache.
func ValidateProductDelegationExchangeResponse(resp *ExchangeProductDelegationResponse, nowUnixMs int64) error {
	if resp == nil || resp.ProductPermission == nil {
		return fmt.Errorf("%w: product permission is required", ErrProductDelegationExchange)
	}
	if nowUnixMs <= 0 {
		return fmt.Errorf("%w: validation clock must be positive", ErrProductDelegationExchange)
	}
	if err := ValidatePermissionContext(resp.ProductPermission); err != nil {
		return fmt.Errorf("%w: product permission: %v", ErrProductDelegationExchange, err)
	}
	if !bytes.HasPrefix(resp.ProductPermission.CapabilityGrant, []byte(GrantPrefix+".")) {
		return fmt.Errorf("%w: product permission must carry grant-v1", ErrProductDelegationExchange)
	}
	if len(resp.ProductPermission.RootRequestDigest) != 32 {
		return fmt.Errorf("%w: product root_request_digest must be 32 bytes", ErrProductDelegationExchange)
	}
	if resp.ExpiresAtUnixMs <= nowUnixMs || resp.ExpiresAtUnixMs-nowUnixMs > DefaultProductDelegationTTL.Milliseconds() {
		return fmt.Errorf("%w: product delegation expiry is invalid", ErrProductDelegationExchange)
	}
	return nil
}

// ValidateParentPermissionRootDigest checks the authenticated parent grant's
// digest field before the broker calls VerifyRootBinding. It is separate from
// ValidatePermissionContext so a service cannot accidentally treat an absent
// digest as an unchecked parent delegation.
func ValidateParentPermissionRootDigest(permission *PermissionContext) error {
	if permission == nil {
		return fmt.Errorf("%w: parent permission is required", ErrProductDelegationExchange)
	}
	if err := ValidatePermissionContext(permission); err != nil {
		return fmt.Errorf("%w: permission: %v", ErrProductDelegationExchange, err)
	}
	if len(permission.RootRequestDigest) != 32 {
		return fmt.Errorf("%w: root_request_digest must be 32 bytes", ErrProductDelegationExchange)
	}
	if !bytes.HasPrefix(permission.CapabilityGrant, []byte(GrantPrefix+".")) {
		return fmt.Errorf("%w: root grant must use grant-v1", ErrProductDelegationExchange)
	}
	return nil
}

func isProductDelegationExchangeRoute(route string, hop int32) bool {
	switch route {
	case NodeAgent:
		return hop == 1
	case NodeMessage + RouteSeparator + NodeAgent:
		return hop == 2
	case NodeAgent + RouteSeparator + NodeProduct:
		return hop == 2
	case NodeMessage + RouteSeparator + NodeAgent + RouteSeparator + NodeProduct:
		return hop == 3
	default:
		return false
	}
}

func productTargetKindString(kind ExchangeProductTargetKind) string {
	switch kind {
	case ExchangeProductTargetKind_EXCHANGE_PRODUCT_TARGET_KIND_QUERY:
		return ProductTargetQuery
	case ExchangeProductTargetKind_EXCHANGE_PRODUCT_TARGET_KIND_START_OPERATION:
		return ProductTargetStart
	default:
		return ""
	}
}

// ProductDelegationDigestEqual compares a broker-computed Product root digest
// to the signed child claim without allowing a caller-supplied digest to skip
// the constant-time equality check.
func ProductDelegationDigestEqual(claims GrantClaims, productRootDigest []byte) error {
	if len(productRootDigest) != 32 || subtle.ConstantTimeCompare(claims.RootRequestDigest, productRootDigest) != 1 {
		return ErrProductDelegationBinding
	}
	return nil
}

func scopesSubset(required, granted []string) bool {
	for _, scope := range required {
		if !containsScope(granted, scope) {
			return false
		}
	}
	return true
}
