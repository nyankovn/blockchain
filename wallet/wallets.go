package wallet

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"log"
	"os"
)

const walletFile = "./tmp/wallets_%s.data"

type Wallets struct {
	Wallets map[string]*Wallet
}

func (ws *Wallets) SaveFile(nodeID string) {
	walletFile := fmt.Sprintf(walletFile, nodeID)

	data, err := ws.Serialize()
	if err != nil {
		log.Panic(err)
	}

	err = os.WriteFile(walletFile, data, 0644)
	if err != nil {
		log.Panic(err)
	}
}

func CreateWallets(nodeID string) (*Wallets, error) {
	wallets := Wallets{}
	wallets.Wallets = make(map[string]*Wallet)

	err := wallets.LoadFile(nodeID)

	return &wallets, err
}

func (ws *Wallets) AddWallet() string {
	wallet := MakeWallet()
	address := fmt.Sprintf("%s", wallet.Address())

	ws.Wallets[address] = wallet

	return address
}

func (ws Wallets) GetWallet(address string) Wallet {
	return *ws.Wallets[address]
}

func (ws *Wallets) GetAllAddresses() []string {
	var addresses []string
	for address := range ws.Wallets {
		addresses = append(addresses, address)
	}
	return addresses
}

func (ws *Wallets) LoadFile(nodeID string) error {
	walletFile := fmt.Sprintf(walletFile, nodeID)

	if _, err := os.Stat(walletFile); os.IsNotExist(err) {
		return err
	}

	fileContent, err := os.ReadFile(walletFile)
	if err != nil {
		return err
	}

	err = ws.Deserialize(fileContent)
	if err != nil {
		return err
	}

	return nil
}

func (ws *Wallets) Serialize() ([]byte, error) {
	buf := new(bytes.Buffer)

	// Serialize the number of wallets
	if err := binary.Write(buf, binary.LittleEndian, int32(len(ws.Wallets))); err != nil {
		return nil, err
	}

	// Serialize each wallet
	for key, wallet := range ws.Wallets {
		// Serialize the key
		keyBytes := []byte(key)
		if err := binary.Write(buf, binary.LittleEndian, int32(len(keyBytes))); err != nil {
			return nil, err
		}
		if _, err := buf.Write(keyBytes); err != nil {
			return nil, err
		}

		// Serialize the wallet
		walletBytes, err := wallet.Serialize()
		if err != nil {
			return nil, err
		}
		if err := binary.Write(buf, binary.LittleEndian, int32(len(walletBytes))); err != nil {
			return nil, err
		}
		if _, err := buf.Write(walletBytes); err != nil {
			return nil, err
		}
	}

	return buf.Bytes(), nil
}

func (ws *Wallets) Deserialize(data []byte) error {
	buf := bytes.NewReader(data)

	// Deserialize the number of wallets
	var count int32
	if err := binary.Read(buf, binary.LittleEndian, &count); err != nil {
		return err
	}

	ws.Wallets = make(map[string]*Wallet)
	for i := int32(0); i < count; i++ {
		// Deserialize the key
		var keyLen int32
		if err := binary.Read(buf, binary.LittleEndian, &keyLen); err != nil {
			return err
		}
		keyBytes := make([]byte, keyLen)
		if _, err := buf.Read(keyBytes); err != nil {
			return err
		}
		key := string(keyBytes)

		// Deserialize the wallet
		var walletLen int32
		if err := binary.Read(buf, binary.LittleEndian, &walletLen); err != nil {
			return err
		}
		walletBytes := make([]byte, walletLen)
		if _, err := buf.Read(walletBytes); err != nil {
			return err
		}

		wallet := &Wallet{}
		if err := wallet.Deserialize(walletBytes); err != nil {
			return err
		}

		ws.Wallets[key] = wallet
	}

	return nil
}
