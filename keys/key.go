package keys

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"

	"github.com/pkg/errors"
	"golang.org/x/crypto/ripemd160"

	"github.com/netcloth/netcloth-chain/modules/auth"
	"github.com/netcloth/netcloth-chain/types"
	ctypes "github.com/netcloth/netcloth-chain/types"

	"github.com/netcloth/go-sdk/types/tx"

	tceec_secp256k1 "github.com/btcsuite/btcd/btcec"
	"github.com/netcloth/netcloth-chain/crypto/keys/mintkey"
	"github.com/tendermint/tendermint/crypto"
	"github.com/tendermint/tendermint/crypto/secp256k1"
)

type KeyManager interface {
	Sign(msg tx.StdSignMsg) ([]byte, error)
	SignBytes(msg []byte) ([]byte, error)
	GetPrivKey() crypto.PrivKey
	GetAddr() types.AccAddress
	GetUCPubKey() (UCPubKey []byte, err error)
}

type keyManager struct {
	privKey  crypto.PrivKey
	addr     types.AccAddress
	mnemonic string
}

func (k *keyManager) Sign(msg tx.StdSignMsg) ([]byte, error) {
	sig, err := k.makeSignature(msg)
	if err != nil {
		return nil, err
	}

	newTx := auth.NewStdTx(msg.Msgs, msg.Fee, []auth.StdSignature{sig}, msg.Memo)
	bz, err := tx.Cdc.MarshalBinaryLengthPrefixed(newTx)
	if err != nil {
		return nil, err
	}

	return bz, nil
}

func (k *keyManager) SignBytes(msg []byte) ([]byte, error) {
	return k.privKey.Sign(msg)
}

func (k *keyManager) GetPrivKey() crypto.PrivKey {
	return k.privKey
}

func (k *keyManager) GetAddr() types.AccAddress {
	return k.addr
}

func (k *keyManager) GetUCPubKey() (UCPubKey []byte, err error) {
	pubkey, err := tceec_secp256k1.ParsePubKey(k.GetPrivKey().PubKey().Bytes()[5:], tceec_secp256k1.S256())
	if err != nil {
		return nil, err
	}

	return pubkey.SerializeUncompressed(), nil
}

/*
	params:
		uncompressedPubKey: "04b2bf9a87dd7cf1ad998721ffef00713a4d5fb2bae0316eea04268ae877a0bcacd41b5b363911a30c0254ca12148d48e3cd4562e3e4b5d8cd3e6d2107a69754e6"
	return value:
		compressedPubKey: 33 bytes compressed pubkey
		err: is nil if success
*/
func GetCompressedPubKey(uncompressedPubKey string) (compressedPubKey []byte, err error) {
	uncompressedPubKeyHex, err := hex.DecodeString(uncompressedPubKey)
	if err != nil {
		return nil, err
	}

	pubkey, err := tceec_secp256k1.ParsePubKey(uncompressedPubKeyHex, tceec_secp256k1.S256())
	if err != nil {
		return nil, err
	}

	return pubkey.SerializeCompressed(), nil
}

func GetCompressedAddress(uncompressedPubKey string) (crypto.Address, error) {
	pubKey, err := GetCompressedPubKey(uncompressedPubKey)
	if err != nil {
		return nil, err
	}

	hasherSHA256 := sha256.New()
	hasherSHA256.Write(pubKey[:])
	sha := hasherSHA256.Sum(nil)

	hasherRIPEMD160 := ripemd160.New()
	hasherRIPEMD160.Write(sha)
	return hasherRIPEMD160.Sum(nil), nil
}

func GetUCAddressBech32(uncompressedPubKey string) (string, error) {
	addr, err := GetCompressedAddress(uncompressedPubKey)
	if err != nil {
		return "", err
	}

	return types.AccAddress(addr).String(), nil
}

func GetBech32AddrByPubkeyStr(pubkeyStr string) (string, error) {
	if len(pubkeyStr) == 0 {
		return "", fmt.Errorf("pubkey invalid")
	}

	pubkeyHex, err := hex.DecodeString(pubkeyStr)
	if err != nil {
		return "", err
	}

	var pk secp256k1.PubKeySecp256k1
	copy(pk[:], pubkeyHex)
	addr := types.AccAddress(pk.Address().Bytes())

	return addr.String(), nil
}

func GetBech32AddrByPubkey(pubkey secp256k1.PubKeySecp256k1) (string, error) {
	return types.AccAddress(pubkey.Address().Bytes()).String(), nil
}

func (k *keyManager) makeSignature(msg tx.StdSignMsg) (sig auth.StdSignature, err error) {
	if err != nil {
		return
	}
	sigBytes, err := k.privKey.Sign(msg.Bytes())
	if err != nil {
		return
	}
	return auth.StdSignature{
		PubKey:    k.privKey.PubKey(),
		Signature: sigBytes,
	}, nil
}

func (k *keyManager) recoveryFromKeyStore(keystoreFile string, auth string) error {
	if auth == "" {
		return fmt.Errorf("Password is missing ")
	}
	keyJson, err := ioutil.ReadFile(keystoreFile)
	if err != nil {
		return err
	}
	var encryptedKey EncryptedKeyJSON
	err = json.Unmarshal(keyJson, &encryptedKey)
	if err != nil {
		return err
	}
	keyBytes, err := decryptKey(&encryptedKey, auth)
	if err != nil {
		return err
	}
	if len(keyBytes) != 32 {
		return fmt.Errorf("Len of Keybytes is not equal to 32 ")
	}
	var keyBytesArray [32]byte
	copy(keyBytesArray[:], keyBytes[:32])
	privKey := secp256k1.PrivKeySecp256k1(keyBytesArray)
	addr := ctypes.AccAddress(privKey.PubKey().Address())
	k.addr = addr
	k.privKey = privKey
	return nil
}

func (k *keyManager) ImportKeystore(keystoreFile string, passphrase string) error {
	if passphrase == "" {
		return fmt.Errorf("Password is missing ")
	}

	armor, err := ioutil.ReadFile(keystoreFile)
	if err != nil {
		return err
	}

	privKey, err := mintkey.UnarmorDecryptPrivKey(string(armor), passphrase)
	if err != nil {
		return errors.Wrap(err, "couldn't import private key")
	}

	addr := ctypes.AccAddress(privKey.PubKey().Address())
	k.addr = addr
	k.privKey = privKey
	return nil
}

func NewKeyStoreKeyManager(file string, auth string) (KeyManager, error) {
	k := keyManager{}
	err := k.recoveryFromKeyStore(file, auth)
	return &k, err
}

func NewKeystoreByImportKeystore(file string, auth string) (KeyManager, error) {
	k := keyManager{}
	err := k.ImportKeystore(file, auth)
	return &k, err
}
