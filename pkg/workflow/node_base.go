package workflow

import "fmt"

// ExecutableNode unify all concrete nodes
type ExecutableNode interface {
	Base() *Node
	Run(ctx *WorkflowContext) ([]string, error)
}

type Node struct {
	ID           string                           // the only one node ID
	Name         string                           // node name
	Type         NodeType                         // node type
	InputParams  map[string]string                // input params
	OutputParams map[string]string                // output params
	NextNodes    []string                         // next node ids
	PrevNodes    []string                         // previous node ids
	Properties   map[string]string                // node properties
	Execute      func(ctx *WorkflowContext) error // node execute function
}

func (n *Node) Base() *Node {
	return n
}

func (n *Node) Run(ctx *WorkflowContext) ([]string, error) {
	if n == nil {
		return nil, fmt.Errorf("nil node cannot run")
	}
	if n.Execute != nil {
		if err := n.Execute(ctx); err != nil {
			return nil, err
		}
	}
	return n.NextNodes, nil
}

// PrepareInputs resolves Node.InputParams from context
func (n *Node) PrepareInputs(ctx *WorkflowContext) (map[string]interface{}, error) {
	if n == nil || len(n.InputParams) == 0 {
		return map[string]interface{}{}, nil
	}
	result := make(map[string]interface{}, len(n.InputParams))
	for alias, source := range n.InputParams {
		val, ok := ctx.ResolveValue(source)
		if !ok {
			// If source is the same as alias (e.g., "input-0" -> "input-0"),
			// it means the node expects data from previous node's output with the same name
			// In this case, provide a default nil value instead of failing
			if source == alias {
				// This is likely a placeholder mapping, provide nil as default
				result[alias] = nil
			} else {
				return nil, fmt.Errorf("node %s missing input %s from %s", n.Name, alias, source)
			}
		} else {
			result[alias] = val
		}
	}
	return result, nil
}

// PersistOutputs writes outputs into context according to mapping
func (n *Node) PersistOutputs(ctx *WorkflowContext, outputs map[string]interface{}) {
	if n == nil || len(n.OutputParams) == 0 || len(outputs) == 0 {
		return
	}
	for alias, target := range n.OutputParams {
		if val, ok := outputs[alias]; ok {
			ctx.SetData(target, val)
		}
	}
}
