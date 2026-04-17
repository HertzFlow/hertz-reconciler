package reconciler

import (
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/common"

	"github.com/HertzFlow/chain-reconciler/internal/chain"
	"github.com/HertzFlow/chain-reconciler/internal/config"
	"github.com/HertzFlow/chain-reconciler/internal/metrics"
)

// Queue mirrors spec §4 P0/P1: snapshot DataStore list counts. Originally
// these were probed via blackbox-exporter contract_call, but that exporter
// build cannot encode the bytes32 key argument and silently returns no data.
// Reading them via the reconciler's eth_call path is the safe fallback.
type Queue struct {
	C *chain.Client
}

func (Queue) Name() string { return "queue" }

func (q Queue) Run(_ context.Context, vc config.VersionConfig) (Report, error) {
	rep := Report{}
	ds := common.HexToAddress(vc.DataStore)

	type listSpec struct {
		label  string
		key    common.Hash
		isAddr bool // MARKET_LIST is an address set; the rest are bytes32 sets
	}
	lists := []listSpec{
		{"order", chain.OrderListRoot, false},
		{"deposit", chain.DepositListRoot, false},
		{"withdrawal", chain.WithdrawalListRoot, false},
		{"position", chain.PositionListRoot, false},
		{"market", chain.MarketListRoot, true},
	}

	for _, l := range lists {
		var (
			count uint64
			err   error
		)
		if l.isAddr {
			count, err = q.C.GetAddressCount(ds, l.key)
		} else {
			count, err = q.C.GetBytes32Count(ds, l.key)
		}
		if err != nil {
			rep.Findings = append(rep.Findings, fmt.Sprintf("queue %s count: %v", l.label, err))
			continue
		}
		metrics.QueueCount.WithLabelValues(vc.Name, l.label).Set(float64(count))
	}

	rep.Lines = append(rep.Lines, fmt.Sprintf("queue counts snapshotted (%d lists)", len(lists)))
	metrics.ReconcilerLastRunTimestamp.WithLabelValues("queue_" + vc.Name).SetToCurrentTime()
	return rep, nil
}
