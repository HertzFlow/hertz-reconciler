# chain-reconciler

Daily on-chain reconciliation + governance snapshot for Hertz Protocol BSC
testnet (V1 + V2 contracts).

| Spec section | Covered |
| --- | --- |
| §3 Vault balance reconciliation | ✅ Native BNB + USDT per vault, per version |
| §6 RoleStore changes | ✅ `getRoleMembers` snapshot; change detection via `changes()` |
| §6 Timelock state | ✅ `getMinDelay()` per timelock |
| §7 Critical param changes | ✅ minCollateralFactor / borrowingFactor / liquidationFeeFactor for Tier 0+1+2 markets |
| §8 Fee accumulation vs distribution | ✅ per-market `claimableFeeAmount(USDT)` vs `FeeDistributorVault.balanceOf(USDT)` |
| §8 Long/short funding imbalance | ✅ `fundingFeeAmountPerSize` long vs short, ratio metric |
| §8 Keeper gas vs distribution | ✅ keeper balance + FeeDistributor USDT for delta() rules |
| §8 Daily Slack digest | ✅ webhook at configured UTC time |

## Architecture

Long-running Deployment (not CronJob) so VMServiceScrape can pull `/metrics`
like every other exporter. Two internal loops:

- **Quick loop** every `scheduler.quick_interval` (default 1h) — keeps
  metrics fresh; alerts fire from VictoriaMetrics rules.
- **Daily loop** at `scheduler.daily_at_utc` (default 00:00 UTC) — runs
  tasks then posts a digest to Slack.

`-once` flag: one-shot runs for cron / smoke tests.

## Layout

```
.
├── main.go                       # entry: scheduler, /metrics, /health
├── config.yaml                   # V1/V2 contracts + role list + keepers
├── Dockerfile                    # distroless, ~22MB binary
├── go.mod
└── internal/
    ├── config/                   # YAML + env loader
    ├── chain/                    # ETH RPC + DataStore key derivation
    ├── reconciler/               # one Task per concern
    │   ├── reconciler.go         # Task interface, RunAll, Render
    │   ├── vault.go              # §3
    │   ├── role.go               # §6 roles
    │   ├── timelock.go           # §6 timelock
    │   ├── params.go             # §7
    │   ├── fee.go                # §8 fee accumulation
    │   ├── funding.go            # §8 funding imbalance
    │   └── gas.go                # §8 keeper gas
    ├── metrics/                  # Prometheus gauges
    └── notify/                   # Slack webhook client
```

Deployment manifests + VMRule alerts live in `HertzFlow/cicd` under
`cicd/infra/base/chain-reconciler/` and
`cicd/infra/testnet-htzfl/monitoring/vmrule-reconciler-testnet.tpl.yaml`.

## Metrics

All gauges labelled `version=v1|v2`.

| Metric | Labels |
| --- | --- |
| `reconciler_vault_balance_native` | `vault` |
| `reconciler_vault_balance_usdt` | `vault` |
| `reconciler_role_members_count` | `role` |
| `reconciler_timelock_min_delay_seconds` | `timelock` |
| `reconciler_market_min_collateral_factor` | `market` |
| `reconciler_market_borrowing_factor` | `market`, `side` |
| `reconciler_market_liquidation_fee_factor` | `market` |
| `reconciler_market_claimable_fee_usdt` | `market` |
| `reconciler_total_claimable_fee_usdt` | — |
| `reconciler_fee_distribution_gap_usdt` | — |
| `reconciler_market_funding_fee_per_size` | `market`, `side` |
| `reconciler_market_funding_imbalance_ratio` | `market` |
| `reconciler_keeper_balance_native` | `keeper` |
| `reconciler_fee_distributor_balance_usdt` | — |
| `reconciler_last_run_timestamp` | `task` |
| `reconciler_errors_total` | `task`, `version` |
| `reconciler_up` | — |

## Build & Run

### Local dev

```bash
go build -o chain-reconciler .
./chain-reconciler -config config.yaml -once      # one-shot + print digest
./chain-reconciler -config config.yaml            # long-running + /metrics
```

### Container build

Pushes to `main` trigger `.github/workflows/build.yaml` → publishes
`ghcr.io/hertzflow/chain-reconciler:<short-sha>` and `:latest`.

Manual:

```bash
docker build -t ghcr.io/hertzflow/chain-reconciler:dev .
docker push ghcr.io/hertzflow/chain-reconciler:dev
```

### Kubernetes deploy

Manifests live in the HertzFlow/cicd repo, not here. Flux syncs them.

Ad-hoc:

```bash
kubectl apply -k <cicd-repo>/cicd/infra/base/chain-reconciler
```

### Slack digest secret

```bash
TOKEN="T08G35AED17/B0AEGQMFS3W/WD8xVubUJsBRX9xL7UMwZmVd"   # alert-test webhook
kubectl -n app create secret generic chain-reconciler-secret \
  --from-literal=slackWebhookUrl="https://hooks.slack.com/services/${TOKEN}"
```

When the secret is absent, the pod still publishes metrics; only the daily
Slack digest silently no-ops.

## Adding a reconciler task

1. Create `internal/reconciler/<task>.go` implementing the `Task`
   interface (`Name()`, `Run(ctx, version) (Report, error)`).
2. Register gauges in `internal/metrics/exporter.go`.
3. Append the task in `main.go`'s `tasks := []reconciler.Task{...}`.
4. Add VMRule entries in the cicd repo.
