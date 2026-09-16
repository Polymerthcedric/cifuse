# cifuse

`cifuse` is a local-first guardrail for GitHub Actions. It catches workflow choices that can waste CI minutes or leave the default GitHub token broader than intended.

## MVP

```sh
go install github.com/Polymerthcedric/cifuse/cmd/cifuse@v0.1.0

cifuse audit .
cifuse audit --format json .
```

The first release checks for:

- workflow-level concurrency with cancellation;
- explicit default `GITHUB_TOKEN` permissions;
- a `timeout-minutes` cap on every job; and
- `permissions: write-all`.

It exits 1 when a policy gap is found, so it is ready to place in a GitHub Actions workflow.

## Why this exists

GitHub Actions permits concurrent runs by default. Concurrency limits can cancel stale runs, and `timeout-minutes` can cap stalled steps; GitHub also recommends setting the default `GITHUB_TOKEN` to read-only and elevating only the jobs that need it. [GitHub Actions documentation](https://docs.github.com/en/actions/reference/workflows-and-actions/workflow-syntax) documents concurrency and timeouts, while its [secure-use guidance](https://docs.github.com/en/actions/reference/security/secure-use) documents least-privilege tokens.

`cifuse` deliberately does not claim to enforce a billing cap: GitHub billing is account-level data, not something a workflow-file linter can honestly control.

## 14-day delivery scope

1. Local audit and JSON output.
2. Configurable thresholds and rule exclusions.
3. `cifuse init` to add a self-checking workflow.
4. SARIF output and a GitHub Action wrapper.
5. Release binaries and a public launch post.

Future paid value, if users want it, is team policy sync and historical CI-cost reporting—not required for the useful open-source core.
