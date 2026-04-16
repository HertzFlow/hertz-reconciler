package chain

import (
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
)

// Pre-computed function selectors used by reconciler probes.
var (
	selGetUint           = mustSelector("getUint(bytes32)")
	selGetInt            = mustSelector("getInt(bytes32)")
	selGetBool           = mustSelector("getBool(bytes32)")
	selGetAddressCount   = mustSelector("getAddressCount(bytes32)")
	selGetBytes32Count   = mustSelector("getBytes32Count(bytes32)")
	selGetRoleMemberCount = mustSelector("getRoleMemberCount(bytes32)")
	selGetRoleMembers    = mustSelector("getRoleMembers(bytes32,uint256,uint256)")
	selGetMinDelay       = mustSelector("getMinDelay()")
	selErc20BalanceOf    = mustSelector("balanceOf(address)")
)

func mustSelector(sig string) []byte { return crypto.Keccak256([]byte(sig))[:4] }

// Client wraps an ethclient with helpers tailored to GMX-style DataStore /
// RoleStore reads. Calls are bounded by RequestTimeout.
type Client struct {
	eth     *ethclient.Client
	rpc     *rpc.Client
	timeout time.Duration
}

func NewClient(rpcURL string, timeout time.Duration) (*Client, error) {
	r, err := rpc.Dial(rpcURL)
	if err != nil {
		return nil, fmt.Errorf("dial rpc: %w", err)
	}
	return &Client{eth: ethclient.NewClient(r), rpc: r, timeout: timeout}, nil
}

func (c *Client) Close() { c.rpc.Close() }

func (c *Client) ctx() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), c.timeout)
}

// BalanceOfNative returns native BNB balance in wei.
func (c *Client) BalanceOfNative(addr common.Address) (*big.Int, error) {
	ctx, cancel := c.ctx()
	defer cancel()
	return c.eth.BalanceAt(ctx, addr, nil)
}

// BalanceOfERC20 calls ERC20.balanceOf(addr).
func (c *Client) BalanceOfERC20(token, holder common.Address) (*big.Int, error) {
	data := append([]byte{}, selErc20BalanceOf...)
	data = append(data, addressPadded(holder)...)
	out, err := c.call(token, data)
	if err != nil {
		return nil, err
	}
	return new(big.Int).SetBytes(out), nil
}

// GetUint returns DataStore.getUint(key) as big.Int.
func (c *Client) GetUint(dataStore common.Address, key common.Hash) (*big.Int, error) {
	data := append([]byte{}, selGetUint...)
	data = append(data, key.Bytes()...)
	out, err := c.call(dataStore, data)
	if err != nil {
		return nil, err
	}
	return new(big.Int).SetBytes(out), nil
}

// GetInt returns DataStore.getInt(key) as signed big.Int.
func (c *Client) GetInt(dataStore common.Address, key common.Hash) (*big.Int, error) {
	data := append([]byte{}, selGetInt...)
	data = append(data, key.Bytes()...)
	out, err := c.call(dataStore, data)
	if err != nil {
		return nil, err
	}
	v := new(big.Int).SetBytes(out)
	// reinterpret as signed two's-complement (256 bit)
	if v.Bit(255) == 1 {
		two256 := new(big.Int).Lsh(big.NewInt(1), 256)
		v.Sub(v, two256)
	}
	return v, nil
}

// GetBool returns DataStore.getBool(key).
func (c *Client) GetBool(dataStore common.Address, key common.Hash) (bool, error) {
	data := append([]byte{}, selGetBool...)
	data = append(data, key.Bytes()...)
	out, err := c.call(dataStore, data)
	if err != nil {
		return false, err
	}
	if len(out) == 0 {
		return false, nil
	}
	return out[len(out)-1] != 0, nil
}

// GetBytes32Count returns DataStore.getBytes32Count(key).
func (c *Client) GetBytes32Count(dataStore common.Address, key common.Hash) (uint64, error) {
	data := append([]byte{}, selGetBytes32Count...)
	data = append(data, key.Bytes()...)
	out, err := c.call(dataStore, data)
	if err != nil {
		return 0, err
	}
	return new(big.Int).SetBytes(out).Uint64(), nil
}

// GetAddressCount returns DataStore.getAddressCount(key).
func (c *Client) GetAddressCount(dataStore common.Address, key common.Hash) (uint64, error) {
	data := append([]byte{}, selGetAddressCount...)
	data = append(data, key.Bytes()...)
	out, err := c.call(dataStore, data)
	if err != nil {
		return 0, err
	}
	return new(big.Int).SetBytes(out).Uint64(), nil
}

// GetRoleMemberCount returns RoleStore.getRoleMemberCount(role).
func (c *Client) GetRoleMemberCount(roleStore common.Address, role common.Hash) (uint64, error) {
	data := append([]byte{}, selGetRoleMemberCount...)
	data = append(data, role.Bytes()...)
	out, err := c.call(roleStore, data)
	if err != nil {
		return 0, err
	}
	return new(big.Int).SetBytes(out).Uint64(), nil
}

// GetRoleMembers returns RoleStore.getRoleMembers(role, start, end). Decodes
// the dynamic address[] return value.
func (c *Client) GetRoleMembers(roleStore common.Address, role common.Hash, start, end uint64) ([]common.Address, error) {
	data := append([]byte{}, selGetRoleMembers...)
	data = append(data, role.Bytes()...)
	data = append(data, padUint(start)...)
	data = append(data, padUint(end)...)
	out, err := c.call(roleStore, data)
	if err != nil {
		return nil, err
	}
	return decodeAddressArray(out)
}

// GetMinDelay returns Timelock.getMinDelay() in seconds.
func (c *Client) GetMinDelay(timelock common.Address) (uint64, error) {
	out, err := c.call(timelock, selGetMinDelay)
	if err != nil {
		return 0, err
	}
	return new(big.Int).SetBytes(out).Uint64(), nil
}

func (c *Client) call(to common.Address, data []byte) ([]byte, error) {
	ctx, cancel := c.ctx()
	defer cancel()
	var hex string
	args := map[string]interface{}{
		"to":   to.Hex(),
		"data": "0x" + bytesToHex(data),
	}
	if err := c.rpc.CallContext(ctx, &hex, "eth_call", args, "latest"); err != nil {
		return nil, fmt.Errorf("eth_call %s: %w", to.Hex(), err)
	}
	return hexToBytes(hex), nil
}

func padUint(v uint64) []byte {
	b := big.NewInt(0).SetUint64(v).Bytes()
	out := make([]byte, 32)
	copy(out[32-len(b):], b)
	return out
}

func bytesToHex(b []byte) string {
	const hexChars = "0123456789abcdef"
	out := make([]byte, len(b)*2)
	for i, v := range b {
		out[i*2] = hexChars[v>>4]
		out[i*2+1] = hexChars[v&0x0f]
	}
	return string(out)
}

func hexToBytes(s string) []byte {
	if len(s) >= 2 && s[:2] == "0x" {
		s = s[2:]
	}
	if len(s)%2 != 0 {
		return nil
	}
	out := make([]byte, len(s)/2)
	for i := 0; i < len(s)/2; i++ {
		hi := hexNibble(s[i*2])
		lo := hexNibble(s[i*2+1])
		if hi < 0 || lo < 0 {
			return nil
		}
		out[i] = byte(hi<<4 | lo)
	}
	return out
}

func hexNibble(c byte) int {
	switch {
	case c >= '0' && c <= '9':
		return int(c - '0')
	case c >= 'a' && c <= 'f':
		return int(c-'a') + 10
	case c >= 'A' && c <= 'F':
		return int(c-'A') + 10
	}
	return -1
}

// decodeAddressArray decodes a Solidity dynamic address[] return blob.
//
//	[0:32]   = offset to array data (usually 0x20)
//	[off:..] = uint256 length, followed by length×32-byte slots (each slot is
//	           a left-padded address)
func decodeAddressArray(out []byte) ([]common.Address, error) {
	if len(out) < 64 {
		return nil, nil
	}
	offset := new(big.Int).SetBytes(out[0:32]).Uint64()
	if uint64(len(out)) < offset+32 {
		return nil, fmt.Errorf("decodeAddressArray: short payload")
	}
	length := new(big.Int).SetBytes(out[offset : offset+32]).Uint64()
	body := out[offset+32:]
	if uint64(len(body)) < length*32 {
		return nil, fmt.Errorf("decodeAddressArray: body too short for %d entries", length)
	}
	res := make([]common.Address, length)
	for i := uint64(0); i < length; i++ {
		slot := body[i*32 : (i+1)*32]
		res[i] = common.BytesToAddress(slot[12:])
	}
	return res, nil
}
