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

// VaultBalances reads native BNB and USDT balances for every vault and emits
// metrics. Findings: any vault holding noteworthy USDT (>1) is listed in the
// daily report so operators can spot lingering balances that should have been
// drained.
type VaultBalances struct {
	C *chain.Client
}

func (VaultBalances) Name() string { return "vault_balances" }

func (v VaultBalances) Run(_ context.Context, vc config.VersionConfig) (Report, error) {
	rep := Report{}
	if len(vc.Vaults) == 0 {
		return rep, nil
	}
	usdt := common.HexToAddress(vc.UsdtToken)

	for name, addrHex := range vc.Vaults {
		addr := common.HexToAddress(addrHex)

		// native
		bnb, err := v.C.BalanceOfNative(addr)
		if err != nil {
			rep.Findings = append(rep.Findings, fmt.Sprintf("%s native balance: %v", name, err))
			continue
		}
		bnbEth := weiToEther(bnb)
		metrics.VaultBalanceNative.WithLabelValues(vc.Name, name).Set(bnbEth)

		// USDT (BSC USDT has 18 decimals)
		usdtBal, err := v.C.BalanceOfERC20(usdt, addr)
		if err != nil {
			rep.Findings = append(rep.Findings, fmt.Sprintf("%s USDT balance: %v", name, err))
			continue
		}
		usdtUnits := weiToEther(usdtBal)
		metrics.VaultBalanceUSDT.WithLabelValues(vc.Name, name).Set(usdtUnits)

		rep.Lines = append(rep.Lines, fmt.Sprintf("%s: %.4f BNB / %.2f USDT", name, bnbEth, usdtUnits))
		if usdtUnits > 1.0 && (name == "OrderVault" || name == "DepositVault") {
			rep.Findings = append(rep.Findings, fmt.Sprintf("%s holds %.2f USDT — possible stuck deposit/order", name, usdtUnits))
		}
	}
	metrics.ReconcilerLastRunTimestamp.WithLabelValues("vault_balances_" + vc.Name).SetToCurrentTime()
	return rep, nil
}

func weiToEther(wei *big.Int) float64 {
	if wei == nil {
		return 0
	}
	f := new(big.Float).SetInt(wei)
	f.Quo(f, big.NewFloat(1e18))
	v, _ := f.Float64()
	return v
}
