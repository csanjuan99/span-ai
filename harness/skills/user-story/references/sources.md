# Sources

Load when asked why the format is this way. The procedure is the operational cut. The story does not cite these.

## Card, conversation, confirmation

Ron Jeffries, "Essential XP: Card, Conversation, Confirmation." A card is a token. The requirement is the conversation. Confirmation is an acceptance test with concrete values, agreed before the work is called done.

- https://ronjeffries.com/xprog/articles/expcardconversationconfirmation/
- https://ronjeffries.com/articles/019-01ff/stories-3cs-yet-again/

Operational cut: a thin prompt is still in the conversation. Ask, then stop. Do not freeze confirmation by inventing rules. A specified prompt already finished that conversation: write the tests, do not ask the answered facts again.

## INVEST, and one outcome

Bill Wake, "INVEST in Good Stories, and SMART Tasks" (XP123, 2003). Independent, Negotiable, Valuable, Estimable, Small, Testable. A story is negotiable essence, valuable to the customer, and testable. Mike Cohn popularized the checklist in *User Stories Applied* (2004), chapter 2.

- https://xp123.com/invest-in-good-stories-and-smart-tasks/
- https://www.oreilly.com/library/view/user-stories-applied/0321205685/ch02.html

Jeff Patton, *User Story Mapping* (O'Reilly, 2014). Slice by outcome across the journey, not by component layer.

Richard Lawrence's splitting patterns, as used by the SAFe User Story Primer and the Agile Alliance / Humanizing Work flowchart: workflow, business-rule variation, major effort, simple/complex, data variation. Split only into slices a user could receive.

- https://framework.scaledagile.com/wp-content/uploads/2021/07/User-Story-Primer.pdf
- http://www.agileforall.com/wp-content/uploads/2012/01/Story-Splitting-Flowchart.pdf

Operational cut: one user outcome per story. Quality of that outcome stays. Two origins of the same result stay. A second capability stops for a split. `Para` is the value, not the task.

## Rules are not layout

Business Rules Manifesto, Business Rules Group. Rules are a separate part of the model (1.1), not the process (2.2), never assumed (3.3), declarative rather than procedural (4.1–4.3), and an exception is another rule (4.7). A statement that smuggles in implementation is not a rule (4.4).

- https://www.businessrulesgroup.org/brmanifesto.htm

Operational cut: BR is one declarative constraint. Colors, pixels, fonts, and component order are AC, and only when the prompt already fixed them. Do not add a BR that exists only to lock a visual.

## Criteria say what; examples say which value

ISTQB Certified Tester, Acceptance Testing (CT-AT), 2019 syllabus. Acceptance criteria are measurable true/false statements. They are the test conditions: what to test, not the procedure. Acceptance test cases are derived from them and say how. Traceability runs story → criteria → test cases, and coverage is checked from that trace.

Gojko Adzic, Specification by Example. A criterion is the rule ("more than a month ago"). An example is a concrete value. Given-When-Then became the default example form (71% in his ten-year survey). Use a few key examples, especially boundaries, not a paraphrase of every sentence.

- https://astqb.org/assets/documents/Acceptance-Testing-Syllabus-2019.pdf
- https://gojko.net/2020/03/17/sbe-10-years.html
- https://dannorth.net/blog/the-mystery-of-the-missing-date/
- https://gojko.net/2014/05/05/focus-on-key-examples/

Operational cut: AC is pass/fail and points at the BR with `Confirma:`. TC is Dado / Cuando / Entonces with data, points at the AC with `Cubre`, and includes the boundary where a number or date changes. One TC per AC is not the goal.

## The conversation, mapped

Matt Wynne, "Introducing Example Mapping" (Cucumber blog, 2015). Four card colors: story, rules, examples, questions. Rules get examples; what nobody can answer becomes a question. Many questions mean the story is not ready; many rules mean it should be split.

Operational cut: rule = BR, example = TC, question = open item. The facilitation loop (give me an example, what happens if, is that always true, is that the same rule) is how a rule is tested before it is written. See `references/interview.md`.

George Dinwiddie, "Three Amigos" (writings on the practice). Business, development, and testing read the same story before work starts, because each sees a different gap.

Operational cut: three lenses over the map each turn. Business checks the outcome and persona, development reads the repository and cites the path, testing hunts the missing boundary. One agent, three reads; no extra roles.

Definition of Ready, used as a gate, not a phase. Its known criticism is that it turns into a waterfall hand-off where nothing starts until every box is ticked, and it invites inventing details to pass the gate.

Operational cut: ready means no open question and every fact labeled. An assumption the person accepts counts as closed and lands in `Supuestos`. The gate never licenses inventing a rule to clear a question.

## What this format deliberately is not

Amazon's Working Backwards PR/FAQ and a Google-style design doc are different artifacts. This skill does not produce them. The repeatable shape comes from the sources above, applied to the Como / Quiero / Para card and the BR, AC, and TC blocks.
