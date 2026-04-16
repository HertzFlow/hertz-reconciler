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

// MarketParams snapshots safety-relevant DataStore parameters per market:
// minCollateralFactor, borrowingFactor (long/short), liquidationFeeFactor.
// Detection of changes is delegated to Prometheus via changes() rules.
type MarketParams struct {
	C *chain.Client
}

func (MarketParams) Name() string { return "market_params" }

func (m MarketParams) Run(_ context.Context, vc config.VersionConfig) (Report, error) {
	rep := Report{}
	if len(vc.Markets) == 0 {
		return rep, nil
	}
	ds := common.HexToAddress(vc.DataStore)

	for sym, mAddrHex := range vc.Markets {
		mAddr := common.HexToAddress(mAddrHex)

		mc, err := m.C.GetUint(ds, chain.MinCollateralFactorKey(mAddr))
		if err != nil {
			rep.Findings = append(rep.Findings, fmt.Sprintf("market %s minCollateralFactor: %v", sym, err))
		} else {
			metrics.MarketMinCollateralFactor.WithLabelValues(vc.Name, sym).Set(scaledFloat(mc))
		}

		for _, side := range []struct {
			name   string
			isLong bool
		}{{"long", true}, {"short", false}} {
			bf, err := m.C.GetUint(ds, chain.BorrowingFactorKey(mAddr, side.isLong))
			if err != nil {
				rep.Findings = append(rep.Findings, fmt.Sprintf("market %s/%s borrowingFactor: %v", sym, side.name, err))
				continue
			}
			metrics.MarketBorrowingFactor.WithLabelValues(vc.Name, sym, side.name).Set(scaledFloat(bf))
		}

		lf, err := m.C.GetUint(ds, chain.LiquidationFeeFactorKey(mAddr))
		if err != nil {
			rep.Findings = append(rep.Findings, fmt.Sprintf("market %s liquidationFeeFactor: %v", sym, err))
		} else {
			metrics.MarketLiquidationFee.WithLabelValues(vc.Name, sym).Set(scaledFloat(lf))
		}
	}
	rep.Lines = append(rep.Lines, fmt.Sprintf("snapshotted params for %d markets", len(vc.Markets)))
	metrics.ReconcilerLastRunTimestamp.WithLabelValues("market_params_" + vc.Name).SetToCurrentTime()
	return rep, nil
}

// scaledFloat divides a 1e30-scaled uint by 1e30 to a float64 the dashboards
// can read directly. Loses precision at extreme magnitudes but adequate for
// monitoring changes() detection.
func scaledFloat(v *big.Int) float64 {
	if v == nil {
		return 0
	}
	f := new(big.Float).SetInt(v)
	f.Quo(f, big.NewFloat(1e30))
	out, _ := f.Float64()
	return out
}
