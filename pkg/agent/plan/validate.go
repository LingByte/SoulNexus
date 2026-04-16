package plan

import "strings"

func Validate(p *Plan, maxTasks int) error {
	if p == nil {
		return ErrInvalidPlan
	}
	if strings.TrimSpace(p.Goal) == "" {
		return ErrInvalidPlan
	}
	if len(p.Tasks) == 0 {
		return ErrInvalidPlan
	}
	if maxTasks > 0 && len(p.Tasks) > maxTasks {
		return ErrInvalidPlan
	}

	seen := map[string]struct{}{}
	for _, t := range p.Tasks {
		id := strings.TrimSpace(t.ID)
		instr := strings.TrimSpace(t.Instruction)
		if id == "" || instr == "" {
			return ErrInvalidPlan
		}
		if _, ok := seen[id]; ok {
			return ErrInvalidPlan
		}
		seen[id] = struct{}{}
	}
	return nil
}
