package workflow

import "fmt"

// SubflowNode invokes nested workflows
type SubflowNode struct {
	Node
	Workflow *Workflow
}

func (s *SubflowNode) Execute(ctx *WorkflowContext) error {
	if s.Workflow == nil {
		fmt.Printf("Subflow node %s has no workflow attached\n", s.Name)
		return nil
	}
	fmt.Printf("Executing subflow from node: %s\n", s.Name)
	return s.Workflow.Execute()
}

func (s *SubflowNode) Base() *Node {
	return &s.Node
}

func (s *SubflowNode) Run(ctx *WorkflowContext) ([]string, error) {
	if err := s.Execute(ctx); err != nil {
		return nil, err
	}
	return s.NextNodes, nil
}
