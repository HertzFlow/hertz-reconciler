package chain

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

// IMPORTANT: GMX-style DataStore and RoleStore constants are all hashed as
// `keccak256(abi.encode("NAME"))`, not `keccak256("NAME")`. The difference is
// subtle but silently makes every downstream lookup miss. Every top-level
// root and helper in this file obeys abi.encode.

// abiEncodeString matches Solidity `abi.encode(string)` for a single string
// arg: 32-byte offset (=0x20) || 32-byte length || right-padded bytes.
func abiEncodeString(s string) []byte {
	data := []byte(s)
	padded := ((len(data) + 31) / 32) * 32

	out := make([]byte, 0, 64+padded)
	offset := make([]byte, 32)
	offset[31] = 0x20
	out = append(out, offset...)

	lenBuf := make([]byte, 32)
	n := uint64(len(data))
	for i := 0; i < 8; i++ {
		lenBuf[31-i] = byte(n >> (8 * i))
	}
	out = append(out, lenBuf...)

	body := make([]byte, padded)
	copy(body, data)
	out = append(out, body...)
	return out
}

func rootHash(name string) common.Hash {
	return crypto.Keccak256Hash(abiEncodeString(name))
}

// Pre-computed top-level DataStore keccak roots.
var (
	OpenInterestRoot          = rootHash("OPEN_INTEREST")
	PoolAmountRoot            = rootHash("POOL_AMOUNT")
	MaxOpenInterestRoot       = rootHash("MAX_OPEN_INTEREST")
	MaxPoolAmountRoot         = rootHash("MAX_POOL_AMOUNT")
	MinCollateralRoot         = rootHash("MIN_COLLATERAL_FACTOR")
	BorrowingFactorRoot       = rootHash("BORROWING_FACTOR")
	LiquidationFeeRoot        = rootHash("LIQUIDATION_FEE_FACTOR")
	MaxOraclePriceAgeRoot     = rootHash("MAX_ORACLE_PRICE_AGE")
	MarketListRoot            = rootHash("MARKET_LIST")
	OrderListRoot             = rootHash("ORDER_LIST")
	DepositListRoot           = rootHash("DEPOSIT_LIST")
	WithdrawalListRoot        = rootHash("WITHDRAWAL_LIST")
	PositionListRoot          = rootHash("POSITION_LIST")
	ClaimableFeeAmountRoot    = rootHash("CLAIMABLE_FEE_AMOUNT")
	FundingFeeAmountPerSizeRoot = rootHash("FUNDING_FEE_AMOUNT_PER_SIZE")
)

func keccakOf(parts ...[]byte) common.Hash {
	var buf []byte
	for _, p := range parts {
		buf = append(buf, p...)
	}
	return crypto.Keccak256Hash(buf)
}

func addressPadded(a common.Address) []byte {
	out := make([]byte, 32)
	copy(out[12:], a.Bytes())
	return out
}

func boolPadded(b bool) []byte {
	out := make([]byte, 32)
	if b {
		out[31] = 1
	}
	return out
}

// MinCollateralFactorKey = keccak256(abi.encode(MIN_COLLATERAL_FACTOR, market))
func MinCollateralFactorKey(market common.Address) common.Hash {
	return keccakOf(MinCollateralRoot.Bytes(), addressPadded(market))
}

// MaxOpenInterestKey = keccak256(abi.encode(MAX_OPEN_INTEREST, market, isLong))
func MaxOpenInterestKey(market common.Address, isLong bool) common.Hash {
	return keccakOf(MaxOpenInterestRoot.Bytes(), addressPadded(market), boolPadded(isLong))
}

// PoolAmountKey = keccak256(abi.encode(POOL_AMOUNT, market, token))
func PoolAmountKey(market, token common.Address) common.Hash {
	return keccakOf(PoolAmountRoot.Bytes(), addressPadded(market), addressPadded(token))
}

// MaxPoolAmountKey = keccak256(abi.encode(MAX_POOL_AMOUNT, market, token))
func MaxPoolAmountKey(market, token common.Address) common.Hash {
	return keccakOf(MaxPoolAmountRoot.Bytes(), addressPadded(market), addressPadded(token))
}

// OpenInterestKey = keccak256(abi.encode(OPEN_INTEREST, market, collateralToken, isLong))
func OpenInterestKey(market, collateralToken common.Address, isLong bool) common.Hash {
	return keccakOf(OpenInterestRoot.Bytes(), addressPadded(market), addressPadded(collateralToken), boolPadded(isLong))
}

// BorrowingFactorKey = keccak256(abi.encode(BORROWING_FACTOR, market, isLong))
func BorrowingFactorKey(market common.Address, isLong bool) common.Hash {
	return keccakOf(BorrowingFactorRoot.Bytes(), addressPadded(market), boolPadded(isLong))
}

// LiquidationFeeFactorKey = keccak256(abi.encode(LIQUIDATION_FEE_FACTOR, market))
func LiquidationFeeFactorKey(market common.Address) common.Hash {
	return keccakOf(LiquidationFeeRoot.Bytes(), addressPadded(market))
}

// ClaimableFeeAmountKey = keccak256(abi.encode(CLAIMABLE_FEE_AMOUNT, market, token))
func ClaimableFeeAmountKey(market, token common.Address) common.Hash {
	return keccakOf(ClaimableFeeAmountRoot.Bytes(), addressPadded(market), addressPadded(token))
}

// FundingFeeAmountPerSizeKey = keccak256(abi.encode(FUNDING_FEE_AMOUNT_PER_SIZE, market, collateralToken, isLong))
func FundingFeeAmountPerSizeKey(market, collateralToken common.Address, isLong bool) common.Hash {
	return keccakOf(FundingFeeAmountPerSizeRoot.Bytes(), addressPadded(market), addressPadded(collateralToken), boolPadded(isLong))
}

// RoleHash matches the on-chain `bytes32 constant ROLE = keccak256(abi.encode("NAME"))`.
func RoleHash(name string) common.Hash {
	return rootHash(name)
}
