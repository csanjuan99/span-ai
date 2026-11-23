# Adapt to this repo

Run this before filling the contract. The goal is to write the contract in the form this codebase already uses.

## 1. Language and sibling

Find the module that will own the capability. Open one existing capability there that a caller already uses. That file is the sibling.

Take from the sibling, and from nowhere else:

- language and file suffix
- names of request, response, and error types
- how an error is constructed
- the layer that talks to callers, and the layer that holds the rule
- how a test of that capability is named and run

No sibling means there is no local pattern yet. Use the language of the module you will edit. Mark Wire as Assumption. Do not import a framework to create a pattern.

## 2. Carrier

Stop at the first carrier that already exists for this module:

| Found | Write the contract in |
| --- | --- |
| OpenAPI, GraphQL, protobuf, Smithy, AsyncAPI, JSON Schema, or Pact | that file, next to the sibling operation |
| Exported request and response types at the boundary | those types |
| None of the above | new boundary types beside the sibling, in the same language |

Do not create a second carrier. A repo with handlers and no spec stays types. A repo with a spec changes the spec first, then the code.

## 3. Wire

Copy the sibling's binding: method and path, procedure name, message name, or function signature. The new operation differs in its name and fields, not in its style.

With no sibling, write the smallest binding a caller needs to send the input and receive the output. Mark it Assumption. Do not invent a resource layout, a version prefix, or an envelope the repo does not use.

## 4. Skills named in the command

Read the command. A skill is named when the command gives its name or a path to its `SKILL.md`.

Apply every named skill, in the order written, while implementing the code behind the contract. Read each `SKILL.md` before that implementation. The contract template is still filled first, from this skill.

These are not names:

- "best practices", "clean", "solid", or "the usual skills", unless the command also names the skill
- a skill you happen to know exists
- this skill's own references

If the command names no skill, the Companion skills line is `none`.

If a named skill cannot be found, write `not found` next to it and continue. Do not replace it with a different skill.

If two named skills disagree, the one written last in the command wins. Record both and the winner.

A named skill may shape the code behind the boundary. It does not replace the contract, and it does not add a framework this skill forbids.

## 5. Local style is not a skill

Matching the sibling is mandatory even when no skill is named. That match covers names, types, errors, folders, and tests. It does not give you permission to load an unnamed skill "to be safe".
