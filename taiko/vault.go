package taiko

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"fmt"
	"math/big"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/hive/hivesim"
)

var (
	// simulator test account
	VaultAddr = common.HexToAddress("0xf39fd6e51aad88f6f4ce6ab8827279cfffb92266")
	// This is the account that sends vault funding transactions.
	vaultKey, _ = crypto.HexToECDSA("ac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80")
	// Number of blocks to wait before funding tx is considered valid.
	vaultTxConfirmationCount = uint64(1)
)

// Vault creates accounts for testing and funds them. An instance of the Vault contract is deployed in
// the genesis block. When creating a new account using NewAccoount, the account is funded by sending a
// transaction to this contract.
type Vault struct {
	sync.Mutex
	t *hivesim.T

	nonce uint64 // track the nonce of the vault account

	chainID  *big.Int
	accounts map[common.Address]*ecdsa.PrivateKey
}

func NewVault(t *hivesim.T, chainID *big.Int) *Vault {
	return &Vault{
		t:        t,
		chainID:  chainID,
		accounts: make(map[common.Address]*ecdsa.PrivateKey),
	}
}

func (v *Vault) GenerateKey() common.Address {
	key, err := crypto.GenerateKey()
	if err != nil {
		panic(fmt.Errorf("can't generate account key: %w", err))
	}
	addr := crypto.PubkeyToAddress(key.PublicKey)

	v.Lock()
	defer v.Unlock()
	v.accounts[addr] = key
	return addr
}

func (v *Vault) FindKey(addr common.Address) *ecdsa.PrivateKey {
	v.Lock()
	defer v.Unlock()
	return v.accounts[addr]
}

func (v *Vault) SignTransaction(sender common.Address, tx *types.Transaction) (*types.Transaction, error) {
	key := v.FindKey(sender)
	if key == nil {
		return nil, fmt.Errorf("can't find private key for account %v", sender)
	}
	signer := types.LatestSignerForChainID(v.chainID)
	return types.SignTx(tx, signer, key)
}

func (v *Vault) CreateAccount(ctx context.Context, client *ethclient.Client, amount *big.Int) common.Address {
	if amount == nil {
		amount = big.NewInt(0)
	}
	addr := v.GenerateKey()

	tx := v.makeFundingTx(addr, amount)
	if err := client.SendTransaction(ctx, tx); err != nil {
		v.t.Fatalf("unable to send funding transaction: %w", err)
	}

	// wait for receipt until timeout
	for i := 0; i < 60; i++ {
		receipt, err := client.TransactionReceipt(ctx, tx.Hash())
		if err != nil && !errors.Is(err, ethereum.NotFound) {
			v.t.Fatalf("error getting transaction receipt", err)
		}
		if receipt != nil {
			return addr
		}
		time.Sleep(time.Second)
	}
	v.t.Fatal("timeout getting transaction receipt")
	return common.Address{}
}

func (v *Vault) InsertKey(key *ecdsa.PrivateKey) {
	addr := crypto.PubkeyToAddress(key.PublicKey)

	v.Lock()
	defer v.Unlock()

	v.accounts[addr] = key
}

func (v *Vault) KeyedTransactor(addr common.Address) *bind.TransactOpts {
	opts, err := bind.NewKeyedTransactorWithChainID(v.FindKey(addr), v.chainID)
	if err != nil {
		v.t.Fatal("error getting keyed transactor:", err)
	}
	return opts
}

func (v *Vault) makeFundingTx(recipient common.Address, amount *big.Int) *types.Transaction {
	var (
		nonce    = v.nextNonce()
		gasLimit = uint64(75000)
	)
	tx := types.NewTx(&types.DynamicFeeTx{
		Nonce:     nonce,
		Gas:       gasLimit,
		GasTipCap: big.NewInt(1 * params.GWei),
		GasFeeCap: big.NewInt(30 * params.GWei),
		To:        &recipient,
		Value:     amount,
	})
	signer := types.LatestSignerForChainID(v.chainID)
	signedTx, err := types.SignTx(tx, signer, vaultKey)
	if err != nil {
		v.t.Fatal("can'T sign vault funding tx:", err)
	}
	return signedTx
}

// nextNonce generates the nonce of a funding transaction.
func (v *Vault) nextNonce() uint64 {
	v.Lock()
	defer v.Unlock()

	nonce := v.nonce
	v.nonce++
	return nonce
}
