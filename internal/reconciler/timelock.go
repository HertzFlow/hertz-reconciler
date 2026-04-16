package reconciler

import (
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/common"

	"github.com/HertzFlow/chain-reconciler/internal/chain"
	"github.com/HertzFlow/chain-reconciler/internal/config"
	"github.com/HertzFlow/chain-reconciler/internal/metrics"
)

var _ = chain.OpenInterestRoot // shared chain package used by sibling tasks

// Timelock snapshots Gov/Config timelock minDelay. Sudden delay reduction is
// a safety-relevant signal; downstream alerts use changes() / delta().
type Timelock struct {
	C *chain.Client
}

func (Timelock) Name() string { return "timelock" }

func (t Timelock) Run(_ context.Context, vc config.VersionConfig) (Report, error) {
	rep := Report{}
	for label, addrHex := range map[string]string{
		"gov":    vc.GovTimelock,
		"config": vc.ConfigTimelock,
	} {
		if addrHex == "" {
			continue
		}
		addr := common.HexToAddress(addrHex)
		delay, err := t.C.GetMinDelay(addr)
		if err != nil {
			rep.Findings = append(rep.Findings, fmt.Sprintf("%s timelock minDelay: %v", label, err))
			continue
		}
		metrics.TimelockMinDelay.WithLabelValues(vc.Name, label).Set(float64(delay))
		rep.Lines = append(rep.Lines, fmt.Sprintf("%s timelock minDelay: %ds", label, delay))
		if delay < 60 {
			rep.Findings = append(rep.Findings, fmt.Sprintf("%s timelock minDelay is only %ds — verify", label, delay))
		}
	}
	metrics.ReconcilerLastRunTimestamp.WithLabelValues("timelock_" + vc.Name).SetToCurrentTime()
	return rep, nil
}
