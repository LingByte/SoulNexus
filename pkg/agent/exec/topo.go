package exec

import (
	"fmt"
	"github.com/LingByte/SoulNexus/pkg/agent/plan"
)

func topoOrder(tasks []plan.Task) ([]plan.Task, error) {
	byID := map[string]plan.Task{}
	inDeg := map[string]int{}
	deps := map[string][]string{}
	children := map[string][]string{}

	for _, t := range tasks {
		if t.ID == "" {
			return nil, fmt.Errorf("task id is empty")
		}
		if _, ok := byID[t.ID]; ok {
			return nil, fmt.Errorf("duplicate task id: %s", t.ID)
		}
		byID[t.ID] = t
		inDeg[t.ID] = 0
		deps[t.ID] = append([]string{}, t.DependsOn...)
	}

	for id, d := range deps {
		for _, dep := range d {
			if dep == "" {
				continue
			}
			if _, ok := byID[dep]; !ok {
				return nil, fmt.Errorf("unknown dependency %q for task %q", dep, id)
			}
			inDeg[id]++
			children[dep] = append(children[dep], id)
		}
	}

	queue := make([]string, 0)
	for id, deg := range inDeg {
		if deg == 0 {
			queue = append(queue, id)
		}
	}

	out := make([]plan.Task, 0, len(tasks))
	for len(queue) > 0 {
		id := queue[0]
		queue = queue[1:]
		out = append(out, byID[id])
		for _, ch := range children[id] {
			inDeg[ch]--
			if inDeg[ch] == 0 {
				queue = append(queue, ch)
			}
		}
	}

	if len(out) != len(tasks) {
		return nil, fmt.Errorf("cycle detected in task dependencies")
	}
	return out, nil
}
