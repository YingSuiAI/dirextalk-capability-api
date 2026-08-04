package capabilityv1

import (
	"bytes"
	"crypto/ed25519"
	"crypto/subtle"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode"
)

const (
	CapabilityGrantVersion = 1
	DefaultGrantTTL        = 15 * time.Minute
	DefaultGrantClockSkew  = 30 * time.Second
	DefaultGrantMaxSize    = 8 << 10
	GrantPrivateKeySize    = ed25519.PrivateKeySize
	GrantPublicKeySize     = ed25519.PublicKeySize
	MinGrantKeySize        = GrantPublicKeySize // minimum verifier key material
	GrantSignatureSize     = ed25519.SignatureSize
	GrantPrefix            = "grant-v1"
	GrantKindRoot          = "root"
	GrantKindProductChild  = "product-child"
	ProductTargetQuery     = "query"
	ProductTargetStart     = "start_operation"

	OperationControlGrantVersion = 1
	DefaultControlGrantTTL       = 2 * time.Minute
	ControlGrantPrefix           = "control-grant-v1"
)

var (
	ErrInvalidGrant        = errors.New("invalid capability grant")
	ErrGrantExpired        = errors.New("capability grant expired")
	ErrGrantNotYetValid    = errors.New("capability grant is not yet valid")
	ErrGrantBinding        = errors.New("capability grant binding mismatch")
	ErrGrantKey            = errors.New("capability grant key is invalid")
	ErrGrantTooLarge       = errors.New("capability grant is too large")
	ErrInvalidControlGrant = errors.New("invalid operation control grant")
	ErrControlGrantExpired = errors.New("operation control grant expired")
)

// GrantClaims are the authenticated, non-secret claims carried by a root
// capability grant. The signing key is never serialized into the grant.
// ChainID/EntryRoute/EntryHop bind a grant to the private call that created it.
type GrantClaims struct {
	ChainID         string
	RootOperationID string
	// ChildOperationID is present only on a message-server-signed Product
	// child grant. It binds the opaque delegation to the nested StartOperation
	// UUID while RootOperationID remains the outer Agent request UUID.
	ChildOperationID string
	// GrantKind is a signed purpose discriminator. Root and Product child
	// grants share the grant-v1 envelope but are not interchangeable.
	GrantKind string
	// ProductTargetKind is signed only on Product child grants and separates
	// read-only Query delegation from replayable StartOperation delegation.
	ProductTargetKind   string
	OwnerID             string
	AccountGeneration   int64
	Scopes              []string
	RootCapabilityID    string
	RootOperation       string
	RootRequestDigest   []byte
	CatalogDigest       []byte
	SchemaDigest        []byte
	IssuedAtUnixMs      int64
	ExpiresAtUnixMs     int64
	EntryDeadlineUnixMs int64
	EntryRoute          string
	EntryHop            int32
	MaxHop              int32
	MaxRouteLength      int32
}

// GrantBinding identifies a request context to which a grant may be used.
// When CallContext is supplied its authenticated chain/root identifiers are
// checked in addition to the explicit fields.
type GrantBinding struct {
	CallContext       *CallContext
	ChainID           string
	RootOperationID   string
	OwnerID           string
	AccountGeneration int64
	RootCapabilityID  string
	RootOperation     string
	RootRequestDigest []byte
	CatalogDigest     []byte
	SchemaDigest      []byte
}

// RootGrantBinding is the Product boundary binding. The caller must provide
// the actual request CallContext; route and hop are checked exactly. Product
// authorizes its target descriptor independently, so target capability and
// operation are intentionally absent here.
type RootGrantBinding struct {
	CallContext       *CallContext
	RootOperationID   string
	OwnerID           string
	AccountGeneration int64
	RootRequestDigest []byte
	RequiredScopes    []string
}

// AgentGrantBinding is the Agent boundary binding. It is kept distinct from
// Product binding to make the exact entry route explicit at each peer.
type AgentGrantBinding struct {
	CallContext       *CallContext
	RootOperationID   string
	RootCapabilityID  string
	RootOperation     string
	OwnerID           string
	AccountGeneration int64
	RootRequestDigest []byte
	CatalogDigest     []byte
	SchemaDigest      []byte
	RequiredScopes    []string
}

// GrantCodec provides deterministic Ed25519 grant encoding and clock-injected
// verification. A zero codec uses safe defaults.
type GrantCodec struct {
	Now       func() time.Time
	MaxTTL    time.Duration
	ClockSkew time.Duration
	MaxSize   int
}

func (c GrantCodec) normalized() GrantCodec {
	if c.Now == nil {
		c.Now = time.Now
	}
	if c.MaxTTL <= 0 {
		c.MaxTTL = DefaultGrantTTL
	}
	if c.ClockSkew < 0 {
		c.ClockSkew = 0
	}
	if c.MaxSize <= 0 {
		c.MaxSize = DefaultGrantMaxSize
	}
	return c
}

// Sign creates a versioned Ed25519 capability grant. key may be raw 64-byte
// ed25519.PrivateKey material or a PKCS#8 PEM private key.
func (c GrantCodec) Sign(claims GrantClaims, key []byte) ([]byte, error) {
	c = c.normalized()
	privateKey, err := ParseGrantPrivateKey(key)
	if err != nil {
		return nil, err
	}
	now := c.Now().UnixMilli()
	if claims.IssuedAtUnixMs == 0 {
		claims.IssuedAtUnixMs = now
	}
	if claims.GrantKind == "" {
		claims.GrantKind = GrantKindRoot
	}
	if claims.ExpiresAtUnixMs == 0 {
		claims.ExpiresAtUnixMs = claims.IssuedAtUnixMs + c.MaxTTL.Milliseconds()
	}
	if claims.EntryDeadlineUnixMs == 0 {
		claims.EntryDeadlineUnixMs = claims.ExpiresAtUnixMs
	}
	if claims.MaxRouteLength == 0 {
		claims.MaxRouteLength = MaxRouteLength
	}
	if err := ValidateGrantClaims(claims, c.MaxTTL); err != nil {
		return nil, err
	}
	payload, err := marshalGrantClaims(claims)
	if err != nil {
		return nil, err
	}
	signature := ed25519.Sign(privateKey, payload)
	grant := GrantPrefix + "." + base64.RawURLEncoding.EncodeToString(payload) + "." + base64.RawURLEncoding.EncodeToString(signature)
	if len(grant) > c.MaxSize {
		return nil, ErrGrantTooLarge
	}
	return []byte(grant), nil
}

// Verify authenticates and validates a grant with an Ed25519 public key and a
// complete request binding. Signature-only verification is intentionally not an
// exported operation: callers must use an Agent/Product boundary helper or
// provide every owner/root/route/digest field through GrantBinding.
func (c GrantCodec) Verify(grant, key []byte, binding *GrantBinding) (GrantClaims, error) {
	if binding == nil {
		return GrantClaims{}, fmt.Errorf("%w: complete binding is required", ErrGrantBinding)
	}
	claims, err := c.verifySigned(grant, key)
	if err != nil {
		return GrantClaims{}, err
	}
	if err := validateGrantBinding(claims, *binding); err != nil {
		return GrantClaims{}, err
	}
	return claims, nil
}

// verifySigned performs signature, claim and time validation without a
// boundary binding. It is package-private so only strict Agent/Product
// helpers can use it after they have the actual CallContext.
func (c GrantCodec) verifySigned(grant, key []byte) (GrantClaims, error) {
	c = c.normalized()
	publicKey, err := ParseGrantPublicKey(key)
	if err != nil {
		return GrantClaims{}, err
	}
	if len(grant) == 0 {
		return GrantClaims{}, fmt.Errorf("%w: empty grant", ErrInvalidGrant)
	}
	if len(grant) > c.MaxSize {
		return GrantClaims{}, ErrGrantTooLarge
	}
	parts := strings.Split(string(grant), ".")
	if len(parts) != 3 || parts[0] != GrantPrefix || parts[1] == "" || parts[2] == "" {
		return GrantClaims{}, fmt.Errorf("%w: malformed envelope", ErrInvalidGrant)
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil || len(payload) == 0 {
		return GrantClaims{}, fmt.Errorf("%w: malformed payload encoding", ErrInvalidGrant)
	}
	signature, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil || len(signature) != GrantSignatureSize {
		return GrantClaims{}, fmt.Errorf("%w: malformed signature encoding", ErrInvalidGrant)
	}
	if !ed25519.Verify(publicKey, payload, signature) {
		return GrantClaims{}, fmt.Errorf("%w: signature mismatch", ErrInvalidGrant)
	}
	claims, err := unmarshalGrantClaims(payload)
	if err != nil {
		return GrantClaims{}, err
	}
	if err := ValidateGrantClaims(claims, c.MaxTTL); err != nil {
		return GrantClaims{}, err
	}
	now := c.Now().UnixMilli()
	if now+c.ClockSkew.Milliseconds() < claims.IssuedAtUnixMs {
		return GrantClaims{}, ErrGrantNotYetValid
	}
	if now >= claims.ExpiresAtUnixMs {
		return GrantClaims{}, ErrGrantExpired
	}
	return claims, nil
}

// SignCapabilityGrant and VerifyCapabilityGrant are functional wrappers.
func SignCapabilityGrant(claims GrantClaims, key []byte) ([]byte, error) {
	return (GrantCodec{}).Sign(claims, key)
}

func VerifyCapabilityGrant(grant, key []byte, now time.Time, binding *GrantBinding) (GrantClaims, error) {
	return (GrantCodec{Now: func() time.Time { return now }}).Verify(grant, key, binding)
}

// VerifyAgentRootBinding authenticates an Agent request and requires the exact
// ms→agent CallContext route. A grant with an agent entry is not accepted at
// this boundary because Agent requests are always entered from message-server.
func VerifyAgentRootBinding(grant, key []byte, now time.Time, binding AgentGrantBinding) (GrantClaims, error) {
	claims, err := (GrantCodec{Now: func() time.Time { return now }}).verifySigned(grant, key)
	if err != nil {
		return GrantClaims{}, err
	}
	if claims.GrantKind != GrantKindRoot || claims.ChildOperationID != "" {
		return GrantClaims{}, fmt.Errorf("%w: Product child grant cannot enter Agent root boundary", ErrGrantBinding)
	}
	if err := validateRootBinding(claims, binding.CallContext, binding.RootOperationID, binding.OwnerID, binding.AccountGeneration, binding.RootRequestDigest, binding.RequiredScopes, NodeAgent); err != nil {
		return GrantClaims{}, err
	}
	if err := validateAgentGrantDescriptor(claims, binding.RootCapabilityID, binding.RootOperation, binding.CatalogDigest, binding.SchemaDigest); err != nil {
		return GrantClaims{}, err
	}
	return claims, nil
}

// VerifyAgentQueryGrant is the explicit Agent Query spelling. Query callers
// must bind the root request UUID, capability/operation descriptor, request
// digest, catalog/schema digests and required scopes to the actual route.
func VerifyAgentQueryGrant(grant, key []byte, now time.Time, binding AgentGrantBinding) (GrantClaims, error) {
	return VerifyAgentRootBinding(grant, key, now, binding)
}

// VerifyRootBinding authenticates a Product request and requires the exact
// ms→agent→product or agent→product route. It does not bind target capability
// or operation; the Product descriptor and required scopes do that separately.
func VerifyRootBinding(grant, key []byte, now time.Time, binding RootGrantBinding) (GrantClaims, error) {
	claims, err := (GrantCodec{Now: func() time.Time { return now }}).verifySigned(grant, key)
	if err != nil {
		return GrantClaims{}, err
	}
	if claims.GrantKind != GrantKindRoot || claims.ChildOperationID != "" {
		return GrantClaims{}, fmt.Errorf("%w: Product child grant cannot enter root boundary", ErrGrantBinding)
	}
	if err := validateRootBinding(claims, binding.CallContext, binding.RootOperationID, binding.OwnerID, binding.AccountGeneration, binding.RootRequestDigest, binding.RequiredScopes, NodeProduct); err != nil {
		return GrantClaims{}, err
	}
	return claims, nil
}

func ValidateGrantClaims(claims GrantClaims, maxTTL time.Duration) error {
	if err := ValidateOperationID(claims.ChainID); err != nil {
		return fmt.Errorf("%w: chain_id: %v", ErrInvalidGrant, err)
	}
	if err := ValidateOperationID(claims.RootOperationID); err != nil {
		return fmt.Errorf("%w: root_operation_id: %v", ErrInvalidGrant, err)
	}
	if claims.GrantKind != GrantKindRoot && claims.GrantKind != GrantKindProductChild {
		return fmt.Errorf("%w: unsupported grant_kind", ErrInvalidGrant)
	}
	if claims.ChildOperationID != "" {
		if err := ValidateOperationID(claims.ChildOperationID); err != nil {
			return fmt.Errorf("%w: child_operation_id: %v", ErrInvalidGrant, err)
		}
	}
	if claims.GrantKind == GrantKindRoot && (claims.ProductTargetKind != "" || claims.ChildOperationID != "") {
		return fmt.Errorf("%w: root grant cannot carry Product child fields", ErrInvalidGrant)
	}
	if claims.GrantKind == GrantKindProductChild && claims.ProductTargetKind != ProductTargetQuery && claims.ProductTargetKind != ProductTargetStart {
		return fmt.Errorf("%w: Product child target kind is invalid", ErrInvalidGrant)
	}
	if claims.ProductTargetKind == ProductTargetQuery && claims.ChildOperationID != "" {
		return fmt.Errorf("%w: Query child grant cannot carry child_operation_id", ErrInvalidGrant)
	}
	if claims.ProductTargetKind == ProductTargetStart && claims.ChildOperationID == "" {
		return fmt.Errorf("%w: Start child grant requires child_operation_id", ErrInvalidGrant)
	}
	if err := validateGrantToken("owner_id", claims.OwnerID, 256); err != nil {
		return err
	}
	if claims.AccountGeneration <= 0 {
		return fmt.Errorf("%w: account_generation must be positive", ErrInvalidGrant)
	}
	if err := validateGrantToken("root_capability_id", claims.RootCapabilityID, 256); err != nil {
		return err
	}
	if err := validateGrantToken("root_operation", claims.RootOperation, 256); err != nil {
		return err
	}
	if len(claims.RootRequestDigest) != 32 || len(claims.CatalogDigest) != 32 || len(claims.SchemaDigest) != 32 {
		return fmt.Errorf("%w: root/catalog/schema digest must be 32 bytes", ErrInvalidGrant)
	}
	if len(claims.Scopes) == 0 || !isSortedUnique(claims.Scopes) {
		return fmt.Errorf("%w: scopes must be non-empty, sorted, and unique", ErrInvalidGrant)
	}
	if claims.IssuedAtUnixMs <= 0 || claims.ExpiresAtUnixMs <= claims.IssuedAtUnixMs {
		return fmt.Errorf("%w: invalid issued/expiry timestamps", ErrInvalidGrant)
	}
	if claims.EntryDeadlineUnixMs < claims.IssuedAtUnixMs || claims.EntryDeadlineUnixMs > claims.ExpiresAtUnixMs {
		return fmt.Errorf("%w: entry deadline must be within grant lifetime", ErrInvalidGrant)
	}
	if maxTTL <= 0 {
		maxTTL = DefaultGrantTTL
	}
	if claims.ExpiresAtUnixMs-claims.IssuedAtUnixMs > maxTTL.Milliseconds() {
		return fmt.Errorf("%w: TTL exceeds %s", ErrInvalidGrant, maxTTL)
	}
	if claims.MaxHop <= 0 || claims.MaxHop > MaxCallHop {
		return fmt.Errorf("%w: max_hop must be in [1,%d]", ErrInvalidGrant, MaxCallHop)
	}
	if claims.MaxRouteLength <= 0 || claims.MaxRouteLength > MaxRouteLength {
		return fmt.Errorf("%w: max_route_length must be in [1,%d]", ErrInvalidGrant, MaxRouteLength)
	}
	if err := validateEntryRoute(claims.EntryRoute, claims.EntryHop); err != nil {
		return err
	}
	if claims.MaxRouteLength < int32(len(claims.EntryRoute)) {
		return fmt.Errorf("%w: max_route_length is shorter than entry route", ErrInvalidGrant)
	}
	return nil
}

// ParseGrantPrivateKey accepts only raw Ed25519 key material or a PKCS#8 PEM
// block. Raw private keys are exactly 64 bytes. A 32-byte public key is never
// accepted as signing material, preventing verifier-only processes from
// minting grants.
func ParseGrantPrivateKey(data []byte) (ed25519.PrivateKey, error) {
	block, err := decodeKeyPEM(data)
	if err != nil {
		return nil, err
	}
	if block != nil {
		if block.Type != "PRIVATE KEY" {
			return nil, fmt.Errorf("%w: expected PRIVATE KEY PEM", ErrGrantKey)
		}
		key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("%w: invalid PKCS#8 private key: %v", ErrGrantKey, err)
		}
		private, ok := key.(ed25519.PrivateKey)
		if !ok || len(private) != ed25519.PrivateKeySize {
			return nil, fmt.Errorf("%w: PEM key is not Ed25519", ErrGrantKey)
		}
		return append(ed25519.PrivateKey(nil), private...), nil
	}
	switch len(data) {
	case GrantPrivateKeySize:
		return append(ed25519.PrivateKey(nil), data...), nil
	default:
		return nil, fmt.Errorf("%w: raw private key must be %d bytes", ErrGrantKey, ed25519.PrivateKeySize)
	}
}

// ParseGrantPublicKey accepts only raw 32-byte Ed25519 public keys or a PKIX
// PUBLIC KEY PEM block.
func ParseGrantPublicKey(data []byte) (ed25519.PublicKey, error) {
	block, err := decodeKeyPEM(data)
	if err != nil {
		return nil, err
	}
	if block != nil {
		if block.Type != "PUBLIC KEY" {
			return nil, fmt.Errorf("%w: expected PUBLIC KEY PEM", ErrGrantKey)
		}
		key, err := x509.ParsePKIXPublicKey(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("%w: invalid PKIX public key: %v", ErrGrantKey, err)
		}
		public, ok := key.(ed25519.PublicKey)
		if !ok || len(public) != ed25519.PublicKeySize {
			return nil, fmt.Errorf("%w: PEM key is not Ed25519", ErrGrantKey)
		}
		return append(ed25519.PublicKey(nil), public...), nil
	}
	if len(data) != ed25519.PublicKeySize {
		return nil, fmt.Errorf("%w: raw public key must be %d bytes", ErrGrantKey, ed25519.PublicKeySize)
	}
	return append(ed25519.PublicKey(nil), data...), nil
}

func MarshalGrantPrivateKeyPEM(key []byte) ([]byte, error) {
	private, err := ParseGrantPrivateKey(key)
	if err != nil {
		return nil, err
	}
	der, err := x509.MarshalPKCS8PrivateKey(private)
	if err != nil {
		return nil, fmt.Errorf("%w: marshal private key: %v", ErrGrantKey, err)
	}
	return pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der}), nil
}

func MarshalGrantPublicKeyPEM(key []byte) ([]byte, error) {
	public, err := ParseGrantPublicKey(key)
	if err != nil {
		return nil, err
	}
	der, err := x509.MarshalPKIXPublicKey(public)
	if err != nil {
		return nil, fmt.Errorf("%w: marshal public key: %v", ErrGrantKey, err)
	}
	return pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: der}), nil
}

func decodeKeyPEM(data []byte) (*pem.Block, error) {
	if !bytes.HasPrefix(data, []byte("-----BEGIN ")) {
		if bytes.Contains(data, []byte("-----BEGIN")) {
			return nil, fmt.Errorf("%w: malformed PEM", ErrGrantKey)
		}
		return nil, nil
	}
	block, rest := pem.Decode(data)
	if block == nil || len(bytes.TrimSpace(rest)) != 0 {
		return nil, fmt.Errorf("%w: malformed PEM", ErrGrantKey)
	}
	return block, nil
}

func validateRootBinding(claims GrantClaims, callCtx *CallContext, rootOperationID, ownerID string, generation int64, rootDigest []byte, requiredScopes []string, terminal string) error {
	if callCtx == nil || rootOperationID == "" || ownerID == "" || generation <= 0 || len(rootDigest) != 32 || len(requiredScopes) == 0 || !isSortedUnique(requiredScopes) {
		return fmt.Errorf("%w: complete root binding is required", ErrGrantBinding)
	}
	if err := ValidateStrictCallContext(callCtx); err != nil {
		return fmt.Errorf("%w: invalid call context: %v", ErrGrantBinding, err)
	}
	if claims.ChainID != callCtx.ChainId {
		return fmt.Errorf("%w: chain_id", ErrGrantBinding)
	}
	if claims.RootOperationID != rootOperationID || callCtx.RootOperationId != rootOperationID {
		return fmt.Errorf("%w: root_operation_id", ErrGrantBinding)
	}
	if claims.OwnerID != ownerID {
		return fmt.Errorf("%w: owner_id", ErrGrantBinding)
	}
	if claims.AccountGeneration != generation {
		return fmt.Errorf("%w: account_generation", ErrGrantBinding)
	}
	if subtle.ConstantTimeCompare(claims.RootRequestDigest, rootDigest) != 1 {
		return fmt.Errorf("%w: root_request_digest", ErrGrantBinding)
	}
	if callCtx.DeadlineUnixMs > claims.EntryDeadlineUnixMs || callCtx.DeadlineUnixMs > claims.ExpiresAtUnixMs {
		return fmt.Errorf("%w: deadline exceeds grant expiry", ErrGrantBinding)
	}
	for _, scope := range requiredScopes {
		if !containsScope(claims.Scopes, scope) {
			return fmt.Errorf("%w: missing scope %s", ErrGrantBinding, scope)
		}
	}
	expectedRoute, expectedHop, err := expectedBoundaryRoute(claims, terminal)
	if err != nil {
		return err
	}
	if callCtx.Route != expectedRoute || callCtx.Hop != expectedHop || callCtx.Hop > claims.MaxHop || int32(len(callCtx.Route)) > claims.MaxRouteLength {
		return fmt.Errorf("%w: route/hop", ErrGrantBinding)
	}
	return nil
}

func validateAgentGrantDescriptor(claims GrantClaims, capabilityID, operation string, catalogDigest, schemaDigest []byte) error {
	if capabilityID == "" || operation == "" || len(catalogDigest) != 32 || len(schemaDigest) != 32 {
		return fmt.Errorf("%w: Agent capability descriptor binding is required", ErrGrantBinding)
	}
	if claims.RootCapabilityID != capabilityID || claims.RootOperation != operation {
		return fmt.Errorf("%w: root capability/operation", ErrGrantBinding)
	}
	if subtle.ConstantTimeCompare(claims.CatalogDigest, catalogDigest) != 1 || subtle.ConstantTimeCompare(claims.SchemaDigest, schemaDigest) != 1 {
		return fmt.Errorf("%w: catalog/schema digest", ErrGrantBinding)
	}
	return nil
}

func expectedBoundaryRoute(claims GrantClaims, terminal string) (string, int32, error) {
	if terminal == NodeAgent {
		if claims.EntryRoute != NodeMessage {
			return "", 0, fmt.Errorf("%w: Agent entry route must be ms", ErrGrantBinding)
		}
		return NodeMessage + RouteSeparator + NodeAgent, claims.EntryHop + 1, nil
	}
	if terminal != NodeProduct {
		return "", 0, fmt.Errorf("%w: unsupported boundary", ErrGrantBinding)
	}
	if claims.EntryRoute == NodeMessage {
		return NodeMessage + RouteSeparator + NodeAgent + RouteSeparator + NodeProduct, claims.EntryHop + 2, nil
	}
	return claims.EntryRoute + RouteSeparator + NodeProduct, claims.EntryHop + 1, nil
}

func validateEntryRoute(route string, hop int32) error {
	nodes, err := parseRoute(route)
	if err != nil || len(nodes) != 1 || (nodes[0] != NodeMessage && nodes[0] != NodeAgent) || hop != 1 {
		return fmt.Errorf("%w: entry route must be a single ms or agent node with hop 1", ErrInvalidGrant)
	}
	return nil
}

type grantWire struct {
	Version             int32    `json:"v"`
	ChainID             string   `json:"chain_id"`
	RootOperationID     string   `json:"root_operation_id"`
	ChildOperationID    string   `json:"child_operation_id"`
	GrantKind           string   `json:"grant_kind"`
	ProductTargetKind   string   `json:"product_target_kind"`
	OwnerID             string   `json:"owner_id"`
	AccountGeneration   int64    `json:"account_generation"`
	Scopes              []string `json:"scopes"`
	RootCapabilityID    string   `json:"root_capability_id"`
	RootOperation       string   `json:"root_operation"`
	RootRequestDigest   string   `json:"root_request_digest"`
	CatalogDigest       string   `json:"catalog_digest"`
	SchemaDigest        string   `json:"schema_digest"`
	IssuedAtUnixMs      int64    `json:"issued_at_unix_ms"`
	ExpiresAtUnixMs     int64    `json:"expires_at_unix_ms"`
	EntryDeadlineUnixMs int64    `json:"entry_deadline_unix_ms"`
	EntryRoute          string   `json:"entry_route"`
	EntryHop            int32    `json:"entry_hop"`
	MaxHop              int32    `json:"max_hop"`
	MaxRouteLength      int32    `json:"max_route_length"`
}

func marshalGrantClaims(claims GrantClaims) ([]byte, error) {
	return Canonicalize(map[string]interface{}{
		"v":                      CapabilityGrantVersion,
		"chain_id":               claims.ChainID,
		"root_operation_id":      claims.RootOperationID,
		"child_operation_id":     claims.ChildOperationID,
		"grant_kind":             claims.GrantKind,
		"product_target_kind":    claims.ProductTargetKind,
		"owner_id":               claims.OwnerID,
		"account_generation":     claims.AccountGeneration,
		"scopes":                 claims.Scopes,
		"root_capability_id":     claims.RootCapabilityID,
		"root_operation":         claims.RootOperation,
		"root_request_digest":    base64.RawURLEncoding.EncodeToString(claims.RootRequestDigest),
		"catalog_digest":         base64.RawURLEncoding.EncodeToString(claims.CatalogDigest),
		"schema_digest":          base64.RawURLEncoding.EncodeToString(claims.SchemaDigest),
		"issued_at_unix_ms":      claims.IssuedAtUnixMs,
		"expires_at_unix_ms":     claims.ExpiresAtUnixMs,
		"entry_deadline_unix_ms": claims.EntryDeadlineUnixMs,
		"entry_route":            claims.EntryRoute,
		"entry_hop":              claims.EntryHop,
		"max_hop":                claims.MaxHop,
		"max_route_length":       claims.MaxRouteLength,
	})
}

func unmarshalGrantClaims(payload []byte) (GrantClaims, error) {
	canonical, err := CanonicalizeJSON(payload)
	if err != nil || string(canonical) != string(payload) {
		return GrantClaims{}, fmt.Errorf("%w: payload is not canonical JSON", ErrInvalidGrant)
	}
	var wire grantWire
	dec := json.NewDecoder(bytes.NewReader(payload))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&wire); err != nil {
		return GrantClaims{}, fmt.Errorf("%w: invalid payload: %v", ErrInvalidGrant, err)
	}
	if wire.Version != CapabilityGrantVersion {
		return GrantClaims{}, fmt.Errorf("%w: unsupported grant version %d", ErrInvalidGrant, wire.Version)
	}
	decodeDigest := func(name, value string) ([]byte, error) {
		decoded, err := base64.RawURLEncoding.DecodeString(value)
		if err != nil || len(decoded) != 32 {
			return nil, fmt.Errorf("%w: %s digest encoding", ErrInvalidGrant, name)
		}
		return decoded, nil
	}
	rootDigest, err := decodeDigest("root_request", wire.RootRequestDigest)
	if err != nil {
		return GrantClaims{}, err
	}
	catalog, err := decodeDigest("catalog", wire.CatalogDigest)
	if err != nil {
		return GrantClaims{}, err
	}
	schema, err := decodeDigest("schema", wire.SchemaDigest)
	if err != nil {
		return GrantClaims{}, err
	}
	return GrantClaims{
		ChainID:             wire.ChainID,
		RootOperationID:     wire.RootOperationID,
		ChildOperationID:    wire.ChildOperationID,
		GrantKind:           wire.GrantKind,
		ProductTargetKind:   wire.ProductTargetKind,
		OwnerID:             wire.OwnerID,
		AccountGeneration:   wire.AccountGeneration,
		Scopes:              wire.Scopes,
		RootCapabilityID:    wire.RootCapabilityID,
		RootOperation:       wire.RootOperation,
		RootRequestDigest:   rootDigest,
		CatalogDigest:       catalog,
		SchemaDigest:        schema,
		IssuedAtUnixMs:      wire.IssuedAtUnixMs,
		ExpiresAtUnixMs:     wire.ExpiresAtUnixMs,
		EntryDeadlineUnixMs: wire.EntryDeadlineUnixMs,
		EntryRoute:          wire.EntryRoute,
		EntryHop:            wire.EntryHop,
		MaxHop:              wire.MaxHop,
		MaxRouteLength:      wire.MaxRouteLength,
	}, nil
}

func validateGrantBinding(claims GrantClaims, binding GrantBinding) error {
	if binding.CallContext == nil || binding.RootOperationID == "" || binding.OwnerID == "" || binding.AccountGeneration <= 0 || binding.RootCapabilityID == "" || binding.RootOperation == "" || len(binding.RootRequestDigest) != 32 || len(binding.CatalogDigest) != 32 || len(binding.SchemaDigest) != 32 {
		return fmt.Errorf("%w: complete binding is required", ErrGrantBinding)
	}
	if err := ValidateStrictCallContext(binding.CallContext); err != nil {
		return fmt.Errorf("%w: invalid call context: %v", ErrGrantBinding, err)
	}
	if claims.ChainID != binding.CallContext.ChainId || claims.RootOperationID != binding.CallContext.RootOperationId {
		return fmt.Errorf("%w: call context root", ErrGrantBinding)
	}
	if binding.CallContext.Route != claims.EntryRoute || binding.CallContext.Hop != claims.EntryHop {
		return fmt.Errorf("%w: call context entry route", ErrGrantBinding)
	}
	if binding.ChainID != "" && claims.ChainID != binding.ChainID {
		return fmt.Errorf("%w: chain_id", ErrGrantBinding)
	}
	if binding.RootOperationID != "" && claims.RootOperationID != binding.RootOperationID {
		return fmt.Errorf("%w: root_operation_id", ErrGrantBinding)
	}
	if binding.OwnerID != "" && claims.OwnerID != binding.OwnerID {
		return fmt.Errorf("%w: owner_id", ErrGrantBinding)
	}
	if binding.AccountGeneration != 0 && claims.AccountGeneration != binding.AccountGeneration {
		return fmt.Errorf("%w: account_generation", ErrGrantBinding)
	}
	if binding.RootCapabilityID != "" && claims.RootCapabilityID != binding.RootCapabilityID {
		return fmt.Errorf("%w: root_capability_id", ErrGrantBinding)
	}
	if binding.RootOperation != "" && claims.RootOperation != binding.RootOperation {
		return fmt.Errorf("%w: root_operation", ErrGrantBinding)
	}
	if len(binding.RootRequestDigest) > 0 && subtle.ConstantTimeCompare(claims.RootRequestDigest, binding.RootRequestDigest) != 1 {
		return fmt.Errorf("%w: root_request_digest", ErrGrantBinding)
	}
	if len(binding.CatalogDigest) > 0 && subtle.ConstantTimeCompare(claims.CatalogDigest, binding.CatalogDigest) != 1 {
		return fmt.Errorf("%w: catalog_digest", ErrGrantBinding)
	}
	if len(binding.SchemaDigest) > 0 && subtle.ConstantTimeCompare(claims.SchemaDigest, binding.SchemaDigest) != 1 {
		return fmt.Errorf("%w: schema_digest", ErrGrantBinding)
	}
	return nil
}

func validateGrantToken(name, value string, maxLen int) error {
	if value == "" || len(value) > maxLen || strings.TrimSpace(value) != value {
		return fmt.Errorf("%w: %s is invalid", ErrInvalidGrant, name)
	}
	for _, r := range value {
		if unicode.IsControl(r) || r == '→' {
			return fmt.Errorf("%w: %s contains a forbidden delimiter", ErrInvalidGrant, name)
		}
	}
	return nil
}

func isSortedUnique(values []string) bool {
	for i, value := range values {
		if err := validateGrantToken("scope", value, 128); err != nil {
			return false
		}
		if i > 0 && values[i-1] >= value {
			return false
		}
	}
	return true
}

func containsScope(scopes []string, wanted string) bool {
	for _, scope := range scopes {
		if scope == wanted {
			return true
		}
	}
	return false
}

// OperationControlGrant is a separate, short-lived grant for control-plane
// operations on an existing operation. Its domain-separated envelope cannot
// be accepted by root capability grant verifiers and vice versa.
type OperationControlGrant struct {
	ChainID           string
	OwnerID           string
	AccountGeneration int64
	OperationID       string
	ControlAction     string
	ControlScope      string
	EntryRoute        string
	EntryHop          int32
	DeadlineUnixMs    int64
	IssuedAtUnixMs    int64
	ExpiresAtUnixMs   int64
}

type OperationControlGrantBinding struct {
	// CallContext is the actual request context at the receiving boundary;
	// callers must not pass an entry-route prefix or manually strip a hop.
	CallContext       *CallContext
	OwnerID           string
	AccountGeneration int64
	OperationID       string
	ControlAction     string
	ControlScope      string
}

func (c GrantCodec) SignOperationControlGrant(claims OperationControlGrant, key []byte) ([]byte, error) {
	c = c.normalized()
	if c.MaxTTL > DefaultControlGrantTTL {
		c.MaxTTL = DefaultControlGrantTTL
	}
	private, err := ParseGrantPrivateKey(key)
	if err != nil {
		return nil, err
	}
	now := c.Now().UnixMilli()
	if claims.IssuedAtUnixMs == 0 {
		claims.IssuedAtUnixMs = now
	}
	if claims.ExpiresAtUnixMs == 0 {
		claims.ExpiresAtUnixMs = claims.IssuedAtUnixMs + c.MaxTTL.Milliseconds()
	}
	if claims.DeadlineUnixMs == 0 {
		claims.DeadlineUnixMs = claims.ExpiresAtUnixMs
	}
	if err := ValidateOperationControlGrant(claims, c.MaxTTL); err != nil {
		return nil, err
	}
	payload, err := Canonicalize(map[string]interface{}{
		"v":                  OperationControlGrantVersion,
		"chain_id":           claims.ChainID,
		"owner_id":           claims.OwnerID,
		"account_generation": claims.AccountGeneration,
		"operation_id":       claims.OperationID,
		"control_action":     claims.ControlAction,
		"control_scope":      claims.ControlScope,
		"entry_route":        claims.EntryRoute,
		"entry_hop":          claims.EntryHop,
		"deadline_unix_ms":   claims.DeadlineUnixMs,
		"issued_at_unix_ms":  claims.IssuedAtUnixMs,
		"expires_at_unix_ms": claims.ExpiresAtUnixMs,
	})
	if err != nil {
		return nil, err
	}
	sig := ed25519.Sign(private, payload)
	grant := ControlGrantPrefix + "." + base64.RawURLEncoding.EncodeToString(payload) + "." + base64.RawURLEncoding.EncodeToString(sig)
	if len(grant) > c.MaxSize {
		return nil, ErrGrantTooLarge
	}
	return []byte(grant), nil
}

func (c GrantCodec) VerifyOperationControlGrant(grant, key []byte, binding OperationControlGrantBinding) (OperationControlGrant, error) {
	return c.verifyOperationControlGrant(grant, key, binding, NodeAgent)
}

// VerifyProductOperationControlGrant verifies a control grant at the Product
// boundary and accepts the exact ms→agent→product or agent→product route.
func (c GrantCodec) VerifyProductOperationControlGrant(grant, key []byte, binding OperationControlGrantBinding) (OperationControlGrant, error) {
	return c.verifyOperationControlGrant(grant, key, binding, NodeProduct)
}

func (c GrantCodec) verifyOperationControlGrant(grant, key []byte, binding OperationControlGrantBinding, terminal string) (OperationControlGrant, error) {
	c = c.normalized()
	public, err := ParseGrantPublicKey(key)
	if err != nil {
		return OperationControlGrant{}, err
	}
	if len(grant) == 0 {
		return OperationControlGrant{}, ErrInvalidControlGrant
	}
	if len(grant) > c.MaxSize {
		return OperationControlGrant{}, ErrGrantTooLarge
	}
	parts := strings.Split(string(grant), ".")
	if len(parts) != 3 || parts[0] != ControlGrantPrefix {
		return OperationControlGrant{}, ErrInvalidControlGrant
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil || !ed25519.Verify(public, payload, mustDecodeSignature(parts[2])) {
		return OperationControlGrant{}, ErrInvalidControlGrant
	}
	canonical, err := CanonicalizeJSON(payload)
	if err != nil || !bytes.Equal(canonical, payload) {
		return OperationControlGrant{}, ErrInvalidControlGrant
	}
	var wire struct {
		Version           int32  `json:"v"`
		ChainID           string `json:"chain_id"`
		OwnerID           string `json:"owner_id"`
		AccountGeneration int64  `json:"account_generation"`
		OperationID       string `json:"operation_id"`
		ControlAction     string `json:"control_action"`
		ControlScope      string `json:"control_scope"`
		EntryRoute        string `json:"entry_route"`
		EntryHop          int32  `json:"entry_hop"`
		DeadlineUnixMs    int64  `json:"deadline_unix_ms"`
		IssuedAtUnixMs    int64  `json:"issued_at_unix_ms"`
		ExpiresAtUnixMs   int64  `json:"expires_at_unix_ms"`
	}
	dec := json.NewDecoder(bytes.NewReader(payload))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&wire); err != nil || wire.Version != OperationControlGrantVersion {
		return OperationControlGrant{}, ErrInvalidControlGrant
	}
	claims := OperationControlGrant{wire.ChainID, wire.OwnerID, wire.AccountGeneration, wire.OperationID, wire.ControlAction, wire.ControlScope, wire.EntryRoute, wire.EntryHop, wire.DeadlineUnixMs, wire.IssuedAtUnixMs, wire.ExpiresAtUnixMs}
	if err := ValidateOperationControlGrant(claims, minDuration(c.MaxTTL, DefaultControlGrantTTL)); err != nil {
		return OperationControlGrant{}, err
	}
	now := c.Now().UnixMilli()
	if now >= claims.ExpiresAtUnixMs {
		return OperationControlGrant{}, ErrControlGrantExpired
	}
	if now+c.ClockSkew.Milliseconds() < claims.IssuedAtUnixMs {
		return OperationControlGrant{}, ErrGrantNotYetValid
	}
	if err := validateControlBinding(claims, binding, terminal); err != nil {
		return OperationControlGrant{}, err
	}
	return claims, nil
}

func ValidateOperationControlGrant(claims OperationControlGrant, maxTTL time.Duration) error {
	if err := ValidateOperationID(claims.ChainID); err != nil {
		return fmt.Errorf("%w: chain_id", ErrInvalidControlGrant)
	}
	if err := ValidateOperationID(claims.OperationID); err != nil {
		return fmt.Errorf("%w: operation_id", ErrInvalidControlGrant)
	}
	if err := validateGrantToken("owner_id", claims.OwnerID, 256); err != nil {
		return fmt.Errorf("%w: owner_id", ErrInvalidControlGrant)
	}
	if claims.AccountGeneration <= 0 || claims.EntryHop <= 0 || claims.DeadlineUnixMs <= 0 || claims.IssuedAtUnixMs <= 0 || claims.ExpiresAtUnixMs <= claims.IssuedAtUnixMs || claims.DeadlineUnixMs > claims.ExpiresAtUnixMs {
		return ErrInvalidControlGrant
	}
	if claims.ControlAction != "get" && claims.ControlAction != "watch" && claims.ControlAction != "cancel" && claims.ControlAction != "reconcile" {
		return ErrInvalidControlGrant
	}
	if claims.ControlScope != "operation:control:"+claims.ControlAction {
		return ErrInvalidControlGrant
	}
	if err := validateEntryRoute(claims.EntryRoute, claims.EntryHop); err != nil {
		return fmt.Errorf("%w: entry route", ErrInvalidControlGrant)
	}
	if maxTTL <= 0 {
		maxTTL = DefaultControlGrantTTL
	}
	if claims.ExpiresAtUnixMs-claims.IssuedAtUnixMs > maxTTL.Milliseconds() {
		return ErrInvalidControlGrant
	}
	return nil
}

func validateControlBinding(claims OperationControlGrant, binding OperationControlGrantBinding, terminal string) error {
	if binding.CallContext == nil || binding.OwnerID == "" || binding.OperationID == "" || binding.AccountGeneration <= 0 || binding.ControlAction == "" || binding.ControlScope == "" {
		return ErrGrantBinding
	}
	if err := ValidateStrictCallContext(binding.CallContext); err != nil {
		return fmt.Errorf("%w: call context", ErrGrantBinding)
	}
	if claims.ChainID != binding.CallContext.ChainId || claims.OwnerID != binding.OwnerID || claims.AccountGeneration != binding.AccountGeneration || claims.OperationID != binding.OperationID || claims.ControlAction != binding.ControlAction || claims.ControlScope != binding.ControlScope {
		return ErrGrantBinding
	}
	if binding.CallContext.DeadlineUnixMs > claims.DeadlineUnixMs {
		return ErrGrantBinding
	}
	expectedRoute, expectedHop, err := expectedControlBoundaryRoute(claims, terminal)
	if err != nil || binding.CallContext.Route != expectedRoute || binding.CallContext.Hop != expectedHop {
		return ErrGrantBinding
	}
	return nil
}

func expectedControlBoundaryRoute(claims OperationControlGrant, terminal string) (string, int32, error) {
	switch terminal {
	case NodeAgent:
		if claims.EntryRoute != NodeMessage {
			return "", 0, fmt.Errorf("%w: Agent control entry must be ms", ErrGrantBinding)
		}
		return NodeMessage + RouteSeparator + NodeAgent, claims.EntryHop + 1, nil
	case NodeProduct:
		if claims.EntryRoute == NodeMessage {
			return NodeMessage + RouteSeparator + NodeAgent + RouteSeparator + NodeProduct, claims.EntryHop + 2, nil
		}
		return NodeAgent + RouteSeparator + NodeProduct, claims.EntryHop + 1, nil
	default:
		return "", 0, fmt.Errorf("%w: unsupported control boundary", ErrGrantBinding)
	}
}

func minDuration(a, b time.Duration) time.Duration {
	if a < b {
		return a
	}
	return b
}

func mustDecodeSignature(value string) []byte {
	sig, _ := base64.RawURLEncoding.DecodeString(value)
	if len(sig) != ed25519.SignatureSize {
		return nil
	}
	return sig
}
