package workflow

import "fmt"

// ConditionNode evaluates expressions and pushes results into context
type ConditionNode struct {
	Node
	Expression string
	Evaluator  func(ctx *WorkflowContext, expression string) (bool, error)
}

func (c *ConditionNode) Evaluate(ctx *WorkflowContext) (bool, error) {
	fmt.Printf("Evaluating condition node %s\n", c.Name)
	if c.Evaluator != nil {
		return c.Evaluator(ctx, c.Expression)
	}
	return c.Expression != "", nil
}

func (c *ConditionNode) Base() *Node {
	return &c.Node
}

func (c *ConditionNode) Run(ctx *WorkflowContext) ([]string, error) {
	result, err := c.Evaluate(ctx)
	if err != nil {
		return nil, err
	}
	if ctx != nil {
		if ctx.NodeData == nil {
			ctx.NodeData = make(map[string]interface{})
		}
		key := c.Properties["result_key"]
		if key == "" {
			key = c.ID + "_result"
		}
		ctx.NodeData[key] = result
	}
	next := c.pickNext(result)
	if next == "" {
		return c.NextNodes, nil
	}
	return []string{next}, nil
}

func (c *ConditionNode) pickNext(result bool) string {
	if result {
		if next := c.Properties["true_next"]; next != "" {
			return next
		}
		if len(c.NextNodes) > 0 {
			return c.NextNodes[0]
		}
		return ""
	}
	if next := c.Properties["false_next"]; next != "" {
		return next
	}
	if len(c.NextNodes) > 1 {
		return c.NextNodes[1]
	}
	return ""
}
