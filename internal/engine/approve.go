package engine

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"span/internal/statement"
)

const approvePrefix = "approve-"

// approvalStages are the stages whose mode can be hold: a person's gate before an effect.
var approvalStages = map[string]bool{open: true, merge: true, promote: true}

func approvalFile(stage, environment string) string {
	if stage == promote {
		return approvePrefix + promote + "-" + environment
	}
	return approvePrefix + stage
}

// Approve records a person's approval of the stage a run is held at. A stage whose mode is
// hold pauses before its effect and, without this, nothing can ever release it: a retry
// reaches the same hold. The approval is bound to the commit the person saw, so a new head
// is held again, and it is spent only when the stage advances, so a transient failure
// after approving does not ask for it a second time.
func Approve(root, id, stage, environment string) error {
	if !validID(id) {
		return os.ErrNotExist
	}
	if !approvalStages[stage] {
		return fmt.Errorf("%q has no approval; only open, merge and promote can be held", stage)
	}
	if stage == promote && !validID(environment) {
		return fmt.Errorf("promote needs the environment to approve")
	}
	snap, err := Load(root, id)
	if err != nil {
		return err
	}
	cur := snap.Cursor
	if !cur.Hold || cur.Stage != stage {
		return fmt.Errorf("this run is not held at %s", stage)
	}
	return writeAtomic(filepath.Join(root, id, approvalFile(stage, environment)), []byte(cur.GitSHA+"\n"))
}

// approved reports whether a person approved this stage for the head the run is at.
func (r *runner) approved(stage, environment string) bool {
	raw, err := os.ReadFile(filepath.Join(r.spanDir(), approvalFile(stage, environment)))
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(raw)) == r.cur.GitSHA
}

// spendApproval removes the approval once the stage it covered has advanced.
func (r *runner) spendApproval(stage, environment string) {
	_ = os.Remove(filepath.Join(r.spanDir(), approvalFile(stage, environment)))
}

// gated reports whether a stage in hold mode must still wait for a person.
func (r *runner) gated(mode, stage, environment string) bool {
	return mode == statement.Hold && !r.approved(stage, environment)
}
