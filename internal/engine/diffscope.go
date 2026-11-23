package engine

import (
	"context"
	"strings"

	"span/internal/prove"
)

// changer is a source that can list what the branch changed since it left its base. It is
// optional: a source without it keeps prove exactly as it was.
type changer interface {
	Changed(ctx context.Context, base string) ([]string, error)
}

// focusOnDiff narrows a plan that nothing else focused to the tests the branch's diff can
// affect, so a repair reruns those instead of the whole suite. It needs a recorded baseline:
// imports are not the only dependencies (a test that runs a binary, a file read at runtime),
// so the wide band still runs once per commit and a wide failure the base commit did not
// have blocks. Without a baseline, which failures are new is unknown, and the plan stays as
// it was. Anything that cannot be narrowed safely returns plan unchanged. The baseline
// never comes here: on the base commit there is no diff.
func (r *runner) focusOnDiff(ctx context.Context, plan prove.Plan) prove.Plan {
	if len(plan.Focused) > 0 || len(r.opt.Policy.Prove.Focused) > 0 || !r.opt.Policy.DiffScopeOn() {
		return plan
	}
	source, ok := r.opt.Source.(changer)
	if !ok || r.cur.Base == "" {
		return plan
	}
	if plan.Excuse == nil {
		r.logLine("prove: not focused on the diff: no baseline was recorded, so the wide band decides")
		return plan
	}
	wide := plan.Fast
	if len(wide) == 0 {
		wide = r.opt.Policy.Prove.Fast
	}
	changed, err := source.Changed(ctx, r.cur.Base)
	if err != nil {
		r.logLine("prove: not focused on the diff: " + err.Error())
		return plan
	}
	focused := prove.DiffFocus(ctx, r.opt.Check, r.opt.Workspace, wide, changed)
	if len(focused) == 0 {
		r.logLine("prove: not focused on the diff: no safe narrowing for these changes; the wide band decides")
		return plan
	}
	narrowed := plan
	narrowed.Focused = focused
	narrowed.Fast = r.opt.Policy.Prove.Fast
	if r.cur.WideSHA != "" && r.cur.WideSHA == r.cur.GitSHA {
		// The whole suite already passed on these exact bytes.
		narrowed.Fast = nil
	}
	narrowed.WideBlocks = true
	// The package list changes between repairs; the failure it finds must not look new.
	narrowed.FocusLabel = strings.TrimSpace(wide[0])
	r.logLine("prove: focused on the diff: " + strings.Join(focused, "; "))
	return narrowed
}
