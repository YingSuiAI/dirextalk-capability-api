package capabilityv1

import (
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"strings"
)

const (
	// CapabilityAuthorizationMetadata is the single canonical gRPC metadata
	// key. Both private directions use this key and scheme.
	CapabilityAuthorizationMetadata = "authorization"
	CapabilityInstanceMetadata      = "x-dirextalk-instance-id"
	CapabilityGenerationMetadata    = "x-dirextalk-account-generation"
	CapabilityTokenScheme           = "DTX-Capability-Token"
	CapabilityTokenBytes            = 32
	CapabilityTokenLength           = 43 // unpadded base64url(32 bytes)
)

var ErrInvalidCapabilityMetadata = errors.New("invalid capability metadata")

// FormatCapabilityToken formats a canonical unpadded base64url token for the
// authorization metadata value. The token must encode exactly 32 random bytes;
// the secret never appears in protobuf fields.
func FormatCapabilityToken(token string) (string, error) {
	if err := validateCapabilityToken(token); err != nil {
		return "", err
	}
	return CapabilityTokenScheme + " " + token, nil
}

// EncodeCapabilityToken encodes raw 32-byte token material for configuration
// and tests without permitting alternate textual encodings.
func EncodeCapabilityToken(raw []byte) (string, error) {
	if len(raw) != CapabilityTokenBytes {
		return "", fmt.Errorf("%w: token must be %d bytes", ErrInvalidCapabilityMetadata, CapabilityTokenBytes)
	}
	return base64.RawURLEncoding.EncodeToString(raw), nil
}

// ParseCapabilityToken parses the canonical authorization value. Schemes,
// whitespace, and token delimiters are intentionally strict to avoid accepting
// ambiguous values through different gRPC stacks.
func ParseCapabilityToken(value string) (string, error) {
	if strings.Count(value, " ") != 1 {
		return "", fmt.Errorf("%w: authorization must be '<scheme> <token>'", ErrInvalidCapabilityMetadata)
	}
	parts := strings.SplitN(value, " ", 2)
	if parts[0] != CapabilityTokenScheme {
		return "", fmt.Errorf("%w: unsupported authorization scheme", ErrInvalidCapabilityMetadata)
	}
	if err := validateCapabilityToken(parts[1]); err != nil {
		return "", err
	}
	return parts[1], nil
}

// CapabilityMetadata is the normalized peer-authentication metadata needed by
// both private services.
type CapabilityMetadata struct {
	Token             string
	InstanceID        string
	AccountGeneration int64
}

// FormatCapabilityMetadata returns canonical metadata values for outgoing
// gRPC calls.
func FormatCapabilityMetadata(token, instanceID string, accountGeneration int64) (map[string]string, error) {
	authorization, err := FormatCapabilityToken(token)
	if err != nil {
		return nil, err
	}
	if err := validateInstanceID(instanceID); err != nil {
		return nil, err
	}
	if accountGeneration <= 0 {
		return nil, fmt.Errorf("%w: account generation must be positive", ErrInvalidCapabilityMetadata)
	}
	return map[string]string{
		CapabilityAuthorizationMetadata: authorization,
		CapabilityInstanceMetadata:      instanceID,
		CapabilityGenerationMetadata:    strconv.FormatInt(accountGeneration, 10),
	}, nil
}

// ParseCapabilityMetadata validates canonical metadata received from a peer.
// The input map uses lowercase metadata keys; callers adapting grpc.Metadata
// should pass md.Get(key) values and preserve duplicate-value rejection.
func ParseCapabilityMetadata(values map[string][]string) (CapabilityMetadata, error) {
	if values == nil {
		return CapabilityMetadata{}, fmt.Errorf("%w: metadata is missing", ErrInvalidCapabilityMetadata)
	}
	getOne := func(key string) (string, error) {
		items := values[key]
		if len(items) != 1 || strings.TrimSpace(items[0]) != items[0] || items[0] == "" {
			return "", fmt.Errorf("%w: metadata %s must have exactly one value", ErrInvalidCapabilityMetadata, key)
		}
		return items[0], nil
	}
	authorization, err := getOne(CapabilityAuthorizationMetadata)
	if err != nil {
		return CapabilityMetadata{}, err
	}
	token, err := ParseCapabilityToken(authorization)
	if err != nil {
		return CapabilityMetadata{}, err
	}
	instanceID, err := getOne(CapabilityInstanceMetadata)
	if err != nil {
		return CapabilityMetadata{}, err
	}
	if err := validateInstanceID(instanceID); err != nil {
		return CapabilityMetadata{}, err
	}
	generationText, err := getOne(CapabilityGenerationMetadata)
	if err != nil {
		return CapabilityMetadata{}, err
	}
	if generationText == "0" || (len(generationText) > 1 && generationText[0] == '0') || strings.HasPrefix(generationText, "+") || strings.HasPrefix(generationText, "-") {
		return CapabilityMetadata{}, fmt.Errorf("%w: account generation must be canonical positive decimal", ErrInvalidCapabilityMetadata)
	}
	generation, err := strconv.ParseInt(generationText, 10, 64)
	if err != nil || generation <= 0 {
		return CapabilityMetadata{}, fmt.Errorf("%w: invalid account generation", ErrInvalidCapabilityMetadata)
	}
	return CapabilityMetadata{Token: token, InstanceID: instanceID, AccountGeneration: generation}, nil
}

func validateCapabilityToken(token string) error {
	if len(token) != CapabilityTokenLength || strings.TrimSpace(token) != token {
		return fmt.Errorf("%w: token is invalid", ErrInvalidCapabilityMetadata)
	}
	decoded, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil || len(decoded) != CapabilityTokenBytes || base64.RawURLEncoding.EncodeToString(decoded) != token {
		return fmt.Errorf("%w: token must be canonical unpadded base64url", ErrInvalidCapabilityMetadata)
	}
	return nil
}

func validateInstanceID(instanceID string) error {
	if err := ValidateOperationID(instanceID); err != nil {
		return fmt.Errorf("%w: instance id must be a canonical UUID: %v", ErrInvalidCapabilityMetadata, err)
	}
	return nil
}
