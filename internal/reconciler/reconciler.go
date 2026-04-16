package reconciler

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/HertzFlow/chain-reconciler/internal/chain"
	"github.com/HertzFlow/chain-reconciler/internal/config"
	"github.com/HertzFlow/chain-reconciler/internal/metrics"
)

// Task is a self-contained on-chain reconciliation routine. Each task knows
// how to gather a single concern (vault balances, role members, timelock
// state, market params) and report results both as Prometheus metrics and a
// human-readable summary for the daily Slack digest.
type Task interface {
	Name() string
	Run(ctx context.Context, version config.VersionConfig) (Report, error)
}

// Report is what a Task hands back per version. Lines are rendered into the
// Slack daily digest; metrics are already published by the time Run returns.
type Report struct {
	Version string
	Task    string
	Lines   []string
	// Findings are flagged anomalies the operator should look at. Empty slice
	// means "all clean for this task on this version".
	Findings []string
}

// Render produces a Slack-friendly digest, grouped by version then by task,
// with all findings collected at the end.
func Render(reports []Report) string {
	byVersion := make(map[string][]Report)
	var versionOrder []string
	for _, r := range reports {
		if _, ok := byVersion[r.Version]; !ok {
			versionOrder = append(versionOrder, r.Version)
		}
		byVersion[r.Version] = append(byVersion[r.Version], r)
	}

	var b strings.Builder
	b.WriteString("*Chain Reconciler — Daily Report*\n")

	var allFindings []string
	for _, ver := range versionOrder {
		b.WriteString(fmt.Sprintf("\n*=== %s ===*\n", ver))
		for _, r := range byVersion[ver] {
			b.WriteString(fmt.Sprintf("_%s_:\n", r.Task))
			for _, l := range r.Lines {
				b.WriteString("  • " + l + "\n")
			}
			for _, f := range r.Findings {
				allFindings = append(allFindings, fmt.Sprintf("[%s/%s] %s", ver, r.Task, f))
			}
		}
	}
	if len(allFindings) > 0 {
		b.WriteString("\n*⚠️ Findings*\n")
		for _, f := range allFindings {
			b.WriteString("  - " + f + "\n")
		}
	}
	return b.String()
}

// RunAll executes every task across every configured version, returning the
// concatenated reports. Errors from individual tasks are logged and surfaced
// via metrics.ReconcilerErrorsTotal but do not abort the overall run.
func RunAll(ctx context.Context, log *slog.Logger, client *chain.Client, cfg *config.Config, tasks []Task) []Report {
	var reports []Report
	for _, v := range cfg.Versions {
		for _, t := range tasks {
			rep, err := t.Run(ctx, v)
			if err != nil {
				log.Error("task failed", "task", t.Name(), "version", v.Name, "err", err)
				metrics.ReconcilerErrorsTotal.WithLabelValues(t.Name(), v.Name).Inc()
				rep.Findings = append(rep.Findings, fmt.Sprintf("[%s] error: %v", t.Name(), err))
			}
			rep.Version = v.Name
			rep.Task = t.Name()
			reports = append(reports, rep)
		}
	}
	return reports
}
