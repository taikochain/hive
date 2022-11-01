package taiko

import (
	"crypto/ecdsa"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/hive/hivesim"
)

type Account struct {
	PrivateKeyHex string
	PrivateKey    *ecdsa.PrivateKey // private key of the account
	Address       common.Address    // address of the account
}

type Accounts struct {
	L1Deployer            *Account // l1 deployer account for deploy l1 contracts: bridge, rollup l1, vault
	SuggestedFeeRecipient *Account // suggested fee recipient account
	Prover                *Account // l1 prover account for prove zk proof
	Proposer              *Account // l1 proposer account for propose l1 txList
	Throwawayer           *Account // l2 driver account for throwaway invalid block
	// test accounts for both l1 and l2
	Person1 *Account
	Person2 *Account
	Person3 *Account
	Person4 *Account
	Person5 *Account
	Person6 *Account
	Person7 *Account
	Person8 *Account
	Person9 *Account
}

func NewAccount(t *hivesim.T, privKeyHex string) *Account {
	privKey, err := crypto.HexToECDSA(privKeyHex)
	if err != nil {
		t.Fatalf("failed to parse private key: %v", err)
	}
	addr := crypto.PubkeyToAddress(privKey.PublicKey)
	return &Account{
		PrivateKeyHex: privKeyHex,
		PrivateKey:    privKey,
		Address:       addr,
	}
}

var (
	DefaultAccounts = func(test *hivesim.T) *Accounts {
		return &Accounts{
			L1Deployer:            NewAccount(test, "2bdd21761a483f71054e14f5b827213567971c676928d9a1808cbfa4b7501200"),
			SuggestedFeeRecipient: NewAccount(test, "2bdd21761a483f71054e14f5b827213567971c676928d9a1808cbfa4b7501200"),
			Prover:                NewAccount(test, "6bff9a8ffd7f94f43f4f5f642be8a3f32a94c1f316d90862884b2e276293b6ee"),
			Proposer:              NewAccount(test, "2bdd21761a483f71054e14f5b827213567971c676928d9a1808cbfa4b7501200"),
			Throwawayer:           NewAccount(test, "2bdd21761a483f71054e14f5b827213567971c676928d9a1808cbfa4b7501200"),
			Person1:               NewAccount(test, "701b615bbdfb9de65240bc28bd21bbc0d996645a3dd57e7b12bc2bdf6f192c82"),
			Person2:               NewAccount(test, "a267530f49f8280200edf313ee7af6b827f2a8bce2897751d06a843f644967b1"),
			Person3:               NewAccount(test, "47c99abed3324a2707c28affff1267e45918ec8c3f20b8aa892e8b065d2942dd"),
			Person4:               NewAccount(test, "c526ee95bf44d8fc405a158bb884d9d1238d99f0612e9f33d006bb0789009aaa"),
			Person5:               NewAccount(test, "8166f546bab6da521a8369cab06c5d2b9e46670292d85c875ee9ec20e84ffb61"),
			Person6:               NewAccount(test, "ea6c44ac03bff858b476bba40716402b03e41b8e97e276d1baec7c37d42484a0"),
			Person7:               NewAccount(test, "689af8efa8c651a91ad287602527f3af2fe9f6501a7ac4b061667b5a93e037fd"),
			Person8:               NewAccount(test, "de9be858da4a475276426320d5e9262ecfc3ba460bfac56360bfa6c4c28b4ee0"),
			Person9:               NewAccount(test, "df57089febbacf7ba0bc227dafbffa9fc08a93fdc68e1e42411a14efcf23656e"),
		}
	}
)
