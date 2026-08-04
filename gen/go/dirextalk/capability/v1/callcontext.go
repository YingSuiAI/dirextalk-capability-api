package capabilityv1

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

const (
	// MaxCallHop is the maximum number of routed nodes in a private capability
	// call. A direct call has hop 0 and an appended node increments it. The
	// supported private paths agent→product and ms→agent→product therefore
	// have hops 2 and 3 respectively.
	MaxCallHop = 3

	// MaxRouteLength bounds untrusted metadata copied into logs and traces.
	MaxRouteLength = 256

	RouteSeparator = "→"

	NodeMessage = "ms"
	NodeAgent   = "agent"
	NodeProduct = "product"
)

var (
	ErrCycleDetected   = errors.New("cycle detected")
	ErrInvalidCallPath = errors.New("invalid call path")
)

// ValidateCallContext validates the structural, non-authenticated part of a
// call context. Authentication of chain IDs and operation IDs remains a
// service responsibility; ValidateStrictCallContext applies the UUID rules
// when a service has completed that trust step.
func ValidateCallContext(ctx *CallContext) error {
	if ctx == nil {
		return errors.New("call_context is required")
	}
	if err := validateToken("chain_id", ctx.ChainId); err != nil {
		return err
	}
	if err := validateToken("root_operation_id", ctx.RootOperationId); err != nil {
		return err
	}
	if ctx.ParentCallId != "" {
		if err := validateToken("parent_call_id", ctx.ParentCallId); err != nil {
			return err
		}
	}
	if ctx.Hop < 0 {
		return fmt.Errorf("hop must be non-negative, got %d", ctx.Hop)
	}
	if ctx.Hop > MaxCallHop {
		return fmt.Errorf("hop %d exceeds maximum %d: %w", ctx.Hop, MaxCallHop, ErrCycleDetected)
	}
	if len(ctx.Route) > MaxRouteLength {
		return fmt.Errorf("route length %d exceeds maximum %d", len(ctx.Route), MaxRouteLength)
	}
	if ctx.DeadlineUnixMs <= 0 {
		return errors.New("deadline_unix_ms must be positive")
	}
	if ctx.Route == "" {
		if ctx.Hop != 0 {
			return fmt.Errorf("empty route must have hop 0")
		}
		return nil
	}
	nodes, err := parseRoute(ctx.Route)
	if err != nil {
		return err
	}
	if int32(len(nodes)) != ctx.Hop {
		return fmt.Errorf("route has %d nodes but hop is %d", len(nodes), ctx.Hop)
	}
	return DetectCycle(ctx.Route)
}

// ValidateStrictCallContext applies the UUID rules for trusted service
// boundaries. It is intentionally separate from ValidateCallContext so older
// callers can validate a context before they have an operation UUID available.
func ValidateStrictCallContext(ctx *CallContext) error {
	if err := ValidateCallContext(ctx); err != nil {
		return err
	}
	if err := ValidateOperationID(ctx.ChainId); err != nil {
		return fmt.Errorf("invalid chain_id: %w", err)
	}
	if err := ValidateOperationID(ctx.RootOperationId); err != nil {
		return fmt.Errorf("invalid root_operation_id: %w", err)
	}
	if ctx.ParentCallId != "" {
		if err := ValidateOperationID(ctx.ParentCallId); err != nil {
			return fmt.Errorf("invalid parent_call_id: %w", err)
		}
	}
	return nil
}

// AppendCallNode is the sender-side operation. The sender appends its own node
// before making the gRPC call. The receiver then validates the route and
// advances it with ValidateAndAdvanceCallContext.
func AppendCallNode(ctx *CallContext, senderNode string) (*CallContext, error) {
	if ctx == nil {
		return nil, errors.New("call_context is nil")
	}
	if ctx.Route != "" {
		nodes, err := parseRoute(ctx.Route)
		if err != nil {
			return nil, err
		}
		// A node is appended by the receiver when a hop is accepted. If that
		// service immediately becomes the sender for the next private call,
		// re-appending its own node is a no-op rather than a cycle.
		if nodes[len(nodes)-1] == senderNode {
			return cloneCallContext(ctx), nil
		}
	}
	if err := ValidateCallPath(ctx, senderNode); err != nil {
		return nil, err
	}
	newRoute := ctx.Route
	if newRoute == "" {
		newRoute = senderNode
	} else {
		newRoute += RouteSeparator + senderNode
	}
	// Do not copy the generated protobuf struct wholesale: it contains an
	// internal message-state mutex and go vet correctly rejects lock copies.
	return &CallContext{
		ChainId:         ctx.ChainId,
		RootOperationId: ctx.RootOperationId,
		ParentCallId:    ctx.ParentCallId,
		Hop:             ctx.Hop + 1,
		Route:           newRoute,
		DeadlineUnixMs:  ctx.DeadlineUnixMs,
	}, nil
}

func cloneCallContext(ctx *CallContext) *CallContext {
	return &CallContext{
		ChainId:         ctx.ChainId,
		RootOperationId: ctx.RootOperationId,
		ParentCallId:    ctx.ParentCallId,
		Hop:             ctx.Hop,
		Route:           ctx.Route,
		DeadlineUnixMs:  ctx.DeadlineUnixMs,
	}
}

// IncrementHop is retained as a compatibility alias for callers that used the
// original helper name. Its semantics are now explicitly sender-side: the
// supplied node is appended to the route.
func IncrementHop(ctx *CallContext, currentNode string) (*CallContext, error) {
	return AppendCallNode(ctx, currentNode)
}

// ValidateAndAdvanceCallContext is the receiver-side operation. It validates
// that receiverNode is a legal next hop and returns a copy with receiverNode
// appended. This makes a complete trace such as ms→agent→product explicit and
// prevents a receiver from accepting an unbound/empty peer route.
func ValidateAndAdvanceCallContext(ctx *CallContext, receiverNode string) (*CallContext, error) {
	if ctx == nil || ctx.Route == "" {
		return nil, fmt.Errorf("%w: receiver requires a non-empty peer route", ErrInvalidCallPath)
	}
	if err := ValidateCallPath(ctx, receiverNode); err != nil {
		return nil, err
	}
	return AppendCallNode(ctx, receiverNode)
}

// DetectCycle rejects repeated nodes. A repeated node means a synchronous
// return path (for example agent→ms→agent), which is forbidden for Product
// Capability handlers and would otherwise permit a deadlock.
func DetectCycle(route string) error {
	if route == "" {
		return nil
	}
	nodes, err := parseRoute(route)
	if err != nil {
		return err
	}
	seen := make(map[string]struct{}, len(nodes))
	for _, node := range nodes {
		if _, exists := seen[node]; exists {
			return fmt.Errorf("%w: node %s appears more than once in route %s", ErrCycleDetected, node, route)
		}
		seen[node] = struct{}{}
	}
	return nil
}

// ValidateCallPath validates a hypothetical append of currentNode. Only the
// following private transitions are allowed:
//
//   - ms -> agent
//   - agent -> product
//
// Product is terminal. External origins are authenticated before entering a
// private service and are deliberately not represented in this route. No
// transition may revisit a node.
func ValidateCallPath(ctx *CallContext, currentNode string) error {
	if err := ValidateCallContext(ctx); err != nil {
		return err
	}
	if !validNode(currentNode) {
		return fmt.Errorf("%w: unknown node %q", ErrInvalidCallPath, currentNode)
	}
	if ctx.Route == "" {
		// A private call may enter only through a trusted message-server or
		// Agent boundary. Flutter/MCP are external origins and never appear in
		// the private route.
		if currentNode != NodeMessage && currentNode != NodeAgent {
			return fmt.Errorf("%w: private route must start with ms or agent", ErrInvalidCallPath)
		}
		return nil
	}
	nodes, _ := parseRoute(ctx.Route) // ValidateCallContext already parsed it.
	previous := nodes[len(nodes)-1]
	if previous == NodeProduct {
		return fmt.Errorf("%w: product is terminal", ErrInvalidCallPath)
	}
	if _, duplicate := indexOf(nodes, currentNode); duplicate {
		return fmt.Errorf("%w: %s would repeat in route %s", ErrCycleDetected, currentNode, ctx.Route)
	}
	if !allowedTransition(previous, currentNode) {
		return fmt.Errorf("%w: transition %s -> %s is not allowed", ErrInvalidCallPath, previous, currentNode)
	}
	if ctx.Hop >= MaxCallHop {
		return fmt.Errorf("%w: hop %d exceeds maximum %d", ErrCycleDetected, ctx.Hop+1, MaxCallHop)
	}
	return nil
}

// ValidateAgentCallPath is the boundary helper for message-server -> Agent.
func ValidateAgentCallPath(ctx *CallContext) error {
	if err := validatePeerBoundary(ctx, NodeMessage); err != nil {
		return err
	}
	return ValidateCallPath(ctx, NodeAgent)
}

// ValidateProductCallPath is the boundary helper for Agent -> message-server.
func ValidateProductCallPath(ctx *CallContext) error {
	if err := validatePeerBoundary(ctx, NodeAgent); err != nil {
		return err
	}
	return ValidateCallPath(ctx, NodeProduct)
}

// ValidateAndAdvanceAgentCallContext validates a message-server → Agent call
// and appends the Agent receiver node.
func ValidateAndAdvanceAgentCallContext(ctx *CallContext) (*CallContext, error) {
	if err := ValidateAgentCallPath(ctx); err != nil {
		return nil, err
	}
	return AppendCallNode(ctx, NodeAgent)
}

// ValidateAndAdvanceProductCallContext validates an Agent → Product call and
// appends the Product receiver node.
func ValidateAndAdvanceProductCallContext(ctx *CallContext) (*CallContext, error) {
	if err := ValidateProductCallPath(ctx); err != nil {
		return nil, err
	}
	return AppendCallNode(ctx, NodeProduct)
}

// IsTerminalNode reports whether a node must not synchronously forward a call.
func IsTerminalNode(node string) bool { return node == NodeProduct }

// NewCallContext creates a context for an initial call. If no root operation ID
// is supplied (for example a query), chainID is used as the root so the
// context remains auditable and passes validation.
func NewCallContext(chainID, rootOperationID string, deadlineUnixMs int64) *CallContext {
	if rootOperationID == "" {
		rootOperationID = chainID
	}
	return &CallContext{
		ChainId:         chainID,
		RootOperationId: rootOperationID,
		ParentCallId:    "",
		Hop:             0,
		Route:           "",
		DeadlineUnixMs:  deadlineUnixMs,
	}
}

// RemainingDeadlineMs returns the remaining budget, clamped at zero.
func RemainingDeadlineMs(ctx *CallContext, nowUnixMs int64) int64 {
	if ctx == nil || ctx.DeadlineUnixMs <= 0 {
		return 0
	}
	remaining := ctx.DeadlineUnixMs - nowUnixMs
	if remaining < 0 {
		return 0
	}
	return remaining
}

// DeadlineExceeded reports whether a context deadline has elapsed.
func DeadlineExceeded(ctx *CallContext, now time.Time) bool {
	return RemainingDeadlineMs(ctx, now.UnixMilli()) == 0
}

func parseRoute(route string) ([]string, error) {
	if route == "" {
		return nil, nil
	}
	if strings.Contains(route, " ") || strings.Contains(route, "\t") || strings.Contains(route, "\n") {
		return nil, fmt.Errorf("%w: route contains whitespace", ErrInvalidCallPath)
	}
	parts := strings.Split(route, RouteSeparator)
	if len(parts) == 0 {
		return nil, fmt.Errorf("%w: route is empty", ErrInvalidCallPath)
	}
	for _, node := range parts {
		if !validNode(node) {
			return nil, fmt.Errorf("%w: unknown route node %q", ErrInvalidCallPath, node)
		}
	}
	return parts, nil
}

func validNode(node string) bool {
	switch node {
	case NodeMessage, NodeAgent, NodeProduct:
		return true
	default:
		return false
	}
}

func allowedTransition(from, to string) bool {
	switch from {
	case NodeMessage:
		return to == NodeAgent
	case NodeAgent:
		return to == NodeProduct
	default:
		return false
	}
}

func indexOf(nodes []string, target string) (int, bool) {
	for i, node := range nodes {
		if node == target {
			return i, true
		}
	}
	return -1, false
}

func validatePeerBoundary(ctx *CallContext, expectedPeer string) error {
	if ctx == nil || ctx.Route == "" {
		return fmt.Errorf("%w: peer-bound call_context route is required", ErrInvalidCallPath)
	}
	if err := ValidateStrictCallContext(ctx); err != nil {
		return err
	}
	nodes, err := parseRoute(ctx.Route)
	if err != nil {
		return err
	}
	if nodes[len(nodes)-1] != expectedPeer {
		return fmt.Errorf("%w: expected peer %s, got %s", ErrInvalidCallPath, expectedPeer, nodes[len(nodes)-1])
	}
	return nil
}

func validateToken(name, value string) error {
	if value == "" {
		return fmt.Errorf("%s is required", name)
	}
	if len(value) > 128 || strings.TrimSpace(value) != value || strings.ContainsAny(value, "\r\n\t") {
		return fmt.Errorf("%s is invalid", name)
	}
	return nil
}
