# Sources

Load this when the user wants provenance. The skill's rules are the operational cut of these sources. They are technology-agnostic on purpose.

## AWS Well-Architected Framework

Official docs. Applied here to behaviors, not only to cloud cells.

- Operational Excellence design principle, "Make frequent, small, reversible changes." Smaller increments reduce the blast radius and make reversal faster. https://docs.aws.amazon.com/wellarchitected/latest/framework/oe-design-principles.md
- OPS05-BP09, "Make frequent, small, reversible changes." Unintended consequences are easier to see, and recovery is simpler, when the change is small and reversible. Anti-patterns include untracked schema changes and in-place overwrites with no rollback. https://docs.aws.amazon.com/wellarchitected/latest/framework/ops_dev_integ_freq_sm_rev_chg.md
- Reliability, REL 10, "How do you use fault isolation to protect your workload?" Fault isolation limits a failure to a defined boundary. Components outside the boundary stay unaffected. https://docs.aws.amazon.com/wellarchitected/latest/framework/rel-failmgmt.md
- REL10-BP03, "Use bulkhead architectures to limit scope of impact." A bad software update should reach one cell. Cells do not share state. Sharing state, or updating every cell at once, removes the boundary. https://docs.aws.amazon.com/wellarchitected/latest/framework/rel_fault_isolation_use_bulkhead.md

## Change impact and ripple

- Robert S. Arnold and Shawn A. Bohner. Impact analysis as identifying what to modify to accomplish a change, or the potential consequences of a change. *Software Change Impact Analysis*, IEEE Computer Society Press, 1996. Also their ICSM 1993 comparison framework and Bohner's ICSM 1996 process paper. Starting set and estimated set are this vocabulary.
- S. S. Yau, J. S. Collofello, and T. M. MacGregor. "Ripple effect analysis of software maintenance." COMPSAC 1978. The ripple is how a modification propagates.
- S. S. Yau and J. S. Collofello. "Some Stability Measures for Software Maintenance." *IEEE Transactions on Software Engineering*, 1980. Stability is resistance to the potential ripple effect.
- ISO/IEC/IEEE 14764, the software maintenance life-cycle standard, treats impact analysis as an activity of the modification process.

## The day-1 / day-30 failure

- Marcel Böhme, Bruno C. d. S. Oliveira, and Abhik Roychoudhury. "Regression Tests to Expose Change Interaction Errors." ESEC/FSE 2013. https://doi.org/10.1145/2491411.2491430 On GNU Coreutils, one third of the regressions took more than two years to fix, two thirds of those long-standing regressions came from change interactions, and stressing each change in isolation exposed only half of the interaction errors.
- Feature interaction, as studied in the telecommunications literature, is the related case where each behavior is correct alone and the combination is not. The shared-surface trace covers it.

## How to change a system safely

- Michael C. Feathers. *Working Effectively with Legacy Code.* Prentice Hall, 2004. Adding a feature means adding behavior while holding existing behavior constant. The change algorithm is: identify change points, find test points, break dependencies, write tests, then make the change. A seam is a place where behavior can be altered without editing that place. A characterization test pins current behavior. An interception point is where the effect of the change can be detected, as close to the change as possible.
- Meilir Page-Jones. Connascence: two elements are connascent when a change to one requires a change to the other to preserve correctness. *Fundamentals of Object-Oriented Design in UML*, 1999. Name, type, meaning, position, and algorithm are static forms. Execution, timing, value, and identity are dynamic forms and are easy for a session that only reads imports to miss.
- Martin Fowler. Shotgun surgery and divergent change. *Refactoring.* A long estimated set means the rule is smeared across modules. Gather it. Do not celebrate the long edit list.
- Donella H. Meadows. *Thinking in Systems*, 2008. Feedback delays hide the consequence of an intervention. Shortening that delay is a high-leverage move. The pin is that move.
- M. M. Lehman. The law of increasing complexity: as a system evolves, complexity increases unless work is spent to contain it. Unpinned agent sessions are that unpaid work.
