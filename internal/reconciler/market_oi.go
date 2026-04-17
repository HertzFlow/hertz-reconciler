package reconciler

import (
	"context"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"

	"github.com/HertzFlow/chain-reconciler/internal/chain"
	"github.com/HertzFlow/chain-reconciler/internal/config"
	"github.com/HertzFlow/chain-reconciler/internal/metrics"
)

// MarketOI snapshots Open Interest (long/short), maxOpenInterest, pool amount
// (long-collateral side, USDT) and maxPoolAmount per market. Backstop for the
// blackbox-exporter market-oi VMProbe whose bytes32 keys never encode.
//
// We deliberately read pool amount in USDT (the long collateral on Hertz).
// Index-token pool amounts could be added later if it matters for the
// short side.
type MarketOI struct {
	C *chain.Client
}

func (MarketOI) Name() string { return "market_oi" }

func (m MarketOI) Run(_ context.Context, vc config.VersionConfig) (Report, error) {
	rep := Report{}
	if len(vc.Markets) == 0 {
		return rep, nil
	}
	ds := common.HexToAddress(vc.DataStore)
	usdt := common.HexToAddress(vc.UsdtToken)

	for sym, mAddrHex := range vc.Markets {
		mAddr := common.HexToAddress(mAddrHex)

		for _, side := range []struct {
			name   string
			isLong bool
		}{{"long", true}, {"short", false}} {
			oi, err := m.C.GetUint(ds, chain.OpenInterestKey(mAddr, usdt, side.isLong))
			if err == nil {
				metrics.MarketOI.WithLabelValues(vc.Name, sym, side.name).Set(weiTo30(oi))
			} else {
				rep.Findings = append(rep.Findings, fmt.Sprintf("market %s/%s OI: %v", sym, side.name, err))
			}

			maxOI, err := m.C.GetUint(ds, chain.MaxOpenInterestKey(mAddr, side.isLong))
			if err == nil {
				metrics.MarketMaxOI.WithLabelValues(vc.Name, sym, side.name).Set(weiTo30(maxOI))
			} else {
				rep.Findings = append(rep.Findings, fmt.Sprintf("market %s/%s maxOI: %v", sym, side.name, err))
			}
		}

		pool, err := m.C.GetUint(ds, chain.PoolAmountKey(mAddr, usdt))
		if err == nil {
			metrics.MarketPoolAmount.WithLabelValues(vc.Name, sym).Set(scaledFloat18(pool))
		} else {
			rep.Findings = append(rep.Findings, fmt.Sprintf("market %s pool: %v", sym, err))
		}

		maxPool, err := m.C.GetUint(ds, chain.MaxPoolAmountKey(mAddr, usdt))
		if err == nil {
			metrics.MarketMaxPoolAmount.WithLabelValues(vc.Name, sym).Set(scaledFloat18(maxPool))
		} else {
			rep.Findings = append(rep.Findings, fmt.Sprintf("market %s maxPool: %v", sym, err))
		}
	}

	rep.Lines = append(rep.Lines, fmt.Sprintf("market OI/pool snapshotted for %d markets", len(vc.Markets)))
	metrics.ReconcilerLastRunTimestamp.WithLabelValues("market_oi_" + vc.Name).SetToCurrentTime()
	return rep, nil
}

// weiTo30 converts a 1e30-scaled USD value to a float64 dashboards can show
// directly (USD units). Same scaling convention as scaledFloat in params.go.
func weiTo30(v *big.Int) float64 {
	if v == nil {
		return 0
	}
	f := new(big.Float).SetInt(v)
	f.Quo(f, big.NewFloat(1e30))
	out, _ := f.Float64()
	return out
}
