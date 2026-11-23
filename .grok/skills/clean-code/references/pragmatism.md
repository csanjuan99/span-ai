# Pragmatism: When to Break the Rules

Load this when a rule and clarity conflict, when the code is a throwaway script or prototype, or when a hot path justifies density.

These rules serve readability and maintainability; they are not the goal themselves. Break one when keeping it would make the code worse.

- **Readability beats purity.** If splitting a function into six tiny pieces scatters one coherent thought across the file, leave it whole. If a touch of duplication reads clearer than a strained abstraction, keep the duplication.
- **Team conventions beat personal preference.** A consistent codebase that follows a different style than you'd choose is more valuable than a locally "perfect" file that fights its surroundings. Match what's there; raise style changes as a team decision, not a drive-by edit.
- **Don't gold-plate.** A throwaway script, a migration that runs once, a prototype validating an idea — these don't need the full treatment. Spend rigor where the code will live and change. Match effort to the code's expected lifespan and blast radius.
- **Performance and constraints sometimes win.** A hot path may justify a denser, less pretty implementation; document *why* so the next reader doesn't "clean it up" and reintroduce the problem.

When you break a rule deliberately, that's engineering judgment. When you break it by accident or laziness, that's the mess these rules exist to prevent. The difference is whether you can state the reason.
