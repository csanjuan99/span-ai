# Primary Sources for SOLID

Load this when the user asks for provenance, exact wording, or further reading on a principle.

| Source | Use it for |
| --- | --- |
| Robert C. Martin, "Design Principles and Design Patterns" (2000, Object Mentor paper) | First published definitions of OCP, LSP, ISP, DIP (and package principles); SRP named in the 2002 book. Widely mirrored. |
| Robert C. Martin, *Agile Software Development, Principles, Patterns, and Practices* (2002), chapters 8–12 | One chapter per principle with worked examples and the original violation symptoms. |
| Robert C. Martin, *Clean Architecture* (2017), Part III, chapters 7–11 | Revised definitions (SRP as "one actor"), component-level application, and architectural consequences. |
| Robert C. Martin, "The Single Responsibility Principle", blog.cleancoder.com (2014-05-08) | Clarifies SRP as reason-to-change and actor, correcting the "do one thing" misreading. https://blog.cleancoder.com/uncle-bob/2014/05/08/SingleReponsibilityPrinciple.html |
| Robert C. Martin, "The Open Closed Principle", blog.cleancoder.com (2014-05-12) | Practical scope of OCP and why full closure is unattainable. https://blog.cleancoder.com/uncle-bob/2014/05/12/TheOpenClosedPrinciple.html |
| Barbara Liskov, "Data Abstraction and Hierarchy", OOPSLA 1987 keynote | Original statement of substitutability that became LSP. |
| Barbara Liskov and Jeannette Wing, "A Behavioral Notion of Subtyping", ACM TOPLAS (1994) | Formal behavioral rules: preconditions, postconditions, invariants, history constraint. |
| Bertrand Meyer, *Object-Oriented Software Construction* (1988) | Origin of the open/closed phrasing, originally framed around inheritance. |
| Martin Fowler, "Inversion of Control Containers and the Dependency Injection pattern" (2004) | Distinguishes dependency injection mechanisms (constructor, setter, interface) from the DIP principle. https://martinfowler.com/articles/injection.html |
| refactoring.guru, *Dive Into Design Patterns*, "SOLID Principles" chapter | Secondary source with illustrated examples; useful for teaching, not for canonical wording. https://refactoring.guru/design-patterns/book |

Citation guidance: attribute SRP wording to Martin (2002) and ISP, DIP wording to Martin (2000); OCP wording to Meyer (1988) with Martin's abstraction-based reinterpretation; LSP to Liskov (1987) and Liskov & Wing (1994).
