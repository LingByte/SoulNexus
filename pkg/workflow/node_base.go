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
				// Debug: log what we're looking for and what's available
				if ctx != nil {
					ctx.AddLog("debug", fmt.Sprintf("ResolveValue failed for %s (source=%s). Available NodeData keys: %v", alias, source, getNodeDataKeys(ctx)), n.ID, n.Name)
				}
				return nil, fmt.Errorf("node %s missing input %s from %s", n.Name, alias, source)
			}
		} else {
			result[alias] = val
		}
	}
	return result, nil
}

// Helper function to get all keys in NodeData for debugging
func getNodeDataKeys(ctx *WorkflowContext) []string {
	if ctx == nil || ctx.NodeData == nil {
		return []string{}
	}
	keys := make([]string, 0, len(ctx.NodeData))
	for k := range ctx.NodeData {
		keys = append(keys, k)
	}
	return keys
}

// PersistOutputs writes outputs into context according to mapping
func (n *Node) PersistOutputs(ctx *WorkflowContext, outputs map[string]interface{}) {
	if n == nil || len(outputs) == 0 {
		return
	}

	if len(n.OutputParams) > 0 {
		// If OutputParams is defined, use it to map outputs
		for alias, target := range n.OutputParams {
			if val, ok := outputs[alias]; ok {
				ctx.SetData(target, val)
				if ctx != nil {
					ctx.AddLog("debug", fmt.Sprintf("PersistOutputs: stored %s -> %s", alias, target), n.ID, n.Name)
				}
			}
		}
	} else {
		// If OutputParams is not defined, store all outputs using nodeId.outputName format
		// This allows downstream nodes to access the outputs
		for key, val := range outputs {
			target := fmt.Sprintf("%s.%s", n.ID, key)
			ctx.SetData(target, val)
			if ctx != nil {
				ctx.AddLog("debug", fmt.Sprintf("PersistOutputs (auto): stored %s -> %s", key, target), n.ID, n.Name)
			}
		}
	}
}
