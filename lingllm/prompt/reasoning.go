package prompt

import (
	"fmt"
	"strings"

	"github.com/LingByte/lingllm/protocol"
)

// Step represents a single reasoning step in chain-of-thought.
type Step struct {
	Thought string // The reasoning thought
	Action  string // The action taken
	Result  string // The result of the action
}

// ReasoningTemplate provides chain-of-thought prompting.
type ReasoningTemplate struct {
	stepsTemplate  *Template
	format         ReasoningFormat
	includeActions bool
	stopAtStep     int // 0 means no limit
}

// ReasoningFormat defines how reasoning steps are formatted.
type ReasoningFormat int

const (
	FormatXML ReasoningFormat = iota
	FormatJSON
	FormatPlain
	FormatNumbered
)

// NewReasoningTemplate creates a new reasoning template.
func NewReasoningTemplate() *ReasoningTemplate {
	return &ReasoningTemplate{
		format:         FormatXML,
		includeActions: true,
	}
}

// WithFormat sets the reasoning format.
func (rt *ReasoningTemplate) WithFormat(format ReasoningFormat) *ReasoningTemplate {
	rt.format = format
	return rt
}

// WithStepsTemplate sets a custom template for rendering steps.
// Available variables: {{thought}}, {{action}}, {{result}}, {{@index}}
func (rt *ReasoningTemplate) WithStepsTemplate(tpl *Template) *ReasoningTemplate {
	rt.stepsTemplate = tpl
	return rt
}

// WithStopAtStep sets when to stop the reasoning (0 = no limit).
func (rt *ReasoningTemplate) WithStopAtStep(step int) *ReasoningTemplate {
	rt.stopAtStep = step
	return rt
}

// BuildSystemPrompt builds the system prompt for chain-of-thought.
func (rt *ReasoningTemplate) BuildSystemPrompt() string {
	if rt.stepsTemplate != nil {
		result, _ := rt.stepsTemplate.Render(map[string]any{
			"thought": "{{thought}}",
			"action":  "{{action}}",
			"result":  "{{result}}",
			"@index":  "{{@index}}",
		})
		return result
	}

	switch rt.format {
	case FormatXML:
		return `You are a helpful AI assistant that reasons step by step.
When solving problems, think out loud by following this format:

<thought>
Your reasoning at this step
</thought>
<action>
The action you will take
</action>
<result>
The result of your action
</result>

Continue until you reach the answer.`

	case FormatJSON:
		return `You are a helpful AI assistant that reasons step by step.
When solving problems, respond in JSON format:

{
  "thought": "Your reasoning at this step",
  "action": "The action you will take",
  "result": "The result of your action"
}

Continue until you reach the answer.`

	case FormatNumbered:
		return `You are a helpful AI assistant that reasons step by step.
When solving problems, think out loud using this format:

Step 1: [Your thought]
Action 1: [The action you take]
Result 1: [The result]

Step 2: [Your next thought]
...

Continue until you reach the answer.`

	default:
		return `You are a helpful AI assistant that reasons step by step.
Think out loud and show your reasoning process.
Take actions and observe results.
Continue until you reach the answer.`
	}
}

// ParseSteps extracts reasoning steps from the model's response.
func (rt *ReasoningTemplate) ParseSteps(response string) ([]Step, error) {
	var steps []Step

	switch rt.format {
	case FormatXML:
		steps = rt.parseXMLSteps(response)
	case FormatJSON:
		steps = rt.parseJSONSteps(response)
	case FormatNumbered:
		steps = rt.parseNumberedSteps(response)
	default:
		steps = rt.parsePlainSteps(response)
	}

	// Apply stopAtStep limit
	if rt.stopAtStep > 0 && len(steps) > rt.stopAtStep {
		steps = steps[:rt.stopAtStep]
	}

	return steps, nil
}

func (rt *ReasoningTemplate) parseXMLSteps(response string) []Step {
	var steps []Step
	lines := strings.Split(response, "\n")

	var current *Step
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "<thought>") {
			if current != nil {
				steps = append(steps, *current)
			}
			current = &Step{}
			current.Thought = strings.TrimPrefix(line, "<thought>")
			current.Thought = strings.TrimSuffix(current.Thought, "</thought>")
		} else if strings.HasPrefix(line, "<action>") {
			if current == nil {
				current = &Step{}
			}
			current.Action = strings.TrimPrefix(line, "<action>")
			current.Action = strings.TrimSuffix(current.Action, "</action>")
		} else if strings.HasPrefix(line, "<result>") {
			if current == nil {
				current = &Step{}
			}
			current.Result = strings.TrimPrefix(line, "<result>")
			current.Result = strings.TrimSuffix(current.Result, "</result>")
		}
	}
	if current != nil {
		steps = append(steps, *current)
	}
	return steps
}

func (rt *ReasoningTemplate) parseJSONSteps(response string) []Step {
	// Simple JSON parsing without external dependency
	var steps []Step
	// For now, just do basic extraction
	// Full JSON parsing would require encoding/json
	return steps
}

func (rt *ReasoningTemplate) parseNumberedSteps(response string) []Step {
	var steps []Step
	lines := strings.Split(response, "\n")

	var current *Step
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "Step ") || strings.HasPrefix(line, "Step ") {
			if current != nil {
				steps = append(steps, *current)
			}
			current = &Step{}
			// Extract thought after "Step N:"
			if idx := strings.Index(line, ":"); idx >= 0 {
				current.Thought = strings.TrimSpace(line[idx+1:])
			}
		} else if strings.HasPrefix(line, "Action ") {
			if current == nil {
				current = &Step{}
			}
			if idx := strings.Index(line, ":"); idx >= 0 {
				current.Action = strings.TrimSpace(line[idx+1:])
			}
		} else if strings.HasPrefix(line, "Result ") {
			if current == nil {
				current = &Step{}
			}
			if idx := strings.Index(line, ":"); idx >= 0 {
				current.Result = strings.TrimSpace(line[idx+1:])
			}
		}
	}
	if current != nil {
		steps = append(steps, *current)
	}
	return steps
}

func (rt *ReasoningTemplate) parsePlainSteps(response string) []Step {
	var steps []Step
	lines := strings.Split(response, "\n")

	var current *Step
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// Simple heuristic: non-empty lines are thoughts
		if current != nil && current.Thought != "" {
			steps = append(steps, *current)
		}
		current = &Step{Thought: line}
	}
	if current != nil {
		steps = append(steps, *current)
	}
	return steps
}

// ToConversation converts the reasoning template to a conversation.
func (rt *ReasoningTemplate) ToConversation(userInput string) *Conversation {
	systemPrompt := rt.BuildSystemPrompt()

	conv := NewConversation("cot-conversation")
	conv.AddSystem(MustNewTemplate("system", systemPrompt))
	conv.AddUser(MustNewTemplate("user", userInput))

	return conv
}

// TreeOfThought represents branching reasoning paths.
type TreeOfThought struct {
	root        *ToTNode
	maxDepth    int
	branchLimit int
}

// ToTNode represents a single node in the reasoning tree.
type ToTNode struct {
	Thought  string
	Score    float32
	Parent   *ToTNode
	Children []*ToTNode
	Depth    int
	Visited  bool
}

// NewTreeOfThought creates a new tree of thought.
func NewTreeOfThought() *TreeOfThought {
	return &TreeOfThought{
		maxDepth:    5,
		branchLimit: 3,
	}
}

// WithMaxDepth sets the maximum reasoning depth.
func (tot *TreeOfThought) WithMaxDepth(depth int) *TreeOfThought {
	tot.maxDepth = depth
	return tot
}

// WithBranchLimit sets the maximum branches per node.
func (tot *TreeOfThought) WithBranchLimit(limit int) *TreeOfThought {
	tot.branchLimit = limit
	return tot
}

// AddRoot adds a root thought.
func (tot *TreeOfThought) AddRoot(thought string) *ToTNode {
	node := &ToTNode{
		Thought: thought,
		Depth:   0,
	}
	tot.root = node
	return node
}

// AddChild adds a child node to a parent.
func (n *ToTNode) AddChild(thought string, score float32) *ToTNode {
	child := &ToTNode{
		Thought: thought,
		Score:   score,
		Parent:  n,
		Depth:   n.Depth + 1,
	}
	n.Children = append(n.Children, child)
	return child
}

// BestPath returns the best reasoning path based on scores.
func (tot *TreeOfThought) BestPath() []*ToTNode {
	if tot.root == nil {
		return nil
	}

	var bestPath []*ToTNode
	var bestScore float32

	var dfs func(node *ToTNode, path []*ToTNode)
	dfs = func(node *ToTNode, path []*ToTNode) {
		path = append(path, node)
		if len(node.Children) == 0 || node.Depth >= tot.maxDepth {
			// Calculate path score
			var total float32
			for _, n := range path {
				total += n.Score
			}
			if total > bestScore {
				bestScore = total
				bestPath = make([]*ToTNode, len(path))
				copy(bestPath, path)
			}
			return
		}

		// Sort children by score
		children := make([]*ToTNode, len(node.Children))
		copy(children, node.Children)
		for i := 0; i < len(children)-1; i++ {
			for j := i + 1; j < len(children); j++ {
				if children[j].Score > children[i].Score {
					children[i], children[j] = children[j], children[i]
				}
			}
		}

		// Only explore top branches
		for i := 0; i < tot.branchLimit && i < len(children); i++ {
			dfs(children[i], path)
		}
	}

	dfs(tot.root, nil)
	return bestPath
}

// String returns a string representation of the path.
func (tot *TreeOfThought) String() string {
	path := tot.BestPath()
	if path == nil {
		return ""
	}

	var sb strings.Builder
	for i, node := range path {
		if i > 0 {
			sb.WriteString("\n-> ")
		}
		sb.WriteString(node.Thought)
		if node.Score > 0 {
			sb.WriteString(fmt.Sprintf(" (score: %.2f)", node.Score))
		}
	}
	return sb.String()
}

// ReflectionTemplate provides self-reflection prompting.
type ReflectionTemplate struct {
	critiqueTemplate    *Template
	improvementTemplate *Template
}

// NewReflectionTemplate creates a new reflection template.
func NewReflectionTemplate() *ReflectionTemplate {
	return &ReflectionTemplate{
		critiqueTemplate: MustNewTemplate("critique", `Review your previous response:
{{response}}

For each point below, rate yourself and provide improvement suggestions:
1. Correctness: Is the information accurate?
2. Completeness: Did you address all parts of the question?
3. Clarity: Is the explanation clear and understandable?
4. Relevance: Did you stay on topic?

Provide your critique and suggested improvements.`),
		improvementTemplate: MustNewTemplate("improvement", `Based on the critique:
{{critique}}

Rewrite your improved response, addressing the identified issues.`),
	}
}

// BuildCritiqueRequest creates a critique request.
func (rt *ReflectionTemplate) BuildCritiqueRequest(response string) *protocol.ChatRequest {
	content, _ := rt.critiqueTemplate.Render(map[string]any{
		"response": response,
	})
	return &protocol.ChatRequest{
		Messages: []protocol.Message{
			{Role: protocol.RoleUser, Content: content},
		},
	}
}

// BuildImprovementRequest creates an improvement request.
func (rt *ReflectionTemplate) BuildImprovementRequest(critique string) *protocol.ChatRequest {
	content, _ := rt.improvementTemplate.Render(map[string]any{
		"critique": critique,
	})
	return &protocol.ChatRequest{
		Messages: []protocol.Message{
			{Role: protocol.RoleUser, Content: content},
		},
	}
}
