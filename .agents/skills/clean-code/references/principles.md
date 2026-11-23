# DRY, KISS, YAGNI — and Their Counter-Warnings

Load this when deciding whether to abstract duplicated code, when a design feels over-engineered, or when speculative flexibility is being added.

**DRY (Don't Repeat Yourself).** Every piece of *knowledge* should have one authoritative home. The failure mode it prevents: a rule encoded in five places, four of which get updated when the rule changes and one of which silently keeps the old behavior. **Counter-warning:** DRY is about knowledge, not about text that happens to look alike. Two snippets that are identical today but change for different reasons are *coincidental* duplication; merging them couples unrelated things, and the wrong abstraction is more expensive to unwind than duplication is to tolerate. Follow the **rule of three**: duplicate once freely, notice at the second repeat, and abstract on the third — by then you understand the shared shape well enough to name it.

**KISS (Keep It Simple).** Prefer the straightforward solution over the impressive one. Complexity is a tax paid on every future read and change. A plain loop or a flat conditional that any reader understands beats a generic, configurable framework solving a problem you have once.

**YAGNI (You Aren't Gonna Need It).** Build for the requirements you have, not the ones you imagine. Speculative generality — extra parameters, abstract base classes, plugin points "for later" — is code you must carry, test, and read now, for a future that usually arrives different from what you guessed, or never arrives. Delete the hooks nobody calls.

These three pull against each other on purpose. The judgment is knowing which tax you're paying: duplication's maintenance cost, or abstraction's comprehension-and-coupling cost. When unsure, lean toward the simple, duplicated version — it is the cheaper mistake to fix.

## SOLID

SOLID has its own skill: `../../solid-principles/SKILL.md`. Load that skill when the pain is a class with several reasons to change, a conditional edited for every new case, a subtype that breaks its base contract, a fat interface, or policy welded to a concrete dependency.
