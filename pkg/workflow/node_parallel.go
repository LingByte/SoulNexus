package workflow

import "fmt"

// ParallelNode fan-out tasks to multiple nodes
type ParallelNode struct {
	Node
	Branches []string
	Runner   func(ctx *WorkflowContext, branch string) error
}

func (p *ParallelNode) Execute(ctx *WorkflowContext) error {
	fmt.Printf("Executing parallel node %s with %d branches\n", p.Name, len(p.Branches))
	for _, branch := range p.Branches {
		if p.Runner != nil {
			if err := p.Runner(ctx, branch); err != nil {
				return err
			}
			continue
		}
		fmt.Printf("Sequentially executing branch %s from parallel node %s\n", branch, p.Name)
	}
	return nil
}

func (p *ParallelNode) Base() *Node {
	return &p.Node
}

func (p *ParallelNode) Run(ctx *WorkflowContext) ([]string, error) {
	if err := p.Execute(ctx); err != nil {
		return nil, err
	}
	return p.NextNodes, nil
}
