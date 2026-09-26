---
name: fdf-brainstorm
description: Use when a feature idea has no Feature document yet, or a draft FDF feature still lacks its slug.spec.md — before any design or implementation work.
---

# FDF Brainstorm

Turn a feature idea into a validated FDF feature document and an approved
design.

New to FDF? The fdf-help skill explains how the fdf skills fit together and
the mechanics they share. The format rules are in the bundle at
`docs/fdf/SPEC.md`; `fdf help` documents every command.

**This skill is for capabilities that do not exist yet.**

- The capability is delivered (`done` or `adopted`) and must change → stop and
  use fdf-change: a second feature document for one capability is the drift
  FDF exists to prevent. If it is broken → fdf-debug first.
- The capability already exists **in the code** but no feature documents it —
  or the work extends an undocumented command, route, screen or job, even
  with behavior that is new → it is not new either: map it with fdf-adopt,
  then change it through fdf-change. Brainstorming existing behavior invents a design history it
  never had.
- The one exception: a `retired` capability that is coming back returns as a
  new feature, brainstormed here, which may name the retired one in
  `depends-on`. The retired document is never flipped back.

## Mechanics

- A `timestamp:` you set is the output of `date -u +%Y-%m-%dT%H:%M:%SZ`, run
  at the moment of the edit — never a time you estimate, round or reuse, and
  never your local time with `Z` added. A restored document keeps its old
  one; a malformed one already committed is repaired by fdf-validate's F1,
  never set to now.
- Log with single quotes — `fdf log <feature-id> '**Specified**: …'` —
  because inside double quotes the shell runs anything between backticks. An
  apostrophe would end the quotes: write ’ instead.
- `<feature-id>` below is the feature's full ID, such as
  `features/payments/instant-refunds`; paths are relative to the bundle root,
  `docs/fdf/`. The spec and surface documents are written by hand — there is
  no command for them.

## Before you design

- **Read the Context documents**: `STACK.md`, `ARCHITECTURE.md`,
  `SURFACES.md`, `INFRA.md` and `DOMAIN.md` at the bundle root. Propose an
  approach that fits the documented stack, principles and surface
  conventions, and say explicitly when a good design would depart from them
  (a new dependency, a new pattern, a new surface convention). If any is still
  an unfilled stub, stop and run fdf-init first.
- **Use the project's words.** `DOMAIN.md` gives one canonical name per
  concept and lists the words banned in its place. Use the canonical names in
  every scenario, in the spec, and in every name you propose — the feature's
  own slug included. The user may use a banned word ("store" where the
  lexicon says Venue): translate it, and say so. If the capability needs a
  concept the lexicon lacks, say so — a new term is a `DOMAIN.md` change and
  needs the user's approval like any other Context edit.
- **Steps name the concept, not the label.** The lexicon is the project's
  *internal* language; a screen, a button or a locale string may say "store"
  for a Venue on purpose, and a scenario does not change that. Write
  `When the merchant opens the Venue settings`, not
  `When I tap "Store settings"`. Where the exact wording is what is under test
  (an error message, a URL, a button label), name the outcome in the scenario
  — `Then the merchant is told the Venue is closed` — and leave the literal
  string for `slug.test.md`. Never change a label to clear an F12 warning.
- **Read the practices that will govern the code.** A practice whose
  `applies-to` covers the paths this feature will touch
  (`grep -rn -A3 -e '^status:' -e '^applies-to:' docs/fdf/practices`; a bare
  `applies-to:` lists its paths on the `- ` lines below it) constrains the
  design before you propose it. Designing against one is a legitimate thing to propose;
  designing in ignorance of one is not.

## Process

1. **Check the bundle**: run `fdf validate` (it respects `--root` and
   `FDF_ROOT_DIR`).
   - No bundle → ask the user before running `fdf init`; then fill the
     Context documents with fdf-init before going on.
   - A `still an unfilled stub` line → stop and run fdf-init.
2. **Start the feature document.**
   - A `draft` feature for this idea already exists → do not run `fdf new`;
     read it and continue from step 3.
   - Otherwise choose where to file it. Read `features/INDEX.md`: if a group
     there fits the product area, file it in that group; otherwise file it
     flat. When about ten flat features would share an area, propose a group
     to the user (moving the others is `fdf mv`, and their call), and file
     flat until they decide. Then run `fdf new [<group>/…]<slug>` (lowercase, hyphenated, in
     `DOMAIN.md`'s terms). `fdf new payments/instant-refunds` writes
     `features/payments/instant-refunds.md`, whose ID is
     `features/payments/instant-refunds`.
   - Read the generated file. A `draft` feature has no trail siblings (at most
     a log) and no task directory.
3. **Understand the feature** through questions, ONE at a time: who is the
   user, what capability, what value, which edge cases? Prefer multiple-choice
   questions with a recommended answer. **Chase ambiguous words**: when the
   user says "newest", "duplicate", "fast", ask which meaning they intend
   (newest = later in the file, or by a timestamp column?). **Chase
   precedence** the same way: when two rules can reject the same request (not
   the owner, and the date has passed), which answer does the caller get?
   Every ambiguous term or order you resolve silently is a design decision
   nobody approved.
4. **Write the feature document.** Replace everything the scaffold left:

   ````markdown
   ---
   type: Feature
   title: Instant refunds
   description: A support agent refunds a settled payment in full, at once.
   status: draft
   depends-on: [features/payments/card-payments]
   timestamp: <output of date -u +%Y-%m-%dT%H:%M:%SZ, run now for this file>
   ---

   # Feature

   ```gherkin
   Feature: Instant refunds
     As a support agent
     I want to refund a settled payment while the customer is on the call
     So that the customer hears the outcome before the call ends
   ```

   Prose between the fences: context, and links to related documents.

   # Scenarios

   ```gherkin
   Scenario: Full refund of a settled payment
     Given a settled payment of 40.00 EUR
     When the support agent refunds it in full
     Then the payment shows as refunded
   ```
   ````

   - **Replace every placeholder**: the `description: TODO — one sentence.`,
     the `<role>`, `<capability>` and `<value>` lines, and the
     `Scenario: Replace me` fence. Also replace the `TODO.` in the feature's
     listing line in the `INDEX.md` beside it with the same one-sentence
     description.
   - **One fence per scenario**, each starting with `Scenario:` (or
     `Scenario Outline:`, `Background:`, `Rule:`, or a tag); one `Feature:`
     block in the whole document.
   - **Coverage**: one happy path, one scenario per edge case the dialogue
     surfaced, plus limits and access control where they exist. Every limit
     or refusal the user states is a scenario, because someone can observe it;
     a spec line only records how something is built.
   - **Observable, not implementation**: a scenario states what a user can see
     or a system can observe. `Then the existing contact shows the new phone
     number` is observable; `Then the service upserts by normalized email` is
     implementation, and belongs in the spec.
   - **Names**: short, distinct, stable — `slug.test.md` and the tasks will
     repeat them word for word.
   - **Surface details stay out of the Gherkin** when they are not outcomes a
     user observes (layout choreography, an API envelope, a CLI flag set): they
     go in `slug.surface.md` (step 7), or in SURFACES.md when they hold for
     every surface.
   - **`depends-on`**: when the feature builds on what another feature built —
     adds a page to its screens, an endpoint or subcommand beside its own, a
     method or field to its types, a column to its table (find it with
     `grep -rn '<path of that code>' docs/fdf/features`) — list that
     feature's full ID in `depends-on` (it must exist, and there must be no
     cycle). A `done` feature and an `adopted` one count alike. When it builds
     on none, leave the field out: the example has it only because instant
     refunds build on card payments. That is lineage, not a substitute for a
     Change: when a scenario of the delivered feature would stop being true,
     you are altering it, and that is a Change.
5. **Present the design and get it approved.** Present, section by section,
   pausing for the user's reaction after each:
   1. the scenarios, by name, with one line on what each promises;
   2. the approach;
   3. the alternatives, with their trade-offs — lead with your
      recommendation;
   4. the interface, when the feature has one: the command, endpoint or
      screen, every output line and message exactly as the surface document
      will say them (step 7), and each status or exit code;
   5. the proof: how each scenario will be checked end to end. When the
      project has no such test yet, say that the plan will add the smallest
      harness (call the entry point, capture its output and exit code) and,
      through it, a check in every scenario's case, refusals included —
      unless the user prefers a manual procedure;
   6. the decisions you took yourself (a term you defined, a rule extended to
      a case the dialogue never covered, a message you worded, the order
      between two refusals), one bullet each — the gate covers your
      decisions too.

   Then ask one explicit question: **"Do you approve this design?"** Nothing
   below is written until the answer is yes. If the user asks for changes,
   make them and present the changed parts again. If the user decides not to
   build it at all, delete the draft feature and its listing line in the
   `INDEX.md` beside it, and log why in the root log:
   `fdf log '**Decision**: <feature-id> dropped before design; <why>.'`
6. **Write the spec and flip the status, in one edit.** Write
   `<feature-id>.spec.md` beside the feature file — never inside a
   `<slug>/` directory:

   ```markdown
   ---
   type: Spec
   title: Instant refunds — design
   description: Why refunds are synchronous, and what was rejected.
   timestamp: <output of date -u +%Y-%m-%dT%H:%M:%SZ, run now for this file>
   ---

   ## What is being built

   The approach, in a few sentences.

   ## Why

   The problem it solves, and for whom.

   ## Design decisions

   - One bullet per resolved ambiguity and per decision from step 5.

   ## Alternatives rejected

   - Each alternative, with the reason it lost.
   ```

   In the same edit, set the feature's `status: specified` and its
   `timestamp` (run `date -u +%Y-%m-%dT%H:%M:%SZ` now): a `draft` may not have a spec (F4), so the spec
   and the flip land together. (No rule checks the spec's headings; use these four.)
7. **Surface document.** Ask one question: does this feature add or change
   anything a person or another system uses directly — a screen, dialog or
   flow; an endpoint, RPC or webhook; a command or flag; an event, message,
   email or notification; a file format or export?
   - **Yes** → write `<feature-id>.surface.md` beside the spec, with one `#`
     heading per interface, following SURFACES.md's conventions:

     ```markdown
     ---
     type: Surface
     title: Instant refunds — surface
     description: The refund endpoint and the support console's refund button.
     timestamp: <output of date -u +%Y-%m-%dT%H:%M:%SZ, run now for this file>
     ---

     # API

     `POST /payments/{id}/refunds` — request and response shapes, status and
     error codes.

     # Support console

     Layout, states (empty, loading, error), copy, accessibility.
     ```

     A command gets its arguments, flags, output and exit codes; an event or
     message gets its fields and when it is sent. The surface document
     describes the interfaces the approved design gives the feature, and it is
     living: every later Change that alters them amends it. It never adds an
     outcome the Gherkin does not name — a new outcome is a scenario first.
   - **No** (a background job whose output nobody reads directly, a change to
     how data is stored) → add `surface: none` to the feature's frontmatter.
8. **Self-review** before validating: re-read the feature document against the
   conversation. Is there a limit or refusal no scenario states, or a user
   decision no scenario or spec line records?
   Two scenarios whose names could be confused? A banned word? Do the slug
   and title use the name the dialogue settled — if not, `fdf mv` the draft
   now? Fix it silently; don't re-ask.
9. **Log and gate.** Log the approval in the feature's own log (it is created
   on first use; feature entries never go in the root `LOG.md`):
   `fdf log <feature-id> '**Specified**: design approved by <who>; <the approach in one line>.'`
   `<who>` is the name the user gave, or "the user" — never a name you
   guessed.
   Then run `fdf validate`: it must exit 0, and use fdf-validate if it does
   not. Fix any `warn:` it prints about this feature — a leftover placeholder,
   a missing description.

Next: the feature is `specified`, and fdf-plan is the next skill. If the user
asked for the capability to be built, not only designed, continue with
fdf-plan — which itself ends by asking whether to go on into fdf-execute here
or in a fresh session; otherwise stop here and say that fdf-plan comes next.

## Rules

- Never skip the approval step; the spec records a human decision.
- No code, no scaffolding beyond `fdf new`, no implementation files until the
  design is approved — however simple the feature seems.
- One feature per brainstorm. If the idea spans independent subsystems,
  decompose it into features first, then brainstorm one.
- Write `slug.spec.md` and `slug.surface.md` beside the feature file — never
  `slug/SPEC.md` or any other trail document inside the task directory, which
  holds only tasks.
