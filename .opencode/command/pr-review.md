---
description: Run architecture + code + security review on the current PR, post as 3 separate comments, then propose remediation
agent: OpenAgent
---

# PR Review Command

Run a full three-pass review on the current pull request and post the results.
Optional `$ARGUMENTS` may name a PR number; if omitted, use the PR for the current branch.

## Workflow (execute in order)

1. **Identify the PR**
   - If `$ARGUMENTS` contains a PR number, use it. Otherwise resolve the current branch's PR:
     `gh pr view --json number,title,headRefName,baseRefName,url,body`
   - Get the changed files and full diff: `gh pr diff <n> --name-only` then read the actual changed files (not just the diff) for full context.
   - Read the linked spec/design docs (e.g. `docs/spec/*.md`) and `docs/architecture/decisions-log.md` so review is grounded in approved decisions.
   - Build + test the affected packages to ground findings in reality (`go build ./...`, `go test ./<changed-pkgs>/...`). Note pass/fail in the reviews.

2. **Write THREE separate PR comments** (one `gh pr comment <n> --body-file` per review). Each comment is self-contained with a verdict and a findings table.

   ### a) 🏛️ Architecture Review
   - Hexagonal compliance: domain core must NOT import adapter/SDK packages; SQL/IO confined to adapters.
   - Port/interface design: minimal, intention-revealing, compile-time `var _ Iface = (*Impl)(nil)` checks present.
   - DDD: bounded-context boundaries, identity/provenance consistency across stories.
   - Deployment topology vs. platform constraints (metastore-locality, multi-account).
   - Config sourcing single-rooted; docs match implementation.
   - Cross-reference `docs/architecture/decisions-log.md` — flag any drift between a recorded decision and the code.

   ### b) 🔍 Code Review
   - Correctness bugs (logic, off-by-one, duplicated/dead code, error swallowing).
   - Robustness: error handling, partial-failure/exit semantics, parsing brittleness, retries/idempotency.
   - Go idioms: error wrapping with `%w`, lowercase error strings, naming, `samber/lo`/`samber/mo` functional style where it fits.
   - Test coverage: enumerate what's covered and what's missing; confirm tests pass.
   - Style nits grouped separately from required fixes.

   ### c) 🔒 Security Review (OWASP-aware)
   - **SQL injection**: all runtime values bound via parameters (`:name` for Databricks, `$1` for Lakebase) — never interpolated. Non-bindable identifiers (catalog/schema/table) must be allow-listed (`shared.ValidateIdent`).
   - **Secrets**: no hardcoded secrets; tokens never logged; prefer OAuth/Service Principal over long-lived PATs.
   - **Anonymization policy**: data shared with Pulsobot must be anonymized/aggregated — check raw-boundary redaction of secret-bearing keys (`spark_env_vars`, `*_secret`, `token`, `base_parameters`, init scripts, cloud attributes) per the decisions log.
   - **Input validation**: all external input validated; sanitize before render/persist.
   - **Artifact integrity**: downloaded binaries/artifacts checksum-verified; restricted temp paths and ACLs.
   - Each finding rated (Critical / High / Medium / Low / Informational) with file:line and a concrete fix.

   Use severity ratings consistently. Ground every finding in a `file:line` reference. Prefer ✅/⚠️/❌ in checklist tables.

3. **Propose remediation (requires human approval)**
   - Synthesize all findings across the three reviews into a single **Remediation Plan**: ordered, each item tagged with its review (Arch/Code/Sec), severity, the file to touch, and whether it's "fix now" or "defer (with target story)".
   - Post the remediation **summary as a PR comment** (`## 🛠️ Proposed Remediation`).
   - Present the same plan in chat and **ask the human to approve** before making any code changes. Do NOT edit code until approved.
   - After approval, apply fixes (TDD where it makes sense), run lint + tests, then post a **Remediation Report** comment (what fixed / what deferred / updated test results) per the story lifecycle.

## Notes
- Use `gh` for all GitHub interaction. Never push directly to `main`.
- Keep each review comment focused; do not merge the three into one.
- Write comment bodies to temp files (`/tmp/*.md`) and post with `--body-file` to preserve markdown formatting.
- This mirrors the Review + Fix steps in `docs/workflow/story-lifecycle.md`.
