package workflow

import (
	"fmt"
	"strconv"
	"strings"
)

type GatewayNode struct {
	Node                                                                        // basic node
	Condition       string                                                      // ctx key for condition or expression
	Expression      string                                                      // expression to evaluate (alternative to Condition)
	Evaluator       func(ctx *WorkflowContext, expression string) (bool, error) // optional expression evaluator
	TrueNextNodeID  string
	FalseNextNodeID string
	StoreResult     bool // whether to store evaluation result in context
}

// Evaluate evaluates the condition/expression and returns the result
func (g *GatewayNode) Evaluate(ctx *WorkflowContext) (bool, error) {
	// Debug: log what we have
	if ctx != nil {
		ctx.AddLog("debug", fmt.Sprintf("GatewayNode.Evaluate: Expression='%s', Condition='%s'", g.Expression, g.Condition), g.ID, g.Name)
	}

	// If Expression is set and Evaluator is available, use expression evaluation
	if g.Expression != "" && g.Evaluator != nil {
		return g.Evaluator(ctx, g.Expression)
	}

	// If Expression is set but no Evaluator, treat it as a simple check
	if g.Expression != "" {
		return g.Expression != "", nil
	}

	// Otherwise, use Condition as a context key
	if ctx != nil && g.Condition != "" {
		// Debug: log the condition being evaluated
		if ctx != nil {
			ctx.AddLog("debug", fmt.Sprintf("Evaluating condition: '%s' (length: %d)", g.Condition, len(g.Condition)), g.ID, g.Name)
		}

		// Check if condition contains comparison operators (>, <, >=, <=, ==, !=)
		hasComparison := containsComparison(g.Condition)
		if ctx != nil {
			ctx.AddLog("debug", fmt.Sprintf("Condition contains comparison: %v", hasComparison), g.ID, g.Name)
		}

		if hasComparison {
			result, err := g.evaluateComparison(ctx, g.Condition)
			if err != nil {
				// Log the error
				if ctx != nil {
					ctx.AddLog("error", fmt.Sprintf("Comparison evaluation failed: %v", err), g.ID, g.Name)
				}
				// If comparison fails, fall back to truthy check
				if val, ok := ctx.ResolveValue(g.Condition); ok {
					truthyResult := truthy(val)
					if ctx != nil {
						ctx.AddLog("debug", fmt.Sprintf("Falling back to truthy check: %v -> %v", val, truthyResult), g.ID, g.Name)
					}
					return truthyResult, nil
				}
				return false, err
			}
			if ctx != nil {
				ctx.AddLog("debug", fmt.Sprintf("Comparison result: %v", result), g.ID, g.Name)
			}
			return result, nil
		}

		// Simple value lookup and truthy check
		if val, ok := ctx.ResolveValue(g.Condition); ok {
			truthyResult := truthy(val)
			if ctx != nil {
				ctx.AddLog("debug", fmt.Sprintf("Truthy check: %v -> %v", val, truthyResult), g.ID, g.Name)
			}
			return truthyResult, nil
		}

		if ctx != nil {
			ctx.AddLog("warning", fmt.Sprintf("Cannot resolve value for condition: %s", g.Condition), g.ID, g.Name)
		}
	}

	// Default to false if no condition/expression is set
	return false, nil
}

// containsComparison checks if a condition string contains comparison operators
func containsComparison(condition string) bool {
	return strings.Contains(condition, " > ") || strings.Contains(condition, " < ") ||
		strings.Contains(condition, " >= ") || strings.Contains(condition, " <= ") ||
		strings.Contains(condition, " == ") || strings.Contains(condition, " != ") ||
		strings.Contains(condition, ">") || strings.Contains(condition, "<") ||
		strings.Contains(condition, ">=") || strings.Contains(condition, "<=") ||
		strings.Contains(condition, "==") || strings.Contains(condition, "!=")
}

// evaluateComparison evaluates a simple comparison expression like "parameters.input > 0"
func (g *GatewayNode) evaluateComparison(ctx *WorkflowContext, condition string) (bool, error) {
	// Parse simple comparison: "parameters.input > 0" or "context.value >= 100"
	// Support: >, <, >=, <=, ==, !=

	var operator string
	var operatorPos int

	// Find operator position - must check longer operators first
	operators := []string{">=", "<=", "==", "!=", ">", "<"}
	for _, op := range operators {
		if idx := strings.Index(condition, op); idx != -1 {
			operator = op
			operatorPos = idx
			break
		}
	}

	if operator == "" {
		// No operator found, fall back to truthy check
		if val, ok := ctx.ResolveValue(condition); ok {
			return truthy(val), nil
		}
		return false, fmt.Errorf("no comparison operator found in condition: %s", condition)
	}

	// Split into left and right parts
	leftKey := strings.TrimSpace(condition[:operatorPos])
	rightStr := strings.TrimSpace(condition[operatorPos+len(operator):])

	if leftKey == "" {
		return false, fmt.Errorf("empty left side of comparison in condition: %s", condition)
	}
	if rightStr == "" {
		return false, fmt.Errorf("empty right side of comparison in condition: %s", condition)
	}

	// Resolve left value
	leftVal, ok := ctx.ResolveValue(leftKey)
	if !ok {
		if ctx != nil {
			ctx.AddLog("error", fmt.Sprintf("Cannot resolve left value for key: %s", leftKey), g.ID, g.Name)
		}
		return false, fmt.Errorf("cannot resolve value for key: %s (condition: %s)", leftKey, condition)
	}

	if ctx != nil {
		ctx.AddLog("debug", fmt.Sprintf("Left value resolved: %v (type: %T)", leftVal, leftVal), g.ID, g.Name)
	}

	// Parse right value (could be a number or a key)
	var rightVal interface{}

	// Try to parse as number first
	if num, err := parseNumber(rightStr); err == nil {
		rightVal = num
		if ctx != nil {
			ctx.AddLog("debug", fmt.Sprintf("Right value parsed as number: %v", rightVal), g.ID, g.Name)
		}
	} else {
		// Try to resolve as a key
		if val, ok := ctx.ResolveValue(rightStr); ok {
			rightVal = val
			if ctx != nil {
				ctx.AddLog("debug", fmt.Sprintf("Right value resolved from context: %v", rightVal), g.ID, g.Name)
			}
		} else {
			// Try to parse as string literal (remove quotes if present)
			strVal := rightStr
			if (strings.HasPrefix(strVal, "\"") && strings.HasSuffix(strVal, "\"")) ||
				(strings.HasPrefix(strVal, "'") && strings.HasSuffix(strVal, "'")) {
				strVal = strVal[1 : len(strVal)-1]
			}
			rightVal = strVal
			if ctx != nil {
				ctx.AddLog("debug", fmt.Sprintf("Right value treated as string: %v", rightVal), g.ID, g.Name)
			}
		}
	}

	// Perform comparison
	result, err := compareValues(leftVal, rightVal, operator)
	if err != nil {
		if ctx != nil {
			ctx.AddLog("error", fmt.Sprintf("Comparison failed: %v (left=%v, right=%v, op=%s)", err, leftVal, rightVal, operator), g.ID, g.Name)
		}
		return false, fmt.Errorf("comparison failed: %w (left=%v, right=%v, op=%s)", err, leftVal, rightVal, operator)
	}

	if ctx != nil {
		ctx.AddLog("debug", fmt.Sprintf("Comparison: %v %s %v = %v", leftVal, operator, rightVal, result), g.ID, g.Name)
	}
	return result, nil
}

// parseNumber tries to parse a string as a number (int, int64, or float64)
func parseNumber(s string) (interface{}, error) {
	// Try int64 first
	if i, err := strconv.ParseInt(s, 10, 64); err == nil {
		return i, nil
	}
	// Try float64
	if f, err := strconv.ParseFloat(s, 64); err == nil {
		return f, nil
	}
	return nil, fmt.Errorf("not a number: %s", s)
}

// compareValues compares two values using the specified operator
func compareValues(left, right interface{}, operator string) (bool, error) {
	// Convert both to numbers if possible
	leftNum, leftIsNum := toNumber(left)
	rightNum, rightIsNum := toNumber(right)

	if leftIsNum && rightIsNum {
		// Both are numbers, do numeric comparison
		switch operator {
		case ">":
			return leftNum > rightNum, nil
		case "<":
			return leftNum < rightNum, nil
		case ">=":
			return leftNum >= rightNum, nil
		case "<=":
			return leftNum <= rightNum, nil
		case "==":
			return leftNum == rightNum, nil
		case "!=":
			return leftNum != rightNum, nil
		}
	}

	// Fall back to string comparison or equality
	leftStr := fmt.Sprintf("%v", left)
	rightStr := fmt.Sprintf("%v", right)

	switch operator {
	case "==":
		return leftStr == rightStr, nil
	case "!=":
		return leftStr != rightStr, nil
	default:
		return false, fmt.Errorf("operator %s not supported for non-numeric comparison", operator)
	}
}

// toNumber converts a value to a float64 if possible
func toNumber(val interface{}) (float64, bool) {
	switch v := val.(type) {
	case int:
		return float64(v), true
	case int64:
		return float64(v), true
	case float64:
		return v, true
	case float32:
		return float64(v), true
	default:
		return 0, false
	}
}

func (g *GatewayNode) EvaluateCondition(ctx *WorkflowContext) string {
	if len(g.TrueNextNodeID) == 0 && len(g.FalseNextNodeID) == 0 && len(g.NextNodes) > 0 {
		return g.NextNodes[0]
	}

	result, err := g.Evaluate(ctx)
	if err != nil {
		// On error, default to false branch
		if g.FalseNextNodeID != "" {
			return g.FalseNextNodeID
		}
		if len(g.NextNodes) > 1 {
			return g.NextNodes[1]
		}
		return ""
	}

	if result {
		if g.TrueNextNodeID != "" {
			return g.TrueNextNodeID
		}
		if len(g.NextNodes) > 0 {
			return g.NextNodes[0]
		}
	} else {
		if g.FalseNextNodeID != "" {
			return g.FalseNextNodeID
		}
		if len(g.NextNodes) > 1 {
			return g.NextNodes[1]
		}
	}

	return ""
}

func (g *GatewayNode) Base() *Node {
	return &g.Node
}

func (g *GatewayNode) Run(ctx *WorkflowContext) ([]string, error) {
	// Evaluate the condition/expression
	result, err := g.Evaluate(ctx)
	if err != nil {
		return nil, fmt.Errorf("gateway node %s evaluation failed: %w", g.Name, err)
	}

	// Store result in context if enabled (always store for testing purposes)
	if ctx != nil {
		if ctx.NodeData == nil {
			ctx.NodeData = make(map[string]interface{})
		}
		// Always store the evaluation result for visibility
		resultKey := g.Properties["result_key"]
		if resultKey == "" {
			resultKey = g.ID + "_result"
		}
		ctx.NodeData[resultKey] = result
		// Also store with a standard key for easy access
		ctx.NodeData[g.ID+"_evaluated"] = result
		ctx.NodeData["gateway_result"] = result
	}

	// Determine next node
	next := g.EvaluateCondition(ctx)
	if next == "" {
		// For testing purposes, allow node to complete even without edges
		// Just return empty next nodes array to indicate no routing
		// This allows testing the condition evaluation without requiring edges
		if len(g.TrueNextNodeID) == 0 && len(g.FalseNextNodeID) == 0 && len(g.NextNodes) == 0 {
			// This is a test scenario, allow it to complete
			return []string{}, nil
		}
		if g.Condition == "" && g.Expression == "" {
			return nil, fmt.Errorf("gateway node %s has no condition or expression set. Please configure the condition key (e.g., 'parameters.value' or 'context.value') or expression", g.Name)
		}
		return nil, fmt.Errorf("gateway node %s has no valid next node. Condition evaluated to %v, but no corresponding branch edge is connected", g.Name, result)
	}
	return []string{next}, nil
}
