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

// Fees reconciles per-market claimableFeeAmount(USDT) against the
// FeeDistributorVault's actual USDT balance. Spec §8: "累计收取手续费 vs
// 已分发到 FeeDistributorVault 的金额". We expose both sides as metrics and
// compute the outstanding gap so VictoriaMetrics rules can alert on it.
type Fees struct {
	C *chain.Client
}

func (Fees) Name() string { return "fees" }

func (f Fees) Run(_ context.Context, vc config.VersionConfig) (Report, error) {
	rep := Report{}
	if len(vc.Markets) == 0 {
		return rep, nil
	}
	ds := common.HexToAddress(vc.DataStore)
	usdt := common.HexToAddress(vc.UsdtToken)

	fdVaultAddr, hasFD := vc.Vaults["FeeDistributorVault"]
	totalClaimable := new(big.Int)

	for sym, mHex := range vc.Markets {
		m := common.HexToAddress(mHex)
		claim, err := f.C.GetUint(ds, chain.ClaimableFeeAmountKey(m, usdt))
		if err != nil {
			rep.Findings = append(rep.Findings, fmt.Sprintf("market %s claimable fee: %v", sym, err))
			continue
		}
		metrics.MarketClaimableFee.WithLabelValues(vc.Name, sym).Set(scaledFloat18(claim))
		totalClaimable.Add(totalClaimable, claim)
	}
	metrics.TotalClaimableFeeUSDT.WithLabelValues(vc.Name).Set(scaledFloat18(totalClaimable))
	rep.Lines = append(rep.Lines, fmt.Sprintf("total claimable fee (USDT): %.4f across %d markets",
		scaledFloat18(totalClaimable), len(vc.Markets)))

	if !hasFD {
		return rep, nil
	}
	fdBal, err := f.C.BalanceOfERC20(usdt, common.HexToAddress(fdVaultAddr))
	if err != nil {
		rep.Findings = append(rep.Findings, fmt.Sprintf("FeeDistributorVault USDT balance: %v", err))
		return rep, nil
	}
	// undistributed = claimable - already in FeeDistributorVault
	gap := new(big.Int).Sub(totalClaimable, fdBal)
	metrics.FeeDistributionGapUSDT.WithLabelValues(vc.Name).Set(scaledFloat18(gap))
	rep.Lines = append(rep.Lines, fmt.Sprintf("undistributed fee gap: %.4f USDT", scaledFloat18(gap)))
	// Threshold: flag when gap grows beyond 1000 USDT (configurable via alert rule)
	if gap.Cmp(new(big.Int).Mul(big.NewInt(1000), big.NewInt(1e18))) > 0 {
		rep.Findings = append(rep.Findings,
			fmt.Sprintf("fee distribution gap %.2f USDT — FEE_KEEPER may not be draining claimables",
				scaledFloat18(gap)))
	}

	metrics.ReconcilerLastRunTimestamp.WithLabelValues("fees_" + vc.Name).SetToCurrentTime()
	return rep, nil
}

// scaledFloat18 divides a wei-denominated value by 1e18 to get a float
// in USDT/BNB units. Lossy for huge magnitudes; fine for alerting.
func scaledFloat18(v *big.Int) float64 {
	if v == nil {
		return 0
	}
	f := new(big.Float).SetInt(v)
	f.Quo(f, big.NewFloat(1e18))
	out, _ := f.Float64()
	return out
}
