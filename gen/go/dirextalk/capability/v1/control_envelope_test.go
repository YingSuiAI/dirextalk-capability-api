package capabilityv1

import (
	"bytes"
	"testing"
	"time"

	"google.golang.org/protobuf/proto"
)

func validControlGrantEnvelopes(now int64) []*OperationControlGrantEnvelope {
	return []*OperationControlGrantEnvelope{
		{Action: "get", Grant: []byte(ControlGrantPrefix + ".get"), ExpiresAtUnixMs: now + time.Minute.Milliseconds()},
		{Action: "watch", Grant: []byte(ControlGrantPrefix + ".watch"), ExpiresAtUnixMs: now + time.Minute.Milliseconds()},
		{Action: "cancel", Grant: []byte(ControlGrantPrefix + ".cancel"), ExpiresAtUnixMs: now + time.Minute.Milliseconds()},
		{Action: "reconcile", Grant: []byte(ControlGrantPrefix + ".reconcile"), ExpiresAtUnixMs: now + time.Minute.Milliseconds()},
	}
}

func TestValidateOperationControlGrantEnvelopes(t *testing.T) {
	now := time.Now().UnixMilli()
	if err := ValidateOperationControlGrantEnvelopes(validControlGrantEnvelopes(now), now); err != nil {
		t.Fatalf("valid control grant envelopes rejected: %v", err)
	}
	if err := ValidateOperationControlGrantEnvelopes(nil, now); err != nil {
		t.Fatalf("optional empty control grants rejected: %v", err)
	}

	tests := []struct {
		name   string
		mutate func([]*OperationControlGrantEnvelope, int64)
	}{
		{name: "duplicate action", mutate: func(envelopes []*OperationControlGrantEnvelope, _ int64) {
			envelopes[1].Action = envelopes[0].Action
		}},
		{name: "invalid action", mutate: func(envelopes []*OperationControlGrantEnvelope, _ int64) {
			envelopes[0].Action = "delete"
		}},
		{name: "empty grant", mutate: func(envelopes []*OperationControlGrantEnvelope, _ int64) {
			envelopes[0].Grant = nil
		}},
		{name: "wrong grant domain", mutate: func(envelopes []*OperationControlGrantEnvelope, _ int64) {
			envelopes[0].Grant = []byte(GrantPrefix + ".root")
		}},
		{name: "expired", mutate: func(envelopes []*OperationControlGrantEnvelope, now int64) {
			envelopes[0].ExpiresAtUnixMs = now
		}},
		{name: "ttl too long", mutate: func(envelopes []*OperationControlGrantEnvelope, now int64) {
			envelopes[0].ExpiresAtUnixMs = now + DefaultControlGrantTTL.Milliseconds() + 1
		}},
		{name: "nil envelope", mutate: func(envelopes []*OperationControlGrantEnvelope, _ int64) {
			envelopes[0] = nil
		}},
		{name: "oversized grant", mutate: func(envelopes []*OperationControlGrantEnvelope, _ int64) {
			envelopes[0].Grant = append([]byte(ControlGrantPrefix+"."), bytes.Repeat([]byte{'x'}, DefaultGrantMaxSize)...)
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			envelopes := validControlGrantEnvelopes(now)
			test.mutate(envelopes, now)
			if err := ValidateOperationControlGrantEnvelopes(envelopes, now); err == nil {
				t.Fatal("invalid control grant envelope accepted")
			}
		})
	}
}

func TestValidateOperationControlGrantEnvelopesBounds(t *testing.T) {
	now := time.Now().UnixMilli()
	tooMany := validControlGrantEnvelopes(now)
	tooMany = append(tooMany, &OperationControlGrantEnvelope{
		Action: "get", Grant: []byte(ControlGrantPrefix + ".extra"), ExpiresAtUnixMs: now + time.Minute.Milliseconds(),
	})
	if err := ValidateOperationControlGrantEnvelopes(tooMany, now); err == nil {
		t.Fatal("more than four control grant envelopes accepted")
	}
	if err := ValidateOperationControlGrantEnvelopes(validControlGrantEnvelopes(now), 0); err == nil {
		t.Fatal("non-positive validation clock accepted")
	}
}

func TestStartOperationResponseControlGrantWireRoundTrip(t *testing.T) {
	now := time.Now().UnixMilli()
	response := &StartOperationResponse{
		OperationId: "018f2f1e-7b5b-7b21-8f2e-4a6f3b1d9c11",
		State:       OperationState_OPERATION_STATE_PENDING,
		ControlGrants: []*OperationControlGrantEnvelope{
			{Action: "watch", Grant: []byte(ControlGrantPrefix + ".payload.signature"), ExpiresAtUnixMs: now + time.Minute.Milliseconds()},
		},
	}
	wire, err := proto.Marshal(response)
	if err != nil {
		t.Fatalf("marshal response: %v", err)
	}
	decoded := new(StartOperationResponse)
	if err := proto.Unmarshal(wire, decoded); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if len(decoded.ControlGrants) != 1 || decoded.ControlGrants[0].Action != "watch" ||
		!bytes.Equal(decoded.ControlGrants[0].Grant, response.ControlGrants[0].Grant) ||
		decoded.ControlGrants[0].ExpiresAtUnixMs != response.ControlGrants[0].ExpiresAtUnixMs {
		t.Fatalf("control grant envelope did not round-trip: %#v", decoded.ControlGrants)
	}
	field := (&StartOperationResponse{}).ProtoReflect().Descriptor().Fields().ByName("control_grants")
	if field == nil || field.Number() != 4 || !field.IsList() {
		t.Fatalf("control_grants wire descriptor changed: %v", field)
	}
}
