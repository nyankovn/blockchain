package wallet

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"log"
	"math/big"
)

const (
	checksumLength = 4
	version        = byte(0x00)
)

type Wallet struct {
	PrivateKey ecdsa.PrivateKey
	PublicKey  []byte
}

func (w Wallet) Address() []byte {
	pubHash := PublicKeyHash(w.PublicKey)

	versionedHash := append([]byte{version}, pubHash...)
	checksum := Checksum(versionedHash)

	fullHash := append(versionedHash, checksum...)

	address := Base58Encode(fullHash)

	fmt.Printf("pub key: %x\n", w.PublicKey)
	fmt.Printf("pub hash: %x\n", pubHash)
	fmt.Printf("address: %x\n", address)

	return address
}

func NewKeyPair() (ecdsa.PrivateKey, []byte) {
	curve := elliptic.P256()

	private, err := ecdsa.GenerateKey(curve, rand.Reader)
	if err != nil {
		log.Panic(err)
	}

	pub := append(private.PublicKey.X.Bytes(), private.PublicKey.Y.Bytes()...)
	return *private, pub
}

func MakeWallet() *Wallet {
	private, public := NewKeyPair()
	wallet := Wallet{private, public}

	return &wallet
}

func PublicKeyHash(pubKey []byte) []byte {
	pubHash := sha256.Sum256(pubKey)

	//use alternative for the ripemd160.New(
	// hasher := ripemd160.New()
	// _, err := hasher.Write(pubHash[:])
	// if err != nil {
	// 	log.Panic(err)
	// }

	publicRipMD := sha256.Sum256(pubHash[:])
	return publicRipMD[:]
}

func Checksum(payload []byte) []byte {
	firstHash := sha256.Sum256(payload)
	secondHash := sha256.Sum256(firstHash[:])

	return secondHash[:checksumLength]
}

func (w *Wallet) Serialize() ([]byte, error) {
	buf := new(bytes.Buffer)

	// Serialize PrivateKey
	dBytes := w.PrivateKey.D.Bytes()
	if err := binary.Write(buf, binary.LittleEndian, int32(len(dBytes))); err != nil {
		return nil, err
	}
	if _, err := buf.Write(dBytes); err != nil {
		return nil, err
	}

	xBytes := w.PrivateKey.X.Bytes()
	if err := binary.Write(buf, binary.LittleEndian, int32(len(xBytes))); err != nil {
		return nil, err
	}
	if _, err := buf.Write(xBytes); err != nil {
		return nil, err
	}

	yBytes := w.PrivateKey.Y.Bytes()
	if err := binary.Write(buf, binary.LittleEndian, int32(len(yBytes))); err != nil {
		return nil, err
	}
	if _, err := buf.Write(yBytes); err != nil {
		return nil, err
	}

	// Serialize PublicKey
	if err := binary.Write(buf, binary.LittleEndian, int32(len(w.PublicKey))); err != nil {
		return nil, err
	}
	if _, err := buf.Write(w.PublicKey); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func (w *Wallet) Deserialize(data []byte) error {
	buf := bytes.NewReader(data)

	// Deserialize PrivateKey
	var length int32
	if err := binary.Read(buf, binary.LittleEndian, &length); err != nil {
		return err
	}
	dBytes := make([]byte, length)
	if _, err := buf.Read(dBytes); err != nil {
		return err
	}
	w.PrivateKey.D = new(big.Int).SetBytes(dBytes)

	if err := binary.Read(buf, binary.LittleEndian, &length); err != nil {
		return err
	}
	xBytes := make([]byte, length)
	if _, err := buf.Read(xBytes); err != nil {
		return err
	}
	w.PrivateKey.X = new(big.Int).SetBytes(xBytes)

	if err := binary.Read(buf, binary.LittleEndian, &length); err != nil {
		return err
	}
	yBytes := make([]byte, length)
	if _, err := buf.Read(yBytes); err != nil {
		return err
	}
	w.PrivateKey.Y = new(big.Int).SetBytes(yBytes)

	// Deserialize PublicKey
	if err := binary.Read(buf, binary.LittleEndian, &length); err != nil {
		return err
	}
	w.PublicKey = make([]byte, length)
	if _, err := buf.Read(w.PublicKey); err != nil {
		return err
	}

	return nil
}

func ValidateAddress(address string) bool {
	pubKeyHash := Base58Decode([]byte(address))
	actualChecksum := pubKeyHash[len(pubKeyHash)-checksumLength:]
	version := pubKeyHash[0]
	pubKeyHash = pubKeyHash[1 : len(pubKeyHash)-checksumLength]
	targetChecksum := Checksum(append([]byte{version}, pubKeyHash...))

	return bytes.Compare(actualChecksum, targetChecksum) == 0
}
