package reconciler

import (
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/common"

	"github.com/HertzFlow/chain-reconciler/internal/chain"
	"github.com/HertzFlow/chain-reconciler/internal/config"
	"github.com/HertzFlow/chain-reconciler/internal/metrics"
)

// OracleConfig snapshots Oracle-related DataStore parameters that the
// blackbox-exporter contract_call probe cannot encode (bytes32 key bug).
//
// Currently exposes:
//   - MAX_ORACLE_PRICE_AGE: the upper bound on Oracle.minTimestamp staleness
//     before order execution starts reverting. Spec §2 wants alert on change.
type OracleConfig struct {
	C *chain.Client
}

func (OracleConfig) Name() string { return "oracle_config" }

func (o OracleConfig) Run(_ context.Context, vc config.VersionConfig) (Report, error) {
	rep := Report{}
	ds := common.HexToAddress(vc.DataStore)

	maxAge, err := o.C.GetUint(ds, chain.MaxOraclePriceAgeRoot)
	if err != nil {
		rep.Findings = append(rep.Findings, fmt.Sprintf("MAX_ORACLE_PRICE_AGE: %v", err))
	} else {
		// Stored as plain uint seconds (no 1e30 scaling).
		secs := 0.0
		if maxAge != nil {
			f, _ := maxAge.Float64()
			secs = f
		}
		metrics.OracleMaxPriceAge.WithLabelValues(vc.Name).Set(secs)
		rep.Lines = append(rep.Lines, fmt.Sprintf("MAX_ORACLE_PRICE_AGE = %.0fs", secs))
	}

	metrics.ReconcilerLastRunTimestamp.WithLabelValues("oracle_config_" + vc.Name).SetToCurrentTime()
	return rep, nil
}
