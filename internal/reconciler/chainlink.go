package reconciler

import (
	"context"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/common"

	"github.com/HertzFlow/chain-reconciler/internal/chain"
	"github.com/HertzFlow/chain-reconciler/internal/config"
	"github.com/HertzFlow/chain-reconciler/internal/metrics"
)

// Chainlink probes a small set of canonical BSC-testnet Chainlink
// AggregatorV3 price feeds and records the staleness of latestRoundData().
// Spec §2 wants "Chainlink Sequencer/feed 节点状态" alerting; the protocol's
// own ChainlinkPriceFeedProvider wrapper does not expose latestRoundData,
// so we reach the underlying feeds directly.
//
// These feed addresses are publicly documented Chainlink BSC-testnet
// deployments — no Hertz-specific configuration. Treat them as health
// sentinels: if Chainlink upstream stops updating, all of these go stale
// together. One healthy feed implies the upstream is alive.
type Chainlink struct {
	C *chain.Client
}

var defaultBscTestnetFeeds = map[string]string{
	"BNB_USD": "0x2514895c72f50D8bd4B4F9b1110F0D6bD2c97526",
	"BTC_USD": "0x5741306c21795FdCBb9b265Ea0255F499DFe515C",
	"ETH_USD": "0x143db3CEEfbdfe5631aDD3E50f7614B6ba708BA7",
}

func (Chainlink) Name() string { return "chainlink" }

// Run does NOT use VersionConfig — Chainlink is shared across V1 and V2.
// Iterating once on v1 keeps the metric singular.
func (k Chainlink) Run(_ context.Context, vc config.VersionConfig) (Report, error) {
	rep := Report{}
	if vc.Name != "v1" {
		return rep, nil
	}

	now := uint64(time.Now().Unix())
	stale := 0
	for label, addrHex := range defaultBscTestnetFeeds {
		updatedAt, err := k.C.ChainlinkLatestUpdatedAt(common.HexToAddress(addrHex))
		if err != nil {
			rep.Findings = append(rep.Findings, fmt.Sprintf("chainlink %s: %v", label, err))
			continue
		}
		age := float64(0)
		if now > updatedAt {
			age = float64(now - updatedAt)
		}
		metrics.ChainlinkFeedAge.WithLabelValues(label).Set(age)
		if age > 600 { // 10 min sentinel for the daily digest
			stale++
		}
	}

	rep.Lines = append(rep.Lines, fmt.Sprintf("chainlink probed (%d feeds); stale>600s: %d", len(defaultBscTestnetFeeds), stale))
	metrics.ReconcilerLastRunTimestamp.WithLabelValues("chainlink").SetToCurrentTime()
	return rep, nil
}
