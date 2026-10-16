# cifuse

Guard your GitHub Actions before they cost you CI minutes.

`cifuse` checks the workflows in `.github/workflows` and reports a few common gaps that can waste CI time or grant GitHub Actions more access than a job needs:

- **concurrency** that lets duplicate runs pile up on every push;
- **no declared default permissions**, so jobs run with an over-privileged token;
- **jobs without `timeout-minutes`**, so one stalled step can hang the queue; and
- **`permissions: write-all`**, which grants more access than most workflows need.

It is a local-first, single-binary linter. No daemon, no SaaS, no telemetry.

## Install and run

```sh
curl -fsSL https://raw.githubusercontent.com/Polymerthcedric/cifuse/main/scripts/install.sh | bash
cifuse audit .
```

```text
$ cifuse audit .
FAIL  1 finding(s) across 1 workflow(s)
CF001  .github/workflows/ci.yml:1  add workflow concurrency with cancel-in-progress to stop duplicate runs consuming CI minutes
```

Exit code is `1` when a guardrail gap is found, `0` when clean — so it slots directly into CI. Every finding names the rule (CF001–CF005), the file, and the line.

`go install` users can get the same binary with:

```sh
go install github.com/Polymerthcedric/cifuse/cmd/cifuse@v0.2.0
```

## Rules

| Rule | Guardrail | Why it matters |
| --- | --- | --- |
| CF001 | workflow-level `concurrency` with `cancel-in-progress: true` | stale runs on the same ref are cancelled instead of billing minutes twice |
| CF002 | declared default `permissions: contents: read` (elevate per job) | least-privilege token; GitHub recommends read-only by default |
| CF003 | `timeout-minutes` on every job | a hung step fails fast instead of stalling the queue |
| CF004 | no `permissions: write-all` | a workflow token should only reach what its job touches |
| CF005 | at least one job under `jobs:` | a silently empty workflow does nothing at all |

`cifuse` deliberately does **not** claim to enforce a billing cap: GitHub usage is account-level data, and an honest linter cannot pretend to control it. It checks what the workflow file can prove — nothing more.

## Run it every PR

`cifuse init` adds a workflow that checks your workflows on pushes and pull requests. It uploads SARIF results to GitHub code scanning:

```sh
cifuse init
```

It writes `.github/workflows/cifuse.yml` and a `.cifuserc.yml` for your exclusions. The generated workflow pins its actions to immutable commit SHAs — the same discipline cifuse audits — and runs `cifuse --format sarif`, so findings surface in the **Security → Code scanning** tab.

## Formats

- `cifuse audit` — human-readable text
- `cifuse audit --format json` — machine-readable report
- `cifuse audit --format sarif` — SARIF 2.1.0 for code scanning upload

## Exclusions

Exclude rules, whole workflows, or specific findings without loosening your workflow files:

```sh
cifuse audit --exclude CF003 .
cifuse audit --exclude-path .github/workflows/legacy.yml .
```

Persist the same policy in `.cifuserc.yml` (auto-discovered, or point `--config` at another path):

```yaml
ignore:
  rules:
    - CF001
  paths:
    - .github/workflows/legacy.yml
  findings:
    - path: .github/workflows/archive.yml
      rule: CF003
```

## Install options

| Method | Command |
| --- | --- |
| Install script | `curl -fsSL https://raw.githubusercontent.com/Polymerthcedric/cifuse/main/scripts/install.sh \| bash` |
| Install script, pinned | `curl -fsSL ... \| bash -s -- -v v0.2.0` |
| Go module | `go install github.com/Polymerthcedric/cifuse/cmd/cifuse@v0.2.0` |
| GitHub Releases | linux / darwin / windows × amd64 / arm64 tarballs with `checksums.txt` |

Binaries are static (`CGO_ENABLED=0`), so they run anywhere without dependencies.

## Need an application starter too?

`cifuse` checks GitHub Actions workflow files. It does not configure application security. The separate [DevSecOps Starter](https://starter.fidelco.dev/buy) includes a Next.js example with rate limiting, sessions, and a deployment workflow.

## License

MIT — see [LICENSE](LICENSE).
