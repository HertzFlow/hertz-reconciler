package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// All reconciler gauges are labelled with `version` so V1/V2 series stay
// separate and dashboards can filter cleanly.

var (
	VaultBalanceNative = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "reconciler_vault_balance_native",
		Help: "Vault native (BNB) balance in ether",
	}, []string{"version", "vault"})

	VaultBalanceUSDT = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "reconciler_vault_balance_usdt",
		Help: "Vault USDT balance in token units",
	}, []string{"version", "vault"})

	RoleMembersCount = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "reconciler_role_members_count",
		Help: "RoleStore member count; downstream uses changes() to detect grants/revokes",
	}, []string{"version", "role"})

	TimelockMinDelay = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "reconciler_timelock_min_delay_seconds",
		Help: "Timelock.getMinDelay() in seconds",
	}, []string{"version", "timelock"})

	MarketMinCollateralFactor = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "reconciler_market_min_collateral_factor",
		Help: "DataStore.getUint(minCollateralFactorKey) / 1e30",
	}, []string{"version", "market"})

	MarketBorrowingFactor = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "reconciler_market_borrowing_factor",
		Help: "DataStore.getUint(borrowingFactorKey) / 1e30",
	}, []string{"version", "market", "side"})

	MarketLiquidationFee = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "reconciler_market_liquidation_fee_factor",
		Help: "DataStore.getUint(liquidationFeeFactorKey) / 1e30",
	}, []string{"version", "market"})

	MarketClaimableFee = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "reconciler_market_claimable_fee_usdt",
		Help: "DataStore.getUint(claimableFeeAmountKey(market, USDT)) / 1e18",
	}, []string{"version", "market"})

	TotalClaimableFeeUSDT = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "reconciler_total_claimable_fee_usdt",
		Help: "Sum of claimable USDT fees across all monitored markets",
	}, []string{"version"})

	FeeDistributionGapUSDT = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "reconciler_fee_distribution_gap_usdt",
		Help: "totalClaimableFee - FeeDistributorVault.balanceOf(USDT); positive = undistributed",
	}, []string{"version"})

	MarketFundingPerSize = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "reconciler_market_funding_fee_per_size",
		Help: "Signed fundingFeeAmountPerSize / 1e30 per market-side",
	}, []string{"version", "market", "side"})

	MarketFundingImbalanceRatio = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "reconciler_market_funding_imbalance_ratio",
		Help: "|long - short| / max(|long|, |short|); 1.0 = fully one-sided",
	}, []string{"version", "market"})

	KeeperBalanceNative = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "reconciler_keeper_balance_native",
		Help: "Keeper wallet BNB balance in ether (reconciler-side snapshot)",
	}, []string{"keeper"})

	FeeDistributorBalanceUSDT = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "reconciler_fee_distributor_balance_usdt",
		Help: "FeeDistributorVault USDT balance (complement to vault_balance_usdt)",
	}, []string{"version"})

	ReconcilerLastRunTimestamp = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "reconciler_last_run_timestamp",
		Help: "Unix timestamp of the last successful run for a reconciler task",
	}, []string{"task"})

	ReconcilerErrorsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "reconciler_errors_total",
		Help: "Total reconciler errors by task and version",
	}, []string{"task", "version"})

	ReconcilerUp = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "reconciler_up",
		Help: "1 when reconciler process is healthy",
	})

	// FeatureDisabled exposes the result of DataStore.getBool for known
	// feature keys (spec §7 P1). 1 = paused, 0 = active. Labelled by feature
	// constant name and by handler — "root" is the globally-scoped probe,
	// other handler labels come from VersionConfig.Handlers.
	FeatureDisabled = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "reconciler_feature_disabled",
		Help: "1 if the named protocol feature is paused on the given handler",
	}, []string{"version", "feature", "handler"})

	// QueueCount mirrors what the datastore-queue VMProbe was supposed to
	// expose but cannot (blackbox-exporter contract_call has no bytes32 arg
	// support). Spec §4.
	QueueCount = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "reconciler_queue_count",
		Help: "DataStore list count: order/deposit/withdrawal/position/market",
	}, []string{"version", "list"})

	// MarketOI / MarketMaxOI / MarketPoolAmount / MarketMaxPoolAmount
	// backstop the market-oi VMProbe (same blackbox-exporter limitation).
	// Values are scaled to USD (1e30 for OI/maxOI, 1e18 for pool USDT).
	// Spec §5.
	MarketOI = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "reconciler_market_oi_usd",
		Help: "Open interest per market+side in USD",
	}, []string{"version", "market", "side"})

	MarketMaxOI = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "reconciler_market_max_oi_usd",
		Help: "Max allowed open interest per market+side in USD",
	}, []string{"version", "market", "side"})

	MarketPoolAmount = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "reconciler_market_pool_amount_usdt",
		Help: "USDT pool balance per market (long collateral side)",
	}, []string{"version", "market"})

	MarketMaxPoolAmount = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "reconciler_market_max_pool_amount_usdt",
		Help: "Max allowed USDT pool balance per market",
	}, []string{"version", "market"})

	// OracleMaxPriceAge backstops the oracle-v[12] VMProbe's
	// MAX_ORACLE_PRICE_AGE bytes32-key call. Spec §2.
	OracleMaxPriceAge = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "reconciler_oracle_max_price_age_seconds",
		Help: "DataStore.getUint(MAX_ORACLE_PRICE_AGE) — Oracle staleness ceiling",
	}, []string{"version"})
)
