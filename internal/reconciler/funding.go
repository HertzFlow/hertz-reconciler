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

// Funding reconciles long vs short funding-fee-per-size per market (spec §8
// "多空方向应缴和应得的资金费是否平衡"). Large persistent |long - short|
// indicates directional imbalance the protocol is subsidising.
type Funding struct {
	C *chain.Client
}

func (Funding) Name() string { return "funding" }

func (f Funding) Run(_ context.Context, vc config.VersionConfig) (Report, error) {
	rep := Report{}
	if len(vc.Markets) == 0 {
		return rep, nil
	}
	ds := common.HexToAddress(vc.DataStore)
	usdt := common.HexToAddress(vc.UsdtToken)

	imbalances := 0
	for sym, mHex := range vc.Markets {
		market := common.HexToAddress(mHex)
		long, err := f.C.GetInt(ds, chain.FundingFeeAmountPerSizeKey(market, usdt, true))
		if err != nil {
			rep.Findings = append(rep.Findings, fmt.Sprintf("market %s funding long: %v", sym, err))
			continue
		}
		short, err := f.C.GetInt(ds, chain.FundingFeeAmountPerSizeKey(market, usdt, false))
		if err != nil {
			rep.Findings = append(rep.Findings, fmt.Sprintf("market %s funding short: %v", sym, err))
			continue
		}
		metrics.MarketFundingPerSize.WithLabelValues(vc.Name, sym, "long").Set(scaledSigned30(long))
		metrics.MarketFundingPerSize.WithLabelValues(vc.Name, sym, "short").Set(scaledSigned30(short))

		imb := absSub(long, short)
		maxSide := absMax(long, short)
		if maxSide.Sign() > 0 {
			ratio := new(big.Float).Quo(new(big.Float).SetInt(imb), new(big.Float).SetInt(maxSide))
			r, _ := ratio.Float64()
			metrics.MarketFundingImbalanceRatio.WithLabelValues(vc.Name, sym).Set(r)
			if r > 0.9 {
				imbalances++
			}
		}
	}
	rep.Lines = append(rep.Lines, fmt.Sprintf("funding snapshotted; markets with |long-short|/max > 0.9: %d", imbalances))
	metrics.ReconcilerLastRunTimestamp.WithLabelValues("funding_" + vc.Name).SetToCurrentTime()
	return rep, nil
}

func scaledSigned30(v *big.Int) float64 {
	if v == nil {
		return 0
	}
	f := new(big.Float).SetInt(v)
	f.Quo(f, big.NewFloat(1e30))
	out, _ := f.Float64()
	return out
}

func absSub(a, b *big.Int) *big.Int {
	d := new(big.Int).Sub(a, b)
	return d.Abs(d)
}

func absMax(a, b *big.Int) *big.Int {
	aa := new(big.Int).Abs(a)
	bb := new(big.Int).Abs(b)
	if aa.Cmp(bb) >= 0 {
		return aa
	}
	return bb
}
