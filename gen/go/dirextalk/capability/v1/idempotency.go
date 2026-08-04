package capabilityv1

// This file contains the non-generated helpers that are part of the public
// capability API.  They intentionally live next to the generated protobuf
// package so both services use exactly the same digest implementation.

import (
	"bytes"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"math/big"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"unicode/utf16"
	"unicode/utf8"
)

const sha256DigestSize = sha256.Size

// ComputeRequestDigest computes the SHA-256 digest used by StartOperation.
//
// The digest input is encoded as RFC 8785 JSON.  Digest fields are deliberately
// assembled as a map rather than a Go struct so the wire names are explicit and
// cannot change when a Go field is renamed.  The business input is treated as
// JSON data; map key ordering, number formatting, and string escaping are all
// canonicalized before hashing.
func ComputeRequestDigest(
	protocolMajor int32,
	capabilityID string,
	capabilityVersion string,
	schemaDigest []byte,
	operation string,
	expectedRevision int64,
	businessInput map[string]interface{},
	attachmentRefDigests [][]byte,
	permissionGrantDigest []byte,
) ([]byte, error) {
	return computeRequestDigest(protocolMajor, capabilityID, capabilityVersion, schemaDigest, operation, expectedRevision, businessInput, attachmentRefDigests, permissionGrantDigest, true)
}

func computeRequestDigest(
	protocolMajor int32,
	capabilityID string,
	capabilityVersion string,
	schemaDigest []byte,
	operation string,
	expectedRevision int64,
	businessInput map[string]interface{},
	attachmentRefDigests [][]byte,
	permissionGrantDigest []byte,
	includeGrant bool,
) ([]byte, error) {
	if err := ValidateSHA256Digest(schemaDigest); err != nil {
		return nil, fmt.Errorf("schema digest: %w", err)
	}
	if includeGrant {
		if err := ValidateSHA256Digest(permissionGrantDigest); err != nil {
			return nil, fmt.Errorf("permission grant digest: %w", err)
		}
	}
	for i, attachment := range attachmentRefDigests {
		if err := ValidateSHA256Digest(attachment); err != nil {
			return nil, fmt.Errorf("attachment digest %d: %w", i, err)
		}
	}
	digestInput := map[string]interface{}{
		"protocol_major":         protocolMajor,
		"capability_id":          capabilityID,
		"capability_version":     capabilityVersion,
		"schema_digest":          schemaDigest,
		"operation":              operation,
		"expected_revision":      expectedRevision,
		"business_input":         businessInput,
		"attachment_ref_digests": attachmentRefDigests,
	}
	if includeGrant {
		digestInput["permission_grant_digest"] = permissionGrantDigest
	}

	canonicalJSON, err := Canonicalize(digestInput)
	if err != nil {
		return nil, fmt.Errorf("canonicalize request digest input: %w", err)
	}
	hash := sha256.Sum256(canonicalJSON)
	return hash[:], nil
}

// ComputeRootRequestDigest computes the grant-independent digest that
// message-server signs into GrantClaims.RootRequestDigest. The final
// StartOperation request digest is computed separately with
// ComputeRequestDigest after the grant exists, avoiding a circular digest.
// Owner, generation and scopes are signed as distinct GrantClaims fields.
func ComputeRootRequestDigest(
	protocolMajor int32,
	capabilityID string,
	capabilityVersion string,
	schemaDigest []byte,
	operation string,
	expectedRevision int64,
	businessInput map[string]interface{},
	attachmentRefDigests [][]byte,
) ([]byte, error) {
	return computeRequestDigest(
		protocolMajor,
		capabilityID,
		capabilityVersion,
		schemaDigest,
		operation,
		expectedRevision,
		businessInput,
		attachmentRefDigests,
		nil,
		false,
	)
}

// Canonicalize returns the RFC 8785 (JCS) serialization of a JSON-compatible
// Go value.  It rejects unsupported values, non-finite numbers, invalid UTF-8,
// duplicate object names, and lone UTF-16 surrogates.
func Canonicalize(value interface{}) ([]byte, error) {
	return canonicalizeValue(reflect.ValueOf(value), make(map[uintptr]bool))
}

// toCanonicalJSON is retained for package-local conformance tests and for
// source compatibility with the initial v1 helper implementation.
func toCanonicalJSON(value interface{}) (string, error) {
	canonical, err := Canonicalize(value)
	if err != nil {
		return "", err
	}
	return string(canonical), nil
}

// CanonicalizeJSON parses and canonicalizes one JSON value.  It is useful when
// request_json arrived as bytes and avoids the precision loss caused by the
// default encoding/json float64 decoder.
func CanonicalizeJSON(data []byte) ([]byte, error) {
	if len(bytes.TrimSpace(data)) == 0 {
		return nil, errors.New("json input is empty")
	}
	if err := validateJSONStrings(data); err != nil {
		return nil, err
	}
	value, err := decodeJSON(data)
	if err != nil {
		return nil, err
	}
	return canonicalizeValue(reflect.ValueOf(value), make(map[uintptr]bool))
}

// ParseBusinessInput parses a JSON object while retaining numbers as
// json.Number.  Callers must not convert the returned numbers to float64 when
// they will be included in a request digest.
func ParseBusinessInput(canonicalJSON []byte) (map[string]interface{}, error) {
	canonical, err := CanonicalizeJSON(canonicalJSON)
	if err != nil {
		return nil, fmt.Errorf("parse business input: %w", err)
	}
	var result map[string]interface{}
	dec := json.NewDecoder(bytes.NewReader(canonical))
	dec.UseNumber()
	if err := dec.Decode(&result); err != nil {
		return nil, fmt.Errorf("business input must be a JSON object: %w", err)
	}
	if result == nil {
		return nil, errors.New("business input must be a JSON object")
	}
	return result, nil
}

// SerializeBusinessInput returns canonical JSON bytes for an object input.
func SerializeBusinessInput(input map[string]interface{}) ([]byte, error) {
	if input == nil {
		return []byte("null"), nil
	}
	return Canonicalize(input)
}

// ValidateDigest compares two digests in constant time.  It accepts any
// non-empty digest length for compatibility with callers that compare a
// capability-specific digest.  Use ValidateSHA256Digest for request digests.
func ValidateDigest(actual, expected []byte) error {
	if len(actual) != len(expected) {
		return fmt.Errorf("digest length mismatch: got %d, want %d", len(actual), len(expected))
	}
	if len(actual) == 0 || subtle.ConstantTimeCompare(actual, expected) != 1 {
		return errors.New("digest mismatch")
	}
	return nil
}

// ValidateSHA256Digest validates the shape of a SHA-256 digest supplied on the
// wire.  It does not compare the digest to a request; use ValidateDigest for
// that comparison after recomputing the expected value.
func ValidateSHA256Digest(digest []byte) error {
	if len(digest) != sha256DigestSize {
		return fmt.Errorf("sha-256 digest must be %d bytes, got %d", sha256DigestSize, len(digest))
	}
	return nil
}

// ValidateRequestDigest is the operation-request spelling of
// ValidateSHA256Digest and is kept separate to make call sites self-documenting.
func ValidateRequestDigest(digest []byte) error {
	return ValidateSHA256Digest(digest)
}

// CanonicalJSONEqual reports whether data is already in RFC 8785 form.
func CanonicalJSONEqual(data []byte) (bool, error) {
	canonical, err := CanonicalizeJSON(data)
	if err != nil {
		return false, err
	}
	return bytes.Equal(canonical, data), nil
}

func canonicalizeValue(value reflect.Value, visiting map[uintptr]bool) ([]byte, error) {
	if !value.IsValid() {
		return []byte("null"), nil
	}
	for value.Kind() == reflect.Interface || value.Kind() == reflect.Pointer {
		if value.IsNil() {
			return []byte("null"), nil
		}
		if value.Kind() == reflect.Pointer {
			ptr := value.Pointer()
			if ptr != 0 {
				if visiting[ptr] {
					return nil, errors.New("cyclic JSON value")
				}
				visiting[ptr] = true
				defer delete(visiting, ptr)
			}
		}
		value = value.Elem()
	}
	if value.Kind() == reflect.Map || value.Kind() == reflect.Slice {
		if !value.IsNil() {
			ptr := value.Pointer()
			if ptr != 0 {
				if visiting[ptr] {
					return nil, errors.New("cyclic JSON value")
				}
				visiting[ptr] = true
				defer delete(visiting, ptr)
			}
		}
	}

	// json.Number is a string alias and needs number semantics, not a JSON
	// string.  json.RawMessage is handled before the byte-slice case below.
	if value.CanInterface() {
		switch typed := value.Interface().(type) {
		case json.Number:
			return canonicalNumber(typed.String())
		case json.RawMessage:
			return CanonicalizeJSON(typed)
		case json.Marshaler:
			raw, err := typed.MarshalJSON()
			if err != nil {
				return nil, err
			}
			return CanonicalizeJSON(raw)
		}
	}

	switch value.Kind() {
	case reflect.Bool:
		if value.Bool() {
			return []byte("true"), nil
		}
		return []byte("false"), nil
	case reflect.String:
		return canonicalString(value.String())
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return canonicalNumber(strconv.FormatInt(value.Int(), 10))
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return canonicalNumber(strconv.FormatUint(value.Uint(), 10))
	case reflect.Float32, reflect.Float64:
		return canonicalFloat(value.Float())
	case reflect.Slice:
		if value.IsNil() {
			return []byte("null"), nil
		}
		if value.Type().Elem().Kind() == reflect.Uint8 {
			encoded := base64.StdEncoding.EncodeToString(value.Bytes())
			return canonicalString(encoded)
		}
		return canonicalArray(value, visiting)
	case reflect.Array:
		if value.Type().Elem().Kind() == reflect.Uint8 {
			buf := make([]byte, value.Len())
			for i := range buf {
				buf[i] = byte(value.Index(i).Uint())
			}
			return canonicalString(base64.StdEncoding.EncodeToString(buf))
		}
		return canonicalArray(value, visiting)
	case reflect.Map:
		return canonicalMap(value, visiting)
	case reflect.Struct:
		// Struct tags and custom marshalers are delegated to encoding/json, then
		// canonicalized. This preserves the standard JSON field-selection rules.
		raw, err := json.Marshal(value.Interface())
		if err != nil {
			return nil, err
		}
		return CanonicalizeJSON(raw)
	default:
		return nil, fmt.Errorf("unsupported JSON value type %s", value.Type())
	}
}

func canonicalArray(value reflect.Value, visiting map[uintptr]bool) ([]byte, error) {
	var out bytes.Buffer
	out.WriteByte('[')
	for i := 0; i < value.Len(); i++ {
		if i > 0 {
			out.WriteByte(',')
		}
		item, err := canonicalizeValue(value.Index(i), visiting)
		if err != nil {
			return nil, err
		}
		out.Write(item)
	}
	out.WriteByte(']')
	return out.Bytes(), nil
}

func canonicalMap(value reflect.Value, visiting map[uintptr]bool) ([]byte, error) {
	if value.Type().Key().Kind() != reflect.String {
		return nil, fmt.Errorf("JSON object keys must be strings, got %s", value.Type().Key())
	}
	type member struct {
		key   string
		value reflect.Value
	}
	members := make([]member, 0, value.Len())
	iter := value.MapRange()
	for iter.Next() {
		key := iter.Key().String()
		if !utf8.ValidString(key) {
			return nil, errors.New("JSON object key is not valid UTF-8")
		}
		members = append(members, member{key: key, value: iter.Value()})
	}
	sort.Slice(members, func(i, j int) bool { return compareUTF16(members[i].key, members[j].key) < 0 })
	var out bytes.Buffer
	out.WriteByte('{')
	for i, item := range members {
		if i > 0 {
			out.WriteByte(',')
		}
		key, err := canonicalString(item.key)
		if err != nil {
			return nil, err
		}
		val, err := canonicalizeValue(item.value, visiting)
		if err != nil {
			return nil, err
		}
		out.Write(key)
		out.WriteByte(':')
		out.Write(val)
	}
	out.WriteByte('}')
	return out.Bytes(), nil
}

func canonicalString(value string) ([]byte, error) {
	if !utf8.ValidString(value) {
		return nil, errors.New("JSON string is not valid UTF-8")
	}
	var buf bytes.Buffer
	buf.WriteByte('"')
	for _, r := range value {
		switch r {
		case '"':
			buf.WriteString(`\"`)
		case '\\':
			buf.WriteString(`\\`)
		case '\b':
			buf.WriteString(`\b`)
		case '\f':
			buf.WriteString(`\f`)
		case '\n':
			buf.WriteString(`\n`)
		case '\r':
			buf.WriteString(`\r`)
		case '\t':
			buf.WriteString(`\t`)
		default:
			if r < 0x20 {
				fmt.Fprintf(&buf, `\u%04x`, r)
				continue
			}
			// RFC 8785 requires the shortest valid JSON escape. In particular,
			// U+2028 and U+2029 remain literal UTF-8 (encoding/json escapes them
			// even when SetEscapeHTML(false) is used).
			buf.WriteRune(r)
		}
	}
	buf.WriteByte('"')
	return buf.Bytes(), nil
}

func canonicalFloat(value float64) ([]byte, error) {
	if math.IsNaN(value) || math.IsInf(value, 0) {
		return nil, errors.New("JSON numbers must be finite")
	}
	return canonicalNumber(strconv.FormatFloat(value, 'g', -1, 64))
}

// canonicalNumber formats an IEEE-754 number according to the ECMAScript
// thresholds required by RFC 8785: decimal notation for 1e-6 <= |n| < 1e21,
// scientific notation otherwise, with a lower-case e and no padded exponent.
func canonicalNumber(input string) ([]byte, error) {
	input = strings.TrimSpace(input)
	if input == "" {
		return nil, errors.New("empty JSON number")
	}
	if isIntegerLexeme(input) {
		var integer big.Int
		if _, ok := integer.SetString(input, 10); !ok {
			return nil, fmt.Errorf("invalid JSON integer %q", input)
		}
		// JSON numbers are parsed as IEEE-754 values by RFC 8785 consumers.
		// Permit large integers only when the conversion is exact; this keeps
		// canonical output such as 1e17 -> 100000000000000000 idempotent while
		// rejecting 9007199254740993, which would silently round.
		asFloat, _ := new(big.Float).SetInt(&integer).Float64()
		if math.IsInf(asFloat, 0) {
			return nil, fmt.Errorf("integer %q is outside IEEE-754 range", input)
		}
		rounded, _ := new(big.Float).SetFloat64(asFloat).Int(nil)
		if rounded == nil || rounded.Cmp(&integer) != 0 {
			return nil, fmt.Errorf("integer %q is not exactly representable as IEEE-754", input)
		}
	}
	value, err := strconv.ParseFloat(input, 64)
	if err != nil || math.IsNaN(value) || math.IsInf(value, 0) {
		if err != nil {
			return nil, fmt.Errorf("invalid JSON number %q: %w", input, err)
		}
		return nil, fmt.Errorf("JSON number %q is not finite", input)
	}
	if value == 0 {
		return []byte("0"), nil
	}

	formatted := strconv.FormatFloat(value, 'e', -1, 64)
	sign := ""
	if formatted[0] == '-' {
		sign = "-"
		formatted = formatted[1:]
	}
	ePos := strings.IndexByte(formatted, 'e')
	if ePos < 0 {
		return nil, fmt.Errorf("internal number formatting error for %q", input)
	}
	mantissa := formatted[:ePos]
	exponent, err := strconv.Atoi(formatted[ePos+1:])
	if err != nil {
		return nil, fmt.Errorf("internal exponent formatting error for %q: %w", input, err)
	}
	digits := strings.ReplaceAll(mantissa, ".", "")
	if exponent >= -6 && exponent < 21 {
		decimalPos := exponent + 1
		var body string
		switch {
		case decimalPos <= 0:
			body = "0." + strings.Repeat("0", -decimalPos) + digits
		case decimalPos >= len(digits):
			body = digits + strings.Repeat("0", decimalPos-len(digits))
		default:
			body = digits[:decimalPos] + "." + digits[decimalPos:]
		}
		return []byte(sign + body), nil
	}
	body := digits[:1]
	if len(digits) > 1 {
		body += "." + digits[1:]
	}
	if exponent < 0 {
		body += "e-" + strconv.Itoa(-exponent)
	} else {
		body += "e+" + strconv.Itoa(exponent)
	}
	return []byte(sign + body), nil
}

func isIntegerLexeme(input string) bool {
	if input == "" {
		return false
	}
	if input[0] == '-' {
		input = input[1:]
	}
	if input == "" {
		return false
	}
	for _, c := range input {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

func compareUTF16(a, b string) int {
	aa, bb := utf16.Encode([]rune(a)), utf16.Encode([]rune(b))
	for i := 0; i < len(aa) && i < len(bb); i++ {
		if aa[i] < bb[i] {
			return -1
		}
		if aa[i] > bb[i] {
			return 1
		}
	}
	switch {
	case len(aa) < len(bb):
		return -1
	case len(aa) > len(bb):
		return 1
	default:
		return 0
	}
}

func decodeJSON(data []byte) (interface{}, error) {
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()
	value, err := decodeJSONValue(dec)
	if err != nil {
		return nil, fmt.Errorf("decode JSON: %w", err)
	}
	var extra interface{}
	if err := dec.Decode(&extra); err != io.EOF {
		if err == nil {
			return nil, errors.New("JSON contains trailing data")
		}
		return nil, errors.New("JSON contains trailing data")
	}
	if extra != nil {
		return nil, errors.New("JSON contains trailing data")
	}
	return value, nil
}

func decodeJSONValue(dec *json.Decoder) (interface{}, error) {
	token, err := dec.Token()
	if err != nil {
		return nil, err
	}
	switch delimiter := token.(type) {
	case json.Delim:
		switch delimiter {
		case '{':
			object := make(map[string]interface{})
			for dec.More() {
				keyToken, err := dec.Token()
				if err != nil {
					return nil, err
				}
				key, ok := keyToken.(string)
				if !ok {
					return nil, errors.New("JSON object key is not a string")
				}
				if _, exists := object[key]; exists {
					return nil, fmt.Errorf("duplicate JSON object key %q", key)
				}
				value, err := decodeJSONValue(dec)
				if err != nil {
					return nil, err
				}
				object[key] = value
			}
			if _, err := dec.Token(); err != nil {
				return nil, err
			}
			return object, nil
		case '[':
			array := make([]interface{}, 0)
			for dec.More() {
				value, err := decodeJSONValue(dec)
				if err != nil {
					return nil, err
				}
				array = append(array, value)
			}
			if _, err := dec.Token(); err != nil {
				return nil, err
			}
			return array, nil
		default:
			return nil, fmt.Errorf("unexpected delimiter %q", delimiter)
		}
	default:
		return token, nil
	}
}

func validateJSONStrings(data []byte) error {
	for i := 0; i < len(data); {
		if data[i] != '"' {
			i++
			continue
		}
		next, err := validateJSONString(data, i)
		if err != nil {
			return err
		}
		i = next
	}
	return nil
}

func validateJSONString(data []byte, start int) (int, error) {
	for i := start + 1; i < len(data); {
		switch data[i] {
		case '"':
			return i + 1, nil
		case '\\':
			i++
			if i >= len(data) {
				return 0, errors.New("unterminated JSON escape")
			}
			if data[i] != 'u' {
				switch data[i] {
				case '"', '\\', '/', 'b', 'f', 'n', 'r', 't':
					i++
					continue
				default:
					return 0, fmt.Errorf("invalid JSON escape \\%c", data[i])
				}
			}
			code, next, err := readUnicodeEscape(data, i)
			if err != nil {
				return 0, err
			}
			i = next
			if code >= 0xD800 && code <= 0xDBFF {
				if i+6 > len(data) || data[i] != '\\' || data[i+1] != 'u' {
					return 0, errors.New("lone high surrogate in JSON string")
				}
				low, lowNext, err := readUnicodeEscape(data, i+1)
				if err != nil || low < 0xDC00 || low > 0xDFFF {
					return 0, errors.New("invalid UTF-16 surrogate pair in JSON string")
				}
				i = lowNext
			} else if code >= 0xDC00 && code <= 0xDFFF {
				return 0, errors.New("lone low surrogate in JSON string")
			}
		default:
			if data[i] < 0x20 {
				return 0, errors.New("unescaped control character in JSON string")
			}
			if data[i] < utf8.RuneSelf {
				i++
				continue
			}
			_, size := utf8.DecodeRune(data[i:])
			if size == 1 || !utf8.Valid(data[i:i+size]) {
				return 0, errors.New("invalid UTF-8 in JSON string")
			}
			i += size
		}
	}
	return 0, errors.New("unterminated JSON string")
}

func readUnicodeEscape(data []byte, backslashIndex int) (uint16, int, error) {
	// backslashIndex points at the backslash; the caller may pass the index of
	// the 'u' when it has already consumed the backslash.
	i := backslashIndex
	if data[i] == '\\' {
		i++
	}
	if i >= len(data) || data[i] != 'u' || i+5 > len(data) {
		return 0, 0, errors.New("invalid Unicode escape")
	}
	var code uint16
	for _, c := range data[i+1 : i+5] {
		code <<= 4
		switch {
		case c >= '0' && c <= '9':
			code |= uint16(c - '0')
		case c >= 'a' && c <= 'f':
			code |= uint16(c-'a') + 10
		case c >= 'A' && c <= 'F':
			code |= uint16(c-'A') + 10
		default:
			return 0, 0, errors.New("invalid Unicode escape digit")
		}
	}
	return code, i + 5, nil
}
