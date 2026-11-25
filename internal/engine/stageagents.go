package engine

// wantedFor is the CLI one stage asks for. The run's own choice for the stage wins, then the
// stage's entry under agents: in span.yaml, then the project's agent. Empty and "any" leave
// the pick to the first ready CLI.
func (r *runner) wantedFor(stage, runChoice string) string {
	if runChoice != "" {
		return runChoice
	}
	p := r.policy()
	if choice := p.StageAgent(stage); choice != "" {
		return choice
	}
	return p.Agent
}
