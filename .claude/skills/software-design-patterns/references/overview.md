# Design Patterns Overview

Load this when the user asks what a design pattern is, why to use one, or how the three GoF categories differ.

Canonical external source: [refactoring.guru/design-patterns](https://refactoring.guru/design-patterns) — cite and defer to it for deeper diagrams and per-language variants. Concepts are language-agnostic; examples in this skill are in TypeScript.

## What design patterns are (and are not)

A design pattern is a **typical, reusable solution to a recurring design problem** in object-oriented software. It is *not*:

- **Not a finished piece of code you copy-paste.** A pattern is a general concept — a description of how to structure classes and objects to solve a problem. You adapt it to your situation and write the code yourself.
- **Not an algorithm.** An algorithm defines a precise sequence of steps to reach a goal. A pattern is a higher-level *blueprint* for structure: you can implement the same pattern many different ways.
- **Not a library or framework.** Patterns describe relationships and responsibilities, not specific APIs.

Think of a pattern as a blueprint you customize, versus an algorithm which is a concrete recipe.

### Why patterns are valuable

- **Shared vocabulary.** Saying "use a Factory here" or "this is a Strategy" communicates an entire design intent in one or two words. Patterns are a vocabulary for design discussions.
- **Proven, time-tested structure.** Patterns are battle-tested solutions; using them helps you avoid subtle problems that surface only later, and reassures reviewers that the approach is sound.
- **They teach you to design for change.** Even when you do not apply a pattern verbatim, knowing them trains you to spot the object-oriented principles (program to an interface, favor composition over inheritance, encapsulate what varies) that make code flexible.

### The danger: overuse

Patterns are tools, not goals. Common failure modes:

- **Forcing patterns where plain code is simpler and clearer.** A pattern adds indirection. If a problem is solved by a function, a `switch`, or a small class, a pattern usually makes it *worse*, not better. Inexperience plus enthusiasm for patterns leads to over-engineered designs.
- **Cargo-culting structure** without understanding the problem each pattern solves. Apply a pattern only when you feel the *pain* it is designed to relieve.
- **Premature abstraction.** Wait until variation actually appears (or is clearly imminent) before introducing the flexibility a pattern provides.

Rule of thumb: reach for a pattern when you recognize its *problem*, not because you want to use the pattern. If a junior could read the simpler version faster and it would not need to change, prefer the simpler version.

## The three categories

The GoF patterns fall into three groups based on *what kind of problem* they address.

| Category | Concern | Reach for it when... |
|---|---|---|
| **Creational** | *How objects get created.* Make object creation flexible, decoupled, and reusable. | **"I need to create objects flexibly"** — control *which* class is instantiated, hide construction details, reuse or limit instances, or build complex objects step by step. |
| **Structural** | *How objects and classes are composed.* Assemble objects into larger structures while keeping them flexible and efficient. | **"I need to compose / assemble objects and classes"** — make incompatible interfaces work together, add responsibilities, wrap or simplify, share data, or treat groups uniformly. |
| **Behavioral** | *How objects interact.* Manage algorithms, responsibilities, and communication between objects. | **"I need to manage algorithms, responsibilities, or communication"** — vary behavior at runtime, decouple senders from receivers, traverse collections, or coordinate many objects. |

### Quick decision guide

1. **Is the problem about *making* objects?** → Creational. (Which concrete class? How is it built? One instance or many?)
2. **Is the problem about *arranging* existing objects into a structure?** → Structural. (How do these pieces fit together? How do I add to or simplify them without rewriting?)
3. **Is the problem about *what objects do and how they talk*?** → Behavioral. (How do they coordinate? How does behavior change over time or context?)

## Pattern intents (one line each)

| Pattern | Intent |
|---|---|
| Factory Method | Define an interface for creating an object, but let subclasses decide which class to instantiate. |
| Abstract Factory | Create families of related objects without specifying their concrete classes. |
| Builder | Construct complex objects step by step; the same process can build different representations. |
| Prototype | Create new objects by cloning an existing instance. |
| Singleton | Ensure a class has exactly one instance with a global access point. |
| Adapter | Make an incompatible interface usable by wrapping it in an expected one. |
| Bridge | Split an abstraction from its implementation so they vary independently. |
| Composite | Treat individual objects and compositions of objects uniformly via a tree. |
| Decorator | Attach new behaviors by wrapping objects, without subclassing. |
| Facade | Provide a simple unified interface to a complex subsystem. |
| Flyweight | Share common state across many objects to save memory. |
| Proxy | Provide a stand-in that controls access to another object. |
| Chain of Responsibility | Pass a request along a chain of handlers until one handles it. |
| Command | Encapsulate a request as an object (params, queue, log, undo). |
| Iterator | Traverse a collection's elements without exposing its internal structure. |
| Mediator | Centralize complex communication between objects in a mediator. |
| Memento | Capture and restore an object's state without violating encapsulation. |
| Observer | Notify many dependent objects automatically when one changes state. |
| State | Let an object alter its behavior when its internal state changes. |
| Strategy | Define a family of interchangeable algorithms and swap them at runtime. |
| Template Method | Define an algorithm's skeleton, deferring steps to subclasses. |
| Visitor | Add new operations to an object structure without changing its classes. |
| Interpreter | Define a grammar and an interpreter for sentences in a language. |
