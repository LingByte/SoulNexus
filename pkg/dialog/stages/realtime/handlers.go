package realtime

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
	"unicode"

	stageknow "github.com/LingByte/SoulNexus/pkg/dialog/stages/knowledge"
	"github.com/LingByte/SoulNexus/pkg/utils/timeutil"
	"github.com/LingByte/lingllm/realtime"
	"go.uber.org/zap"
)

var weekdayZH = []string{"星期日", "星期一", "星期二", "星期三", "星期四", "星期五", "星期六"}

// voiceRealtimeToolHandler dispatches Qwen-Omni-Realtime function calls for a voice session.
type voiceRealtimeToolHandler struct {
	callID          string
	lg              *zap.Logger
	knowledge       stageknow.Binding
}

// NewToolHandler dispatches Qwen-Omni-Realtime function calls for voice sessions.
func NewToolHandler(callID string, lg *zap.Logger, kb stageknow.Binding) realtime.ToolHandler {
	return (&voiceRealtimeToolHandler{
		callID:    callID,
		lg:        lg,
		knowledge: kb,
	}).handle
}

func (h *voiceRealtimeToolHandler) handle(name string, args map[string]any) string {
	switch name {
	case "search_knowledge_base":
		return h.handleSearchKnowledge(args)
	case "get_current_time":
		return toolJSON(runGetCurrentTime(args))
	case "is_business_hours":
		return toolJSON(runIsBusinessHours(args))
	case "calculate":
		return toolJSON(runCalculate(args))
	default:
		return toolJSON(map[string]any{"ok": false, "error": "unknown tool: " + name})
	}
}

func (h *voiceRealtimeToolHandler) handleSearchKnowledge(args map[string]any) string {
	return stageknow.ExecuteSearchTool(context.Background(), h.callID, args, h.lg)
}

func toolJSON(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		return `{"ok":false,"error":"marshal failed"}`
	}
	return string(b)
}

func realtimeTimezoneFromArgs(args map[string]any) (*time.Location, string) {
	tz := timeutil.Name()
	if v, ok := args["timezone"].(string); ok && strings.TrimSpace(v) != "" {
		tz = strings.TrimSpace(v)
	}
	loc, err := time.LoadLocation(tz)
	if err != nil {
		loc = timeutil.Location()
		tz = timeutil.Name()
	}
	return loc, tz
}

func runGetCurrentTime(args map[string]any) map[string]any {
	loc, tz := realtimeTimezoneFromArgs(args)
	now := time.Now().In(loc)
	wd := weekdayZH[int(now.Weekday())]
	spoken := fmt.Sprintf("%s %s %d点%d分",
		now.Format("2006年1月2日"), wd, now.Hour(), now.Minute())
	return map[string]any{
		"ok":         true,
		"timezone":   tz,
		"iso8601":    now.Format(time.RFC3339),
		"date":       now.Format("2006-01-02"),
		"time":       now.Format("15:04:05"),
		"weekday_zh": wd,
		"spoken_zh":  spoken,
	}
}

func runIsBusinessHours(args map[string]any) map[string]any {
	loc, tz := realtimeTimezoneFromArgs(args)
	now := time.Now().In(loc)
	wd := now.Weekday()
	hour := now.Hour()
	inHours := wd >= time.Monday && wd <= time.Friday && hour >= 9 && hour < 18
	var spoken string
	if inHours {
		spoken = "当前是工作时间（周一至周五 9:00-18:00）。"
	} else {
		spoken = "当前不在工作时间（周一至周五 9:00-18:00）。"
	}
	return map[string]any{
		"ok":                true,
		"timezone":          tz,
		"in_business_hours": inHours,
		"weekday":           wd.String(),
		"hour":              hour,
		"spoken_zh":         spoken,
	}
}

func runCalculate(args map[string]any) map[string]any {
	expr, _ := args["expression"].(string)
	expr = strings.TrimSpace(expr)
	if expr == "" {
		return map[string]any{"ok": false, "error": "expression is required"}
	}
	for _, r := range expr {
		if !unicode.IsDigit(r) && !strings.ContainsRune("+-*/(). ", r) {
			return map[string]any{"ok": false, "error": "expression contains invalid characters"}
		}
	}
	v, err := evalSimpleArithmetic(expr)
	if err != nil {
		return map[string]any{"ok": false, "error": err.Error()}
	}
	return map[string]any{
		"ok":         true,
		"expression": expr,
		"result":     v,
		"spoken_zh":  fmt.Sprintf("计算结果是 %g", v),
	}
}

// evalSimpleArithmetic evaluates + - * / with parentheses (shunting-yard).
func evalSimpleArithmetic(expr string) (float64, error) {
	expr = strings.ReplaceAll(expr, " ", "")
	if expr == "" {
		return 0, fmt.Errorf("empty expression")
	}
	out, err := infixToRPN(tokenize(expr))
	if err != nil {
		return 0, err
	}
	return evalRPN(out)
}

func tokenize(expr string) []string {
	var tokens []string
	var num strings.Builder
	flushNum := func() {
		if num.Len() > 0 {
			tokens = append(tokens, num.String())
			num.Reset()
		}
	}
	for i, r := range expr {
		switch r {
		case '+', '-', '*', '/', '(', ')':
			flushNum()
			// unary minus: at start or after ( or operator
			if r == '-' && (i == 0 || expr[i-1] == '(' || isOpRune(rune(expr[i-1]))) {
				num.WriteRune(r)
			} else {
				tokens = append(tokens, string(r))
			}
		default:
			num.WriteRune(r)
		}
	}
	flushNum()
	return tokens
}

func isOpRune(r rune) bool {
	return r == '+' || r == '-' || r == '*' || r == '/'
}

func prec(op string) int {
	switch op {
	case "+", "-":
		return 1
	case "*", "/":
		return 2
	default:
		return 0
	}
}

func infixToRPN(tokens []string) ([]string, error) {
	var out, stack []string
	for _, t := range tokens {
		switch t {
		case "+", "-", "*", "/":
			for len(stack) > 0 && stack[len(stack)-1] != "(" && prec(stack[len(stack)-1]) >= prec(t) {
				out = append(out, stack[len(stack)-1])
				stack = stack[:len(stack)-1]
			}
			stack = append(stack, t)
		case "(":
			stack = append(stack, t)
		case ")":
			for len(stack) > 0 && stack[len(stack)-1] != "(" {
				out = append(out, stack[len(stack)-1])
				stack = stack[:len(stack)-1]
			}
			if len(stack) == 0 {
				return nil, fmt.Errorf("mismatched parentheses")
			}
			stack = stack[:len(stack)-1]
		default:
			if _, err := strconv.ParseFloat(t, 64); err != nil {
				return nil, fmt.Errorf("invalid number %q", t)
			}
			out = append(out, t)
		}
	}
	for len(stack) > 0 {
		if stack[len(stack)-1] == "(" {
			return nil, fmt.Errorf("mismatched parentheses")
		}
		out = append(out, stack[len(stack)-1])
		stack = stack[:len(stack)-1]
	}
	return out, nil
}

func evalRPN(tokens []string) (float64, error) {
	var stack []float64
	for _, t := range tokens {
		switch t {
		case "+", "-", "*", "/":
			if len(stack) < 2 {
				return 0, fmt.Errorf("invalid expression")
			}
			b, a := stack[len(stack)-1], stack[len(stack)-2]
			stack = stack[:len(stack)-2]
			switch t {
			case "+":
				stack = append(stack, a+b)
			case "-":
				stack = append(stack, a-b)
			case "*":
				stack = append(stack, a*b)
			case "/":
				if b == 0 {
					return 0, fmt.Errorf("division by zero")
				}
				stack = append(stack, a/b)
			}
		default:
			v, err := strconv.ParseFloat(t, 64)
			if err != nil {
				return 0, err
			}
			stack = append(stack, v)
		}
	}
	if len(stack) != 1 {
		return 0, fmt.Errorf("invalid expression")
	}
	if math.IsNaN(stack[0]) || math.IsInf(stack[0], 0) {
		return 0, fmt.Errorf("invalid result")
	}
	return stack[0], nil
}
