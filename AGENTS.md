# Agent context

This repo follows Rubio-Enterprises standards. Run `/audit-standards` from a Claude Code session to check conformance, or `/onboard-repo` for greenfield setup.

Repo-specific context (in-progress migrations, gotchas, agent guidance):

## Standards deviations (§9.8)

Three pins diverge from the Rubio-Enterprises standards template
(`template/v1.2.5`) and must be preserved across `copier update`:

- **`go 1.25.0`** in `.tool-versions` (template pins `1.24.0`). The repo's
  `server/go.mod` declares `go 1.25.0`, so the consumer's pin wins per §9.8.
  Once the template's go-archetype pin bumps to >= 1.25, this deviation can be
  retired.
- **`aqua:golangci/golangci-lint = "2"`** in `.mise.toml`. The go-service
  template defines a `golangci-lint` pre-commit hook (`lefthook.yml`) and a
  `mise run lint` task, but never pins the binary itself. Without this pin
  `lint-hooks` CI fails with "couldn't exec process: No such file or
  directory". The analogous rust-archetype pre-commit pre-fetches the rustup
  toolchain; the go-archetype is missing the equivalent block. Retire once
  the template pins golangci-lint for go-* archetypes.
- **`aqua:gotestyourself/gotestsum = "1"`** in `.mise.toml`. The
  `[tasks.test]` block runs `gotestsum --junitfile=… -- ./...`. Same template
  bug as golangci-lint above. Retire once the template grows the pin.

## Preexisting `mise run lint` surface

`mise run lint` (full-repo `golangci-lint run`) currently reports ~91 issues
across `server/` (mostly `errcheck` for unchecked deferred-close errors and
`revive` for unused parameters in test helpers). These are NOT a CI gate —
no workflow invokes `mise run lint`, and the `lint-hooks` reusable workflow's
lefthook hook scopes via `--new-from-rev=HEAD` which only catches NEW issues.
Existing issues should be cleaned up incrementally; new code should run clean.

## Template-bug findings preserved for separate housekeeping

(Do not patch the template inside this migration PR — follow up separately.)

1. **`.golangci.yml` lists `gosimple`** which golangci-lint v2 rejects with
   "unknown linters". gosimple was folded into staticcheck. The template's
   `template/{...}.golangci.yml.jinja` needs the same fix applied here.
2. **`.mise.toml` go-archetype block** doesn't pin `golangci-lint` or
   `gotestsum`, but `lefthook.yml` calls the former and `[tasks.test]` calls
   the latter. Without pins, `lint-hooks` CI errors with
   "couldn't exec process: No such file or directory". The rust-archetype has
   an analogous `rustup toolchain install` pre-task; go-archetype is missing
   the equivalent surface.
3. **`lefthook.yml` `gofmt` / `golangci-lint` hooks** invoke their binaries
   bare (`golangci-lint run ...`), but git's commit-hook subshell loses
   mise's PATH activation — the hooks then fail with "command not found"
   under real `git commit`. The fix (applied here) is to prefix with
   `mise exec --`, matching how `markdownlint`, `yamllint`, etc. are
   invoked in the same file. Template's go-archetype block needs the same
   prefix.

## Fastlane / lefthook interaction

`ios/fastlane/Fastfile` sets `ENV["LEFTHOOK"] = "0"` so Fastlane's
auto-generated commits (which don't follow Conventional Commits) bypass
lefthook's `commit-msg` commitlint hook. Don't remove this line without first
auditing Fastlane's commit messages.
