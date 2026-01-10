# Review Philosophy

- Comment only when there is **high confidence (>80%)** that an issue, risk, or meaningful improvement exists. Avoid speculative or low-impact feedback.
- Prioritize **signal over noise**. If a comment does not clearly improve correctness, readability, performance, security, or maintainability, do not leave it.
- Be **concise and direct**. Prefer a single, well-phrased sentence per comment whenever possible.
- Focus on **actionable feedback**. Each comment should either explain *what* is wrong, *why* it matters, or *how* to improve it.
- Avoid restating what the code already does. Assume the author can read the code.
- When reviewing text or documentation:
  - Comment only if the wording is **genuinely ambiguous, misleading, or likely to cause incorrect usage**.
  - Do not suggest stylistic or subjective wording changes unless they materially improve clarity or prevent misunderstanding.
- Treat every review as if the code will be **maintained by someone else six months from now**.

## Priority Areas (Review These First)

Focus review effort on the areas below, in order of **risk and long-term impact**.
Deprioritize minor style or preference-based issues unless they materially affect maintainability.

---

### Security & Safety

- Unsafe code blocks **without clear justification, scope, or documented invariants**.
- Command injection risks involving shell execution, dynamic commands, or unsanitized user input.
- Path traversal vulnerabilities when handling file paths, URLs, or external input.
- Credential exposure, hardcoded secrets, tokens, API keys, or sensitive configuration values.
- Missing or insufficient input validation on **external or untrusted data sources**.
- Improper error handling that could **leak sensitive information** through logs, error messages, or responses.
- Security-sensitive behavior that is implicit, undocumented, or relies on assumptions not enforced in code.

---

### Correctness Issues

- Logic errors that could lead to panics, crashes, undefined behavior, or incorrect results.
- Race conditions, shared-state issues, or unsafe access patterns in concurrent or async code.
- Resource leaks involving files, network connections, locks, or memory.
- Boundary issues such as off-by-one errors, empty states, or unhandled edge cases.
- Incorrect error propagation
- Optional types used where a value is guaranteed or required, adding unnecessary complexity.
- Error context that does not meaningfully improve debuggability or understanding.
- Overly defensive code that adds checks without realistic failure modes.
- Comments that restate obvious behavior instead of explaining **why** something exists.

---

### Architecture & Patterns

- Code that violates established patterns, conventions, or architectural decisions in the codebase.
- Missing or inconsistent error handling where a standard approach is already used
- Misuse of async/await, including blocking operations inside async contexts.
- Improper or incomplete trait implementations that break expectations or contracts.
- Abstractions that increase complexity without reducing duplication or improving clarity.
- Public APIs that expose unnecessary surface area or leak internal implementation details.

## Skip These (Low Value)

Do **not** leave review comments for the following, unless they directly impact
correctness, security, or long-term maintainability:

- Style or formatting concerns handled by automated tools (`go fmt`, Prettier).
- Minor naming preferences that do not materially improve clarity or correctness.
- Suggestions to add comments when the code is already self-explanatory.
- Refactoring proposals unless they fix a real bug, remove duplicated logic, or significantly reduce complexity.
- Logging suggestions unless they are required for **security, auditing, or critical observability gaps**.
- Pedantic wording or text accuracy nitpicks unless misunderstanding could lead to incorrect usage or bugs.

When in doubt, **err on the side of silence**.

## Response Format

Use the following structure for every review comment.
Do not deviate unless brevity clearly improves clarity.

1. **State the problem**
   - One clear sentence describing the concrete issue.
   - Avoid speculation or vague phrasing.

2. **Why it matters** (optional)
   - One sentence explaining impact (correctness, safety, maintainability, or developer experience).
   - Omit this step if the impact is obvious.

3. **Suggested fix**
   - Provide a specific action, code snippet, or alternative approach.
   - Prefer minimal, localized changes over broad refactors.

## When to Stay Silent

- If you are **not confident** that something is an actual issue, do not comment.
- Do not speculate or ask hypothetical questions disguised as feedback.
- Silence is preferred over low-confidence, low-impact, or opinion-based comments.
- If an issue depends on missing context and cannot be verified from the diff, assume the author has context and stay silent.
- Only break silence when uncertainty itself creates a **real risk** (e.g., potential security, data loss, or correctness issues).

Default to restraint. A good review is measured by **impact**, not comment count.
