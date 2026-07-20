package prompt

import (
	"testing"

	"github.com/LingByte/lingllm/protocol"
)

func TestReasoningTemplate(t *testing.T) {
	t.Run("basic creation", func(t *testing.T) {
		rt := NewReasoningTemplate()
		if rt == nil {
			t.Fatal("expected non-nil ReasoningTemplate")
		}
	})

	t.Run("with format", func(t *testing.T) {
		rt := NewReasoningTemplate()
		result := rt.WithFormat(FormatXML)

		if result != rt {
			t.Error("expected fluent interface")
		}
	})

	t.Run("with steps template", func(t *testing.T) {
		rt := NewReasoningTemplate()
		tpl := MustNewTemplate("steps", "{{thought}}")
		result := rt.WithStepsTemplate(tpl)

		if result != rt {
			t.Error("expected fluent interface")
		}
	})

	t.Run("with stop at step", func(t *testing.T) {
		rt := NewReasoningTemplate()
		result := rt.WithStopAtStep(5)

		if result != rt {
			t.Error("expected fluent interface")
		}
		if rt.stopAtStep != 5 {
			t.Errorf("expected stopAtStep 5, got %d", rt.stopAtStep)
		}
	})

	t.Run("build system prompt XML format", func(t *testing.T) {
		rt := NewReasoningTemplate()
		rt.WithFormat(FormatXML)

		prompt := rt.BuildSystemPrompt()
		if !containsString(prompt, "<thought>") {
			t.Error("expected XML format in prompt")
		}
	})

	t.Run("build system prompt JSON format", func(t *testing.T) {
		rt := NewReasoningTemplate()
		rt.WithFormat(FormatJSON)

		prompt := rt.BuildSystemPrompt()
		if !containsString(prompt, "thought") {
			t.Error("expected JSON format in prompt")
		}
	})

	t.Run("build system prompt plain format", func(t *testing.T) {
		rt := NewReasoningTemplate()
		rt.WithFormat(FormatPlain)

		prompt := rt.BuildSystemPrompt()
		if prompt == "" {
			t.Error("expected non-empty prompt")
		}
	})

	t.Run("build system prompt numbered format", func(t *testing.T) {
		rt := NewReasoningTemplate()
		rt.WithFormat(FormatNumbered)

		prompt := rt.BuildSystemPrompt()
		if !containsString(prompt, "Step") {
			t.Error("expected numbered format in prompt")
		}
	})

	t.Run("build system prompt with custom template", func(t *testing.T) {
		rt := NewReasoningTemplate()
		tpl := MustNewTemplate("custom", "Think: {{thought}}")
		rt.WithStepsTemplate(tpl)

		prompt := rt.BuildSystemPrompt()
		if !containsString(prompt, "Think:") {
			t.Error("expected custom template in prompt")
		}
	})
}

func TestParseSteps(t *testing.T) {
	t.Run("parse XML steps", func(t *testing.T) {
		rt := NewReasoningTemplate()
		rt.WithFormat(FormatXML)

		response := `<thought>I need to add 2 and 2</thought>
<action>Use basic arithmetic</action>
<result>4</result>`

		steps, err := rt.ParseSteps(response)
		if err != nil {
			t.Fatalf("ParseSteps failed: %v", err)
		}
		if len(steps) != 1 {
			t.Errorf("expected 1 step, got %d", len(steps))
		}
		if len(steps) > 0 && steps[0].Thought != "I need to add 2 and 2" {
			t.Errorf("unexpected thought: %s", steps[0].Thought)
		}
	})

	t.Run("parse multiple XML steps", func(t *testing.T) {
		rt := NewReasoningTemplate()
		rt.WithFormat(FormatXML)

		response := `<thought>Step 1 thought</thought>
<action>Step 1 action</action>
<result>Step 1 result</result>
<thought>Step 2 thought</thought>
<action>Step 2 action</action>
<result>Step 2 result</result>`

		steps, err := rt.ParseSteps(response)
		if err != nil {
			t.Fatalf("ParseSteps failed: %v", err)
		}
		if len(steps) != 2 {
			t.Errorf("expected 2 steps, got %d", len(steps))
		}
	})

	t.Run("parse numbered steps", func(t *testing.T) {
		rt := NewReasoningTemplate()
		rt.WithFormat(FormatNumbered)

		response := `Step 1: First thought
Action 1: First action
Result 1: First result`

		steps, err := rt.ParseSteps(response)
		if err != nil {
			t.Fatalf("ParseSteps failed: %v", err)
		}
		if len(steps) == 0 {
			t.Error("expected at least one step")
		}
	})

	t.Run("parse plain steps", func(t *testing.T) {
		rt := NewReasoningTemplate()
		rt.WithFormat(FormatPlain)

		response := `First line
Second line
Third line`

		steps, err := rt.ParseSteps(response)
		if err != nil {
			t.Fatalf("ParseSteps failed: %v", err)
		}
		if len(steps) != 3 {
			t.Errorf("expected 3 steps, got %d", len(steps))
		}
	})

	t.Run("parse JSON steps (basic)", func(t *testing.T) {
		rt := NewReasoningTemplate()
		rt.WithFormat(FormatJSON)

		// Current implementation returns empty steps
		steps, err := rt.ParseSteps(`{"thought": "test"}`)
		if err != nil {
			t.Fatalf("ParseSteps failed: %v", err)
		}
		// JSON parsing is not fully implemented
		_ = steps
	})

	t.Run("stop at step limit", func(t *testing.T) {
		rt := NewReasoningTemplate()
		rt.WithFormat(FormatXML)
		rt.WithStopAtStep(1)

		response := `<thought>Step 1</thought>
<action>Action 1</action>
<result>Result 1</result>
<thought>Step 2</thought>
<action>Action 2</action>
<result>Result 2</result>`

		steps, err := rt.ParseSteps(response)
		if err != nil {
			t.Fatalf("ParseSteps failed: %v", err)
		}
		if len(steps) != 1 {
			t.Errorf("expected 1 step due to limit, got %d", len(steps))
		}
	})
}

func TestToConversation(t *testing.T) {
	t.Run("basic conversion", func(t *testing.T) {
		rt := NewReasoningTemplate()
		rt.WithFormat(FormatXML)

		conv := rt.ToConversation("What is 2+2?")

		if conv == nil {
			t.Fatal("expected non-nil conversation")
		}
		if conv.Name() != "cot-conversation" {
			t.Errorf("expected name 'cot-conversation', got %s", conv.Name())
		}
	})
}

func TestTreeOfThought(t *testing.T) {
	t.Run("basic creation", func(t *testing.T) {
		tot := NewTreeOfThought()
		if tot == nil {
			t.Fatal("expected non-nil TreeOfThought")
		}
		if tot.maxDepth != 5 {
			t.Errorf("expected maxDepth 5, got %d", tot.maxDepth)
		}
		if tot.branchLimit != 3 {
			t.Errorf("expected branchLimit 3, got %d", tot.branchLimit)
		}
	})

	t.Run("with max depth", func(t *testing.T) {
		tot := NewTreeOfThought()
		result := tot.WithMaxDepth(10)

		if result != tot {
			t.Error("expected fluent interface")
		}
		if tot.maxDepth != 10 {
			t.Errorf("expected maxDepth 10, got %d", tot.maxDepth)
		}
	})

	t.Run("with branch limit", func(t *testing.T) {
		tot := NewTreeOfThought()
		result := tot.WithBranchLimit(5)

		if result != tot {
			t.Error("expected fluent interface")
		}
		if tot.branchLimit != 5 {
			t.Errorf("expected branchLimit 5, got %d", tot.branchLimit)
		}
	})

	t.Run("add root", func(t *testing.T) {
		tot := NewTreeOfThought()
		node := tot.AddRoot("Initial thought")

		if node == nil {
			t.Fatal("expected non-nil node")
		}
		if node.Thought != "Initial thought" {
			t.Errorf("unexpected thought: %s", node.Thought)
		}
		if node.Depth != 0 {
			t.Errorf("expected depth 0, got %d", node.Depth)
		}
	})

	t.Run("add child", func(t *testing.T) {
		tot := NewTreeOfThought()
		root := tot.AddRoot("Root")
		child := root.AddChild("Child thought", 0.8)

		if child == nil {
			t.Fatal("expected non-nil child")
		}
		if child.Thought != "Child thought" {
			t.Errorf("unexpected thought: %s", child.Thought)
		}
		if child.Score != 0.8 {
			t.Errorf("unexpected score: %f", child.Score)
		}
		if child.Parent != root {
			t.Error("expected parent to be set")
		}
		if child.Depth != 1 {
			t.Errorf("expected depth 1, got %d", child.Depth)
		}
	})

	t.Run("best path with no root", func(t *testing.T) {
		tot := NewTreeOfThought()
		path := tot.BestPath()

		if path != nil {
			t.Error("expected nil path for empty tree")
		}
	})

	t.Run("best path single node with children", func(t *testing.T) {
		tot := NewTreeOfThought()
		root := tot.AddRoot("Only node")
		root.AddChild("Child node", 0.5)

		path := tot.BestPath()
		if path == nil {
			t.Fatal("expected non-nil path")
		}
		if len(path) < 1 {
			t.Error("expected at least 1 node in path")
		}
	})

	t.Run("best path with children", func(t *testing.T) {
		tot := NewTreeOfThought()
		root := tot.AddRoot("Start")
		root.AddChild("Path A", 0.9)
		root.AddChild("Path B", 0.5)

		path := tot.BestPath()
		if path == nil {
			t.Fatal("expected non-nil path")
		}
		if path[0].Thought != "Start" {
			t.Errorf("expected first thought 'Start', got %s", path[0].Thought)
		}
	})

	t.Run("best path with grandchildren", func(t *testing.T) {
		tot := NewTreeOfThought()
		root := tot.AddRoot("Start")
		child1 := root.AddChild("High scoring", 0.9)
		child1.AddChild("Best final", 0.95)
		child1.AddChild("Good final", 0.8)
		_ = root.AddChild("Low scoring", 0.3)

		path := tot.BestPath()
		if path == nil {
			t.Fatal("expected non-nil path")
		}
		if len(path) < 2 {
			t.Error("expected at least 2 nodes in path")
		}
	})

	t.Run("string representation", func(t *testing.T) {
		tot := NewTreeOfThought()
		root := tot.AddRoot("Start")
		root.AddChild("Path A", 0.9)
		root.AddChild("Path B", 0.5)

		s := tot.String()
		if s == "" {
			t.Error("expected non-empty string")
		}
		if !containsString(s, "Start") {
			t.Error("expected string to contain 'Start'")
		}
	})

	t.Run("string with zero root", func(t *testing.T) {
		tot := NewTreeOfThought()
		// Don't add any nodes
		s := tot.String()
		if s != "" {
			t.Error("expected empty string for empty tree")
		}
	})
}

func TestToTNode(t *testing.T) {
	t.Run("node fields", func(t *testing.T) {
		tot := NewTreeOfThought()
		root := tot.AddRoot("Root")
		child := root.AddChild("Child", 0.8)
		grandchild := child.AddChild("Grandchild", 0.9)

		if root.Depth != 0 {
			t.Errorf("expected root depth 0, got %d", root.Depth)
		}
		if child.Depth != 1 {
			t.Errorf("expected child depth 1, got %d", child.Depth)
		}
		if grandchild.Depth != 2 {
			t.Errorf("expected grandchild depth 2, got %d", grandchild.Depth)
		}
		if len(root.Children) != 1 {
			t.Errorf("expected 1 child, got %d", len(root.Children))
		}
	})
}

func TestReflectionTemplate(t *testing.T) {
	t.Run("basic creation", func(t *testing.T) {
		rt := NewReflectionTemplate()
		if rt == nil {
			t.Fatal("expected non-nil ReflectionTemplate")
		}
	})

	t.Run("build critique request", func(t *testing.T) {
		rt := NewReflectionTemplate()

		req := rt.BuildCritiqueRequest("My response to critique")

		if req == nil {
			t.Fatal("expected non-nil request")
		}
		if len(req.Messages) != 1 {
			t.Errorf("expected 1 message, got %d", len(req.Messages))
		}
		if req.Messages[0].Role != protocol.RoleUser {
			t.Errorf("expected user role, got %s", req.Messages[0].Role)
		}
	})

	t.Run("build improvement request", func(t *testing.T) {
		rt := NewReflectionTemplate()

		req := rt.BuildImprovementRequest("My critique of the response")

		if req == nil {
			t.Fatal("expected non-nil request")
		}
		if len(req.Messages) != 1 {
			t.Errorf("expected 1 message, got %d", len(req.Messages))
		}
	})
}

func TestReasoningFormat(t *testing.T) {
	t.Run("format constants", func(t *testing.T) {
		if FormatXML != 0 {
			t.Errorf("expected FormatXML to be 0, got %d", FormatXML)
		}
		if FormatJSON != 1 {
			t.Errorf("expected FormatJSON to be 1, got %d", FormatJSON)
		}
		if FormatPlain != 2 {
			t.Errorf("expected FormatPlain to be 2, got %d", FormatPlain)
		}
		if FormatNumbered != 3 {
			t.Errorf("expected FormatNumbered to be 3, got %d", FormatNumbered)
		}
	})
}
