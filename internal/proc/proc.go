// Package proc makes a started command safe to cancel. A cancelled context must end
// the whole tree the command started, not only the process Span launched: agent CLIs
// and test runners spawn children of their own, and a child left running keeps
// editing the working tree after the run says it stopped.
package proc

import "time"

// Grace is how long a cancelled command gets to exit after the polite signal before
// everything it started is killed.
var Grace = 5 * time.Second

// waitDelay bounds Wait after a cancel. Without it, Wait reads the output pipes until
// every process that inherited them closes them, and an orphan can hold them open.
const waitDelay = 5 * time.Second
