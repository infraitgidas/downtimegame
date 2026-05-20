# Skill Registry — downtime-game

**Delegator use only.** Any agent that launches sub-agents reads this registry to resolve compact rules, then injects them directly into sub-agent prompts. Sub-agents do NOT read this registry or individual SKILL.md files.

## User Skills

| Trigger | Skill | Path |
|---------|-------|------|
| When creating a pull request, opening a PR, or preparing changes for review | branch-pr | ~/.config/opencode/skills/branch-pr/SKILL.md |
| When writing Go tests, using teatest, or adding test coverage | go-testing | ~/.config/opencode/skills/go-testing/SKILL.md |
| When creating a GitHub issue, reporting a bug, or requesting a feature | issue-creation | ~/.config/opencode/skills/issue-creation/SKILL.md |
| When user says "judgment day", "judgment-day", "review adversarial", "dual review", "doble review", "juzgar", "que lo juzguen" | judgment-day | ~/.config/opencode/skills/judgment-day/SKILL.md |
| When user asks to create a new skill, add agent instructions, or document patterns for AI | skill-creator | ~/.config/opencode/skills/skill-creator/SKILL.md |

## Compact Rules

Pre-digested rules per skill. Delegators copy matching blocks into sub-agent prompts as `## Project Standards (auto-resolved)`.

### branch-pr
- Every PR MUST link an approved issue (Closes/Fixes/Resolves #N) — no exceptions
- Branch naming: `^(feat|fix|chore|docs|style|refactor|perf|test|build|ci|revert)\/[a-z0-9._-]+$`
- Conventional commits: `type(scope): description` — types: build, chore, ci, docs, feat, fix, perf, refactor, revert, style, test
- PR body MUST include: linked issue, exactly one type:* label, summary, changes table, test plan, contributor checklist
- Automated checks must pass before merge
- No `Co-Authored-By` or AI attribution trailers
- Run `shellcheck` on modified scripts before pushing

### go-testing
- Use table-driven tests for multiple test cases
- Test Bubbletea models directly via `Model.Update()` with `tea.KeyMsg`
- Use `teatest.NewTestModel()` for full TUI integration tests
- Golden file testing: compare vs `testdata/*.golden` with `-update` flag
- System exec: mock via interfaces; files → `t.TempDir()`
- Error cases MUST be tested alongside success cases

### issue-creation
- Blank issues disabled — MUST use template (bug_report.yml or feature_request.yml)
- Every issue gets `status:needs-review` automatically
- A maintainer MUST add `status:approved` before any PR
- Questions go to Discussions, NOT issues
- Search existing issues for duplicates before creating

### judgment-day
- Launch TWO blind judge sub-agents via `delegate` (async, parallel) — NEVER sequential
- Inject `## Project Standards (auto-resolved)` into BOTH judge prompts
- Synthesize verdict: Confirmed, Suspect A/B, Contradiction
- Classify warnings: REAL vs THEORETICAL
- Max 2 fix+re-judge iterations before asking user
- APPROVED: 0 confirmed CRITICALs + 0 confirmed real WARNINGs

### skill-creator
- Frontmatter MUST include: name, description (with Trigger:), license (Apache-2.0), metadata.author, metadata.version
- Structure: `skills/{name}/SKILL.md` (required), `assets/` and `references/` (optional)
- DO: critical patterns first, tables, minimal code examples, Commands section
- DON'T: Keywords section, duplicated content, lengthy explanations, web URLs in references
- Register new skill in AGENTS.md after creation

## Project Conventions

No project-level convention files found in downtime-game.
