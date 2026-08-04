package capabilityv1

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"testing"
	"time"
)

func productDelegationClaims() GrantClaims {
	claims := testGrantClaims()
	claims.ChildOperationID = "018f2f1e-7b5b-7b21-8f2e-4a6f3b1d9c12"
	claims.GrantKind = GrantKindProductChild
	claims.ProductTargetKind = ProductTargetStart
	claims.RootCapabilityID = "product.contacts.v1"
	claims.RootOperation = "list"
	claims.RootRequestDigest = bytes.Repeat([]byte{0x77}, sha256.Size)
	claims.CatalogDigest = bytes.Repeat([]byte{0x78}, sha256.Size)
	claims.SchemaDigest = bytes.Repeat([]byte{0x79}, sha256.Size)
	claims.ExpiresAtUnixMs = claims.IssuedAtUnixMs + 90_000
	claims.EntryDeadlineUnixMs = claims.ExpiresAtUnixMs
	return claims
}

func productDelegationBinding(claims GrantClaims) ProductGrantBinding {
	callContext := testProductCallContext()
	// The child grant intentionally has a short two-minute lifetime; bind the
	// test call deadline to that lifetime so the exact-target success path does
	// not fail before descriptor checks run.
	callContext.DeadlineUnixMs = claims.ExpiresAtUnixMs
	return ProductGrantBinding{
		CallContext:       callContext,
		RootOperationID:   claims.RootOperationID,
		ChildOperationID:  claims.ChildOperationID,
		OwnerID:           claims.OwnerID,
		AccountGeneration: claims.AccountGeneration,
		RequiredScopes:    []string{"contacts:read"},
		CapabilityID:      claims.RootCapabilityID,
		Operation:         claims.RootOperation,
		TargetKind:        ExchangeProductTargetKind_EXCHANGE_PRODUCT_TARGET_KIND_START_OPERATION,
		RootRequestDigest: claims.RootRequestDigest,
		CatalogDigest:     claims.CatalogDigest,
		SchemaDigest:      claims.SchemaDigest,
	}
}

func TestVerifyProductDelegationGrantBindsExactTarget(t *testing.T) {
	publicKey, privateKey := testGrantKeys()
	claims := productDelegationClaims()
	now := time.UnixMilli(claims.IssuedAtUnixMs + 1_000)
	grant, err := (GrantCodec{Now: func() time.Time { return now }}).SignProductDelegationGrant(claims, privateKey)
	if err != nil {
		t.Fatal(err)
	}
	verified, err := VerifyProductDelegationGrant(grant, publicKey, now, productDelegationBinding(claims))
	if err != nil || verified.RootCapabilityID != claims.RootCapabilityID {
		t.Fatalf("exact Product delegation rejected: %#v %v", verified, err)
	}
	if _, err := VerifyAgentRootBinding(grant, publicKey, now, AgentGrantBinding{
		CallContext: testAgentCallContext(), RootOperationID: claims.RootOperationID,
		RootCapabilityID: claims.RootCapabilityID, RootOperation: claims.RootOperation,
		OwnerID: claims.OwnerID, AccountGeneration: claims.AccountGeneration,
		RootRequestDigest: claims.RootRequestDigest, CatalogDigest: claims.CatalogDigest,
		SchemaDigest: claims.SchemaDigest, RequiredScopes: []string{"contacts:read"},
	}); !errors.Is(err, ErrGrantBinding) {
		t.Fatalf("Product child grant entered Agent root boundary: %v", err)
	}
	for name, mutate := range map[string]func(*ProductGrantBinding){
		"capability": func(binding *ProductGrantBinding) { binding.CapabilityID = "product.rooms.v1" },
		"operation":  func(binding *ProductGrantBinding) { binding.Operation = "write" },
		"child operation": func(binding *ProductGrantBinding) {
			binding.ChildOperationID = "018f2f1e-7b5b-7b21-8f2e-4a6f3b1d9c13"
		},
		"root digest": func(binding *ProductGrantBinding) {
			binding.RootRequestDigest = bytes.Repeat([]byte{0x70}, sha256.Size)
		},
		"schema": func(binding *ProductGrantBinding) {
			binding.SchemaDigest = bytes.Repeat([]byte{0x70}, sha256.Size)
		},
		"route truncation": func(binding *ProductGrantBinding) {
			binding.CallContext = &CallContext{ChainId: testGrantChain, RootOperationId: testGrantRoot, Hop: 2, Route: NodeMessage + RouteSeparator + NodeAgent, DeadlineUnixMs: 1_700_000_500_000}
		},
	} {
		t.Run(name, func(t *testing.T) {
			binding := productDelegationBinding(claims)
			mutate(&binding)
			if _, err := VerifyProductDelegationGrant(grant, publicKey, now, binding); !errors.Is(err, ErrProductDelegationBinding) && !errors.Is(err, ErrGrantBinding) {
				t.Fatalf("mutation accepted: %v", err)
			}
		})
	}
	parentClaims := claims
	parentClaims.RootCapabilityID = "agent.skills.v1"
	parentClaims.RootOperation = "invoke_product"
	parentClaims.GrantKind = GrantKindRoot
	parentClaims.ProductTargetKind = ""
	parentClaims.ChildOperationID = ""
	parentGrant, err := (GrantCodec{Now: func() time.Time { return now }}).Sign(parentClaims, privateKey)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := VerifyProductDelegationGrant(parentGrant, publicKey, now, productDelegationBinding(claims)); !errors.Is(err, ErrProductDelegationBinding) && !errors.Is(err, ErrGrantBinding) {
		t.Fatalf("Agent root grant was accepted as Product delegation: %v", err)
	}
	longGeneric := claims
	longGeneric.IssuedAtUnixMs = now.UnixMilli()
	longGeneric.ExpiresAtUnixMs = now.Add(5 * time.Minute).UnixMilli()
	longGeneric.EntryDeadlineUnixMs = longGeneric.ExpiresAtUnixMs
	longGeneric.GrantKind = GrantKindProductChild
	longGeneric.ProductTargetKind = ProductTargetStart
	longGenericGrant, err := (GrantCodec{Now: func() time.Time { return now }}).Sign(longGeneric, privateKey)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := VerifyProductDelegationGrant(longGenericGrant, publicKey, now, productDelegationBinding(claims)); !errors.Is(err, ErrProductDelegationBinding) {
		t.Fatalf("long-lived generic grant accepted as Product child: %v", err)
	}
	rootProductFields := claims
	rootProductFields.GrantKind = GrantKindRoot
	rootProductFields.ProductTargetKind = ""
	rootProductFields.ChildOperationID = ""
	rootProductGrant, err := (GrantCodec{Now: func() time.Time { return now }}).Sign(rootProductFields, privateKey)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := VerifyProductDelegationGrant(rootProductGrant, publicKey, now, productDelegationBinding(claims)); !errors.Is(err, ErrProductDelegationBinding) {
		t.Fatalf("generic root grant accepted at Product boundary: %v", err)
	}
	longLived := claims
	longLived.ExpiresAtUnixMs = longLived.IssuedAtUnixMs + DefaultProductDelegationTTL.Milliseconds() + 1
	longLived.EntryDeadlineUnixMs = longLived.ExpiresAtUnixMs
	if _, err := (GrantCodec{Now: func() time.Time { return now }}).SignProductDelegationGrant(longLived, privateKey); err == nil {
		t.Fatal("Product child delegation exceeded two-minute TTL")
	}
}

func TestVerifyProductDelegationParentBindsParentRootBeforeExchange(t *testing.T) {
	publicKey, privateKey := testGrantKeys()
	now := time.UnixMilli(1_700_000_100_000)
	parent := testGrantClaims()
	parent.EntryDeadlineUnixMs = parent.ExpiresAtUnixMs
	grant, err := (GrantCodec{Now: func() time.Time { return now }}).Sign(parent, privateKey)
	if err != nil {
		t.Fatal(err)
	}
	claims, err := VerifyProductDelegationParent(grant, publicKey, now, RootGrantBinding{
		CallContext:       testProductCallContext(),
		RootOperationID:   parent.RootOperationID,
		OwnerID:           parent.OwnerID,
		AccountGeneration: parent.AccountGeneration,
		RootRequestDigest: parent.RootRequestDigest,
		RequiredScopes:    []string{"contacts:read"},
	})
	if err != nil || claims.RootCapabilityID != parent.RootCapabilityID {
		t.Fatalf("parent Agent grant rejected before exchange: %#v %v", claims, err)
	}
	wrong := parent
	wrong.RootRequestDigest = bytes.Repeat([]byte{0x44}, sha256.Size)
	wrongGrant, err := (GrantCodec{Now: func() time.Time { return now }}).Sign(wrong, privateKey)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := VerifyProductDelegationParent(wrongGrant, publicKey, now, RootGrantBinding{
		CallContext: testProductCallContext(), RootOperationID: parent.RootOperationID,
		OwnerID: parent.OwnerID, AccountGeneration: parent.AccountGeneration,
		RootRequestDigest: parent.RootRequestDigest, RequiredScopes: []string{"contacts:read"},
	}); err == nil {
		t.Fatal("parent root digest substitution accepted")
	}
}

func TestProductDelegationExchangeValidation(t *testing.T) {
	rootDigest := bytes.Repeat([]byte{0x31}, sha256.Size)
	parent := &PermissionContext{AuthenticatedOwnerId: "owner-123", AccountGeneration: 7, GrantedScopes: []string{"contacts:read"}, CapabilityGrant: []byte(GrantPrefix + ".opaque"), RootRequestDigest: rootDigest}
	request := &ExchangeProductDelegationRequest{
		CallContext:      &CallContext{ChainId: testGrantChain, RootOperationId: testGrantRoot, Hop: 2, Route: NodeMessage + RouteSeparator + NodeAgent, DeadlineUnixMs: 1_700_000_500_000},
		ParentPermission: parent, ChildOperationId: "", CapabilityId: "product.contacts.v1", Operation: "list", RequestJson: []byte(`{"limit":10}`), TargetKind: ExchangeProductTargetKind_EXCHANGE_PRODUCT_TARGET_KIND_QUERY,
	}
	if err := ValidateProductDelegationExchangeRequest(request); err != nil {
		t.Fatalf("valid exchange rejected: %v", err)
	}
	request.ChildOperationId = "018f2f1e-7b5b-7b21-8f2e-4a6f3b1d9c12"
	if err := ValidateProductDelegationExchangeRequest(request); err == nil {
		t.Fatal("Query exchange accepted a child operation UUID")
	}
	request.ChildOperationId = ""
	request.TargetKind = ExchangeProductTargetKind_EXCHANGE_PRODUCT_TARGET_KIND_START_OPERATION
	if err := ValidateProductDelegationExchangeRequest(request); err == nil {
		t.Fatal("Start exchange accepted an empty child operation UUID")
	}
	request.ChildOperationId = testGrantRoot
	if err := ValidateProductDelegationExchangeRequest(request); err != nil {
		t.Fatalf("valid Start exchange rejected: %v", err)
	}
	request.TargetKind = ExchangeProductTargetKind_EXCHANGE_PRODUCT_TARGET_KIND_QUERY
	request.ChildOperationId = ""
	request.RequestJson = []byte(`{"limit": 10}`)
	if err := ValidateProductDelegationExchangeRequest(request); err == nil {
		t.Fatal("non-canonical exchange JSON accepted")
	}
	request.RequestJson = []byte(`{"limit":10}`)
	request.ParentPermission.CapabilityGrant = []byte(ControlGrantPrefix + ".control")
	if err := ValidateProductDelegationExchangeRequest(request); err == nil {
		t.Fatal("control grant accepted as parent exchange grant")
	}
	response := &ExchangeProductDelegationResponse{
		ProductPermission: &PermissionContext{AuthenticatedOwnerId: "owner-123", AccountGeneration: 7, GrantedScopes: []string{"contacts:read"}, CapabilityGrant: []byte(GrantPrefix + ".child"), RootRequestDigest: bytes.Repeat([]byte{0x32}, sha256.Size)},
		ExpiresAtUnixMs:   1_700_000_160_000,
	}
	if err := ValidateProductDelegationExchangeResponse(response, 1_700_000_100_000); err != nil {
		t.Fatalf("valid exchange response rejected: %v", err)
	}
	response.ExpiresAtUnixMs += DefaultProductDelegationTTL.Milliseconds()
	if err := ValidateProductDelegationExchangeResponse(response, 1_700_000_100_000); err == nil {
		t.Fatal("long-lived Product delegation response accepted")
	}
}

func TestSignProductDelegationFromParentDerivesExactIdentityAndTarget(t *testing.T) {
	publicKey, privateKey := testGrantKeys()
	now := time.UnixMilli(1_700_000_000_000)
	parent := testGrantClaims()
	parent.EntryDeadlineUnixMs = parent.ExpiresAtUnixMs
	callContext := testProductCallContext()
	callContext.DeadlineUnixMs = now.Add(90 * time.Second).UnixMilli()
	rootDigest := bytes.Repeat([]byte{0x91}, sha256.Size)
	catalogDigest := bytes.Repeat([]byte{0x92}, sha256.Size)
	schemaDigest := bytes.Repeat([]byte{0x93}, sha256.Size)
	childID := "018f2f1e-7b5b-7b21-8f2e-4a6f3b1d9c12"
	grant, err := (GrantCodec{Now: func() time.Time { return now }}).SignProductDelegationFromParent(ProductDelegationIssue{
		ParentClaims: parent, CallContext: callContext, ChildOperationID: childID,
		CapabilityID: "product.contacts.v1", Operation: "list", TargetKind: ExchangeProductTargetKind_EXCHANGE_PRODUCT_TARGET_KIND_START_OPERATION,
		RequiredScopes: []string{"contacts:read"}, RootRequestDigest: rootDigest, CatalogDigest: catalogDigest, SchemaDigest: schemaDigest,
	}, privateKey)
	if err != nil {
		t.Fatal(err)
	}
	claims, err := VerifyProductDelegationGrant(grant, publicKey, now, ProductGrantBinding{
		CallContext: callContext, RootOperationID: parent.RootOperationID, ChildOperationID: childID,
		OwnerID: parent.OwnerID, AccountGeneration: parent.AccountGeneration, RequiredScopes: []string{"contacts:read"},
		CapabilityID: "product.contacts.v1", Operation: "list", TargetKind: ExchangeProductTargetKind_EXCHANGE_PRODUCT_TARGET_KIND_START_OPERATION,
		RootRequestDigest: rootDigest, CatalogDigest: catalogDigest, SchemaDigest: schemaDigest,
	})
	if err != nil || claims.OwnerID != parent.OwnerID || claims.AccountGeneration != parent.AccountGeneration || claims.RootOperationID != parent.RootOperationID {
		t.Fatalf("derived Product grant lost parent identity: %#v %v", claims, err)
	}
	if !sameStringSlice(claims.Scopes, []string{"contacts:read"}) {
		t.Fatalf("derived Product grant did not narrow scopes: got=%v want=%v", claims.Scopes, []string{"contacts:read"})
	}
	if _, err := (GrantCodec{Now: func() time.Time { return now }}).SignProductDelegationFromParent(ProductDelegationIssue{
		ParentClaims: parent, CallContext: callContext, ChildOperationID: childID,
		CapabilityID: "product.contacts.v1", Operation: "list", TargetKind: ExchangeProductTargetKind_EXCHANGE_PRODUCT_TARGET_KIND_START_OPERATION,
		RequiredScopes: []string{"rooms:write"}, RootRequestDigest: rootDigest, CatalogDigest: catalogDigest, SchemaDigest: schemaDigest,
	}, privateKey); err == nil {
		t.Fatal("Product scope outside parent grant was accepted")
	}
	queryGrant, err := (GrantCodec{Now: func() time.Time { return now }}).SignProductDelegationFromParent(ProductDelegationIssue{
		ParentClaims: parent, CallContext: callContext, CapabilityID: "product.contacts.v1", Operation: "list",
		TargetKind: ExchangeProductTargetKind_EXCHANGE_PRODUCT_TARGET_KIND_QUERY, RequiredScopes: []string{"contacts:read"},
		RootRequestDigest: rootDigest, CatalogDigest: catalogDigest, SchemaDigest: schemaDigest,
	}, privateKey)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := VerifyProductDelegationGrant(queryGrant, publicKey, now, ProductGrantBinding{
		CallContext: callContext, RootOperationID: parent.RootOperationID, OwnerID: parent.OwnerID,
		AccountGeneration: parent.AccountGeneration, RequiredScopes: []string{"contacts:read"},
		CapabilityID: "product.contacts.v1", Operation: "list", TargetKind: ExchangeProductTargetKind_EXCHANGE_PRODUCT_TARGET_KIND_QUERY,
		RootRequestDigest: rootDigest, CatalogDigest: catalogDigest, SchemaDigest: schemaDigest,
	}); err != nil {
		t.Fatalf("Query child without UUID rejected: %v", err)
	}
}

func TestProductDelegationDigestEqual(t *testing.T) {
	claims := productDelegationClaims()
	if err := ProductDelegationDigestEqual(claims, claims.RootRequestDigest); err != nil {
		t.Fatalf("matching Product digest rejected: %v", err)
	}
	if !errors.Is(ProductDelegationDigestEqual(claims, bytes.Repeat([]byte{0x55}, sha256.Size)), ErrProductDelegationBinding) {
		t.Fatal("different Product digest accepted")
	}
}
