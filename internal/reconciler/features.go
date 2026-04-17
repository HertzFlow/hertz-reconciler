package reconciler

import (
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/common"

	"github.com/HertzFlow/chain-reconciler/internal/chain"
	"github.com/HertzFlow/chain-reconciler/internal/config"
	"github.com/HertzFlow/chain-reconciler/internal/metrics"
)

// Features reads DataStore.getBool for the well-known protocol-pause feature
// keys (spec §7 P1: 系统功能开关). When CONFIG_KEEPER toggles any of these,
// the corresponding handler stops processing the named action — operators
// must know within seconds.
//
// Two key shapes are supported per feature:
//   1. Root-only:        keccak256(abi.encode(FEATURE_NAME))
//   2. Per-handler form: keccak256(abi.encode(FEATURE_NAME, handlerAddr))
//
// We probe (1) unconditionally, and (2) for every handler address listed in
// VersionConfig.Handlers (may be empty). A `false` read on either form is
// recorded as 0 in the gauge; a `true` read records 1, which the
// `FeatureDisabled` VMRule fires on immediately.
type Features struct {
	C *chain.Client
}

// Default Hertz/GMX feature names. Order matters only for the digest output.
var defaultFeatureNames = []string{
	"CREATE_DEPOSIT_FEATURE_DISABLED",
	"CANCEL_DEPOSIT_FEATURE_DISABLED",
	"EXECUTE_DEPOSIT_FEATURE_DISABLED",
	"CREATE_WITHDRAWAL_FEATURE_DISABLED",
	"CANCEL_WITHDRAWAL_FEATURE_DISABLED",
	"EXECUTE_WITHDRAWAL_FEATURE_DISABLED",
	"CREATE_ORDER_FEATURE_DISABLED",
	"UPDATE_ORDER_FEATURE_DISABLED",
	"CANCEL_ORDER_FEATURE_DISABLED",
	"EXECUTE_ORDER_FEATURE_DISABLED",
	"EXECUTE_ADL_FEATURE_DISABLED",
	"CLAIM_FUNDING_FEES_FEATURE_DISABLED",
	"CLAIM_COLLATERAL_FEATURE_DISABLED",
}

func (Features) Name() string { return "features" }

func (f Features) Run(_ context.Context, vc config.VersionConfig) (Report, error) {
	rep := Report{}
	ds := common.HexToAddress(vc.DataStore)

	disabled := 0
	for _, name := range defaultFeatureNames {
		// Root-only probe — works for protocols that store a global flag.
		root := chain.FeatureKeyRoot(name)
		v, err := f.C.GetBool(ds, root)
		if err != nil {
			rep.Findings = append(rep.Findings, fmt.Sprintf("feature %s root: %v", name, err))
		} else {
			val := 0.0
			if v {
				val = 1
				disabled++
				rep.Findings = append(rep.Findings, fmt.Sprintf("FEATURE DISABLED: %s (handler=root, version=%s)", name, vc.Name))
			}
			metrics.FeatureDisabled.WithLabelValues(vc.Name, name, "root").Set(val)
		}

		// Per-handler probes — only meaningful when handlers are configured.
		for handlerName, handlerHex := range vc.Handlers {
			handler := common.HexToAddress(handlerHex)
			key := chain.FeatureKeyForHandler(name, handler)
			v, err := f.C.GetBool(ds, key)
			if err != nil {
				rep.Findings = append(rep.Findings, fmt.Sprintf("feature %s/%s: %v", name, handlerName, err))
				continue
			}
			val := 0.0
			if v {
				val = 1
				disabled++
				rep.Findings = append(rep.Findings, fmt.Sprintf("FEATURE DISABLED: %s (handler=%s, version=%s)", name, handlerName, vc.Name))
			}
			metrics.FeatureDisabled.WithLabelValues(vc.Name, name, handlerName).Set(val)
		}
	}

	rep.Lines = append(rep.Lines, fmt.Sprintf("features probed (%d names × %d handlers + root); disabled=%d", len(defaultFeatureNames), len(vc.Handlers), disabled))
	metrics.ReconcilerLastRunTimestamp.WithLabelValues("features_" + vc.Name).SetToCurrentTime()
	return rep, nil
}
