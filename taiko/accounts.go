package taiko

import (
	"crypto/ecdsa"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/hive/hivesim"
)

type Account struct {
	PrivateKeyHex string
	PrivateKey    *ecdsa.PrivateKey
	Address       common.Address
}

// TestAccounts test accounts for both l1 and l2
type TestAccounts struct {
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
	DefaultAccounts = func(t *hivesim.T) *TestAccounts {
		return &TestAccounts{
			Person1: NewAccount(t, "701b615bbdfb9de65240bc28bd21bbc0d996645a3dd57e7b12bc2bdf6f192c82"),
			Person2: NewAccount(t, "a267530f49f8280200edf313ee7af6b827f2a8bce2897751d06a843f644967b1"),
			Person3: NewAccount(t, "47c99abed3324a2707c28affff1267e45918ec8c3f20b8aa892e8b065d2942dd"),
			Person4: NewAccount(t, "c526ee95bf44d8fc405a158bb884d9d1238d99f0612e9f33d006bb0789009aaa"),
			Person5: NewAccount(t, "8166f546bab6da521a8369cab06c5d2b9e46670292d85c875ee9ec20e84ffb61"),
			Person6: NewAccount(t, "ea6c44ac03bff858b476bba40716402b03e41b8e97e276d1baec7c37d42484a0"),
			Person7: NewAccount(t, "689af8efa8c651a91ad287602527f3af2fe9f6501a7ac4b061667b5a93e037fd"),
			Person8: NewAccount(t, "de9be858da4a475276426320d5e9262ecfc3ba460bfac56360bfa6c4c28b4ee0"),
			Person9: NewAccount(t, "df57089febbacf7ba0bc227dafbffa9fc08a93fdc68e1e42411a14efcf23656e"),
		}
	}
)
