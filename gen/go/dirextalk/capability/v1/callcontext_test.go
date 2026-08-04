package capabilityv1

import (
	"testing"
	"time"
)

func TestValidateCallContext(t *testing.T) {
	now := time.Now().UnixMilli()
	deadline := now + 30000 // 30 seconds

	tests := []struct {
		name    string
		ctx     *CallContext
		wantErr bool
	}{
		{
			name: "valid context",
			ctx: &CallContext{
				ChainId:         "chain-123",
				RootOperationId: "op-456",
				Hop:             0,
				Route:           "",
				DeadlineUnixMs:  deadline,
			},
			wantErr: false,
		},
		{
			name:    "nil context",
			ctx:     nil,
			wantErr: true,
		},
		{
			name: "missing chain_id",
			ctx: &CallContext{
				RootOperationId: "op-456",
				DeadlineUnixMs:  deadline,
			},
			wantErr: true,
		},
		{
			name: "missing root_operation_id",
			ctx: &CallContext{
				ChainId:        "chain-123",
				DeadlineUnixMs: deadline,
			},
			wantErr: true,
		},
		{
			name: "negative hop",
			ctx: &CallContext{
				ChainId:         "chain-123",
				RootOperationId: "op-456",
				Hop:             -1,
				DeadlineUnixMs:  deadline,
			},
			wantErr: true,
		},
		{
			name: "hop exceeds maximum",
			ctx: &CallContext{
				ChainId:         "chain-123",
				RootOperationId: "op-456",
				Hop:             MaxCallHop + 1,
				Route:           "ms→agent→product",
				DeadlineUnixMs:  deadline,
			},
			wantErr: true,
		},
		{
			name: "invalid deadline",
			ctx: &CallContext{
				ChainId:         "chain-123",
				RootOperationId: "op-456",
				Hop:             0,
				DeadlineUnixMs:  0,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateCallContext(tt.ctx)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateCallContext() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestIncrementHop(t *testing.T) {
	now := time.Now().UnixMilli()
	deadline := now + 30000

	tests := []struct {
		name        string
		ctx         *CallContext
		currentNode string
		wantHop     int32
		wantRoute   string
		wantErr     bool
	}{
		{
			name: "first hop",
			ctx: &CallContext{
				ChainId:         "chain-123",
				RootOperationId: "op-456",
				Hop:             0,
				Route:           "",
				DeadlineUnixMs:  deadline,
			},
			currentNode: "ms",
			wantHop:     1,
			wantRoute:   "ms",
			wantErr:     false,
		},
		{
			name: "second hop",
			ctx: &CallContext{
				ChainId:         "chain-123",
				RootOperationId: "op-456",
				Hop:             1,
				Route:           "ms",
				DeadlineUnixMs:  deadline,
			},
			currentNode: "agent",
			wantHop:     2,
			wantRoute:   "ms→agent",
			wantErr:     false,
		},
		{
			name: "exceeds maximum",
			ctx: &CallContext{
				ChainId:         "chain-123",
				RootOperationId: "op-456",
				Hop:             3,
				Route:           "ms→agent→product",
				DeadlineUnixMs:  deadline,
			},
			currentNode: "agent",
			wantErr:     true,
		},
		{
			name:        "nil context",
			ctx:         nil,
			currentNode: "agent",
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			newCtx, err := IncrementHop(tt.ctx, tt.currentNode)
			if (err != nil) != tt.wantErr {
				t.Errorf("IncrementHop() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if newCtx.Hop != tt.wantHop {
					t.Errorf("IncrementHop() hop = %d, want %d", newCtx.Hop, tt.wantHop)
				}
				if newCtx.Route != tt.wantRoute {
					t.Errorf("IncrementHop() route = %s, want %s", newCtx.Route, tt.wantRoute)
				}
			}
		})
	}
}

func TestDetectCycle(t *testing.T) {
	tests := []struct {
		name    string
		route   string
		wantErr bool
	}{
		{
			name:    "empty route",
			route:   "",
			wantErr: false,
		},
		{
			name:    "simple path",
			route:   "ms→agent→product",
			wantErr: false,
		},
		{
			name:    "back and forth is a cycle",
			route:   "agent→ms→agent→ms",
			wantErr: true,
		},
		{
			name:    "cycle detected - three times",
			route:   "agent→ms→agent→ms→agent",
			wantErr: true,
		},
		{
			name:    "cycle detected - ms three times",
			route:   "ms→agent→ms→agent→ms",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := DetectCycle(tt.route)
			if (err != nil) != tt.wantErr {
				t.Errorf("DetectCycle() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateCallPath(t *testing.T) {
	now := time.Now().UnixMilli()
	deadline := now + 30000

	tests := []struct {
		name        string
		ctx         *CallContext
		currentNode string
		wantErr     bool
	}{
		{
			name: "valid first call",
			ctx: &CallContext{
				ChainId:         "chain-123",
				RootOperationId: "op-456",
				Hop:             0,
				Route:           "",
				DeadlineUnixMs:  deadline,
			},
			currentNode: "agent",
			wantErr:     false,
		},
		{
			name: "valid message server to agent",
			ctx: &CallContext{
				ChainId:         "chain-123",
				RootOperationId: "op-456",
				Hop:             1,
				Route:           "ms",
				DeadlineUnixMs:  deadline,
			},
			currentNode: "agent",
			wantErr:     false,
		},
		{
			name: "terminal node forwarding",
			ctx: &CallContext{
				ChainId:         "chain-123",
				RootOperationId: "op-456",
				Hop:             1,
				Route:           "product",
				DeadlineUnixMs:  deadline,
			},
			currentNode: "agent",
			wantErr:     true,
		},
		{
			name: "cycle in route",
			ctx: &CallContext{
				ChainId:         "chain-123",
				RootOperationId: "op-456",
				Hop:             2,
				Route:           "agent→ms→agent→ms",
				DeadlineUnixMs:  deadline,
			},
			currentNode: "agent",
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateCallPath(tt.ctx, tt.currentNode)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateCallPath() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestRemainingDeadlineMs(t *testing.T) {
	now := time.Now().UnixMilli()

	tests := []struct {
		name      string
		ctx       *CallContext
		nowUnixMs int64
		want      int64
	}{
		{
			name: "30 seconds remaining",
			ctx: &CallContext{
				DeadlineUnixMs: now + 30000,
			},
			nowUnixMs: now,
			want:      30000,
		},
		{
			name: "deadline passed",
			ctx: &CallContext{
				DeadlineUnixMs: now - 1000,
			},
			nowUnixMs: now,
			want:      0,
		},
		{
			name:      "nil context",
			ctx:       nil,
			nowUnixMs: now,
			want:      0,
		},
		{
			name: "zero deadline",
			ctx: &CallContext{
				DeadlineUnixMs: 0,
			},
			nowUnixMs: now,
			want:      0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := RemainingDeadlineMs(tt.ctx, tt.nowUnixMs)
			if got != tt.want {
				t.Errorf("RemainingDeadlineMs() = %d, want %d", got, tt.want)
			}
		})
	}
}
