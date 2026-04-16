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

// KeeperGas snapshots keeper wallet BNB balances + FeeDistributorVault USDT
// balance, giving VictoriaMetrics rules the raw signal to compute
//   delta(keeper_balance[24h])  vs  delta(fee_distributor_balance[24h])
// so an operator can tell whether the FeeDistributor is actually topping up
// keepers as gas is burned. Labelled with version for the vault side but
// keeper addresses are version-agnostic (shared).
//
// Note: we also export this per-keeper balance even though vmprobe
// pool-balance already does so — having a daily-aligned snapshot makes
// 24h-delta calculations cleaner vs 60s-scraped vmprobe series.
type KeeperGas struct {
	C       *chain.Client
	Keepers map[string]string
}

func (KeeperGas) Name() string { return "keeper_gas" }

func (k KeeperGas) Run(_ context.Context, vc config.VersionConfig) (Report, error) {
	rep := Report{}

	// Keeper balances are shared across versions; only iterate them once.
	// Pick v1 arbitrarily to avoid double-snapshot noise.
	if vc.Name == "v1" {
		totalWei := new(big.Int)
		for label, addrHex := range k.Keepers {
			bal, err := k.C.BalanceOfNative(common.HexToAddress(addrHex))
			if err != nil {
				rep.Findings = append(rep.Findings, fmt.Sprintf("keeper %s balance: %v", label, err))
				continue
			}
			metrics.KeeperBalanceNative.WithLabelValues(label).Set(weiToEther(bal))
			totalWei.Add(totalWei, bal)
		}
		rep.Lines = append(rep.Lines, fmt.Sprintf("total keeper BNB across %d wallets: %.4f", len(k.Keepers), weiToEther(totalWei)))
	}

	// Per-version FeeDistributor USDT (raw, not lossy). Mirrors vault task
	// but exported on the dedicated "gas accounting" series so dashboards
	// can pair delta(keeper_balance) with delta(fee_distributor_balance).
	if fdHex, ok := vc.Vaults["FeeDistributorVault"]; ok && vc.UsdtToken != "" {
		bal, err := k.C.BalanceOfERC20(common.HexToAddress(vc.UsdtToken), common.HexToAddress(fdHex))
		if err != nil {
			rep.Findings = append(rep.Findings, fmt.Sprintf("FeeDistributor USDT: %v", err))
		} else {
			metrics.FeeDistributorBalanceUSDT.WithLabelValues(vc.Name).Set(scaledFloat18(bal))
			rep.Lines = append(rep.Lines, fmt.Sprintf("FeeDistributorVault USDT: %.4f", scaledFloat18(bal)))
		}
	}

	metrics.ReconcilerLastRunTimestamp.WithLabelValues("keeper_gas_" + vc.Name).SetToCurrentTime()
	return rep, nil
}
