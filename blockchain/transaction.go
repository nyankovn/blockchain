package blockchain

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/gob"
	"encoding/hex"
	"fmt"
	"log"
	"math/big"
	"strings"

	"github.com/nyankovn/blockchain/wallet"
)

type Transaction struct {
	ID      []byte
	Inputs  []TxInput
	Outputs []TxOutput
}

func NewTransaction(from, to string, amount int, chain *BlockChain) *Transaction {
	var inputs []TxInput
	var outputs []TxOutput

	wallets, err := wallet.CreateWallets()
	Handle(err)

	fmt.Println(1)
	w := wallets.GetWallet(from)
	fmt.Println(2)

	pubKeyHash := wallet.PublicKeyHash(w.PublicKey)
	fmt.Println(3)

	acc, validOutputs := chain.FindSpendableOutputs(pubKeyHash, amount)
	fmt.Println(4)

	if acc < amount {
		log.Panic("Error: not enough funds")
	}

	fmt.Println(5)

	for txid, outs := range validOutputs {
		fmt.Println(7)

		txID, err := hex.DecodeString(txid)
		Handle(err)
		fmt.Println(8)

		for _, out := range outs {
			input := TxInput{txID, out, nil, w.PublicKey}
			inputs = append(inputs, input)
			fmt.Println(9)

		}
	}
	fmt.Println(10)

	outputs = append(outputs, *NewTXOutput(amount, to))
	fmt.Println(11)

	if acc > amount {
		outputs = append(outputs, *NewTXOutput(acc-amount, from))
	}
	fmt.Println(12)

	tx := Transaction{nil, inputs, outputs}
	tx.ID = tx.Hash()
	fmt.Println(13)

	chain.SignTransaction(&tx, w.PrivateKey)
	fmt.Println(14)

	return &tx
}

func (tx *Transaction) SetID() {
	var encoded bytes.Buffer
	var hash [32]byte

	encode := gob.NewEncoder(&encoded)
	err := encode.Encode(tx)
	Handle(err)

	hash = sha256.Sum256(encoded.Bytes())
	tx.ID = hash[:]
}

func CoinbaseTx(to, data string) *Transaction {
	if data == "" {
		data = fmt.Sprintf("Coins to %s", to)
	}
	txIn := TxInput{[]byte{}, -1, nil, []byte(data)}
	txOut := NewTXOutput(100, to)

	tx := Transaction{nil, []TxInput{txIn}, []TxOutput{*txOut}}
	tx.SetID()

	return &tx
}

func (tx *Transaction) IsCoinbase() bool {
	return len(tx.Inputs) == 1 && len(tx.Inputs[0].ID) == 0 && tx.Inputs[0].Out == -1
}

func (tx Transaction) Serialize() []byte {
	var encoded bytes.Buffer

	enc := gob.NewEncoder(&encoded)
	err := enc.Encode(tx)
	if err != nil {
		log.Panic(err)
	}

	return encoded.Bytes()
}

func (tx *Transaction) Hash() []byte {
	var hash [32]byte

	txCopy := *tx
	txCopy.ID = []byte{}

	hash = sha256.Sum256(txCopy.Serialize())

	return hash[:]
}

func (tx *Transaction) Sign(privKey ecdsa.PrivateKey, prevTXs map[string]Transaction) {
	fmt.Println(30)
	if tx.IsCoinbase() {
		return
	}
	fmt.Println(31)

	for _, in := range tx.Inputs {
		if prevTXs[hex.EncodeToString(in.ID)].ID == nil {
			log.Panic("ERROR: previous transaction does not exist")
		}
	}

	fmt.Println(32)

	txCopy := tx.TrimmedCopy()

	fmt.Println(33)

	for inId, in := range txCopy.Inputs {
		fmt.Println(34)

		prevTX := prevTXs[hex.EncodeToString(in.ID)]
		txCopy.Inputs[inId].Signature = nil
		txCopy.Inputs[inId].PubKey = prevTX.Outputs[in.Out].PubKeyHash
		txCopy.ID = txCopy.Hash()
		txCopy.Inputs[inId].PubKey = nil

		fmt.Println(35)
		privKey.PublicKey.Curve = elliptic.P256() // Change this to the correct curve if needed

		fmt.Printf("txCopy.ID: %x\n", txCopy.ID)
		fmt.Printf("privKey: %+v\n", privKey)
		r, s, err := ecdsa.Sign(rand.Reader, &privKey, txCopy.ID)
		// Print the signature components for verification
		fmt.Printf("r: %s\n", r.Text(16))
		fmt.Printf("s: %s\n", s.Text(16))
		fmt.Println(355)

		fmt.Println(err)
		Handle(err)

		fmt.Println(36)

		signature := append(r.Bytes(), s.Bytes()...)
		fmt.Println(37)

		tx.Inputs[inId].Signature = signature
		fmt.Println(38)

	}
}

func (tx *Transaction) TrimmedCopy() Transaction {
	var inputs []TxInput
	var outputs []TxOutput

	for _, in := range tx.Inputs {
		inputs = append(inputs, TxInput{in.ID, in.Out, nil, nil})
	}

	for _, out := range tx.Outputs {
		outputs = append(outputs, TxOutput{out.Value, out.PubKeyHash})
	}

	txCopy := Transaction{tx.ID, inputs, outputs}

	return txCopy
}

func (tx *Transaction) Verify(prevTXs map[string]Transaction) bool {
	if tx.IsCoinbase() {
		return true
	}

	for _, in := range tx.Inputs {
		if prevTXs[hex.EncodeToString(in.ID)].ID == nil {
			log.Panic("Previous transaction not correct")
		}
	}

	txCopy := tx.TrimmedCopy()
	curve := elliptic.P256()

	for inId, in := range tx.Inputs {
		prevTx := prevTXs[hex.EncodeToString(in.ID)]
		txCopy.Inputs[inId].Signature = nil
		txCopy.Inputs[inId].PubKey = prevTx.Outputs[in.Out].PubKeyHash
		txCopy.ID = txCopy.Hash()
		txCopy.Inputs[inId].PubKey = nil

		r := big.Int{}
		s := big.Int{}

		sigLen := len(in.Signature)
		r.SetBytes(in.Signature[:(sigLen / 2)])
		s.SetBytes(in.Signature[(sigLen / 2):])

		x := big.Int{}
		y := big.Int{}
		keyLen := len(in.PubKey)
		x.SetBytes(in.PubKey[:(keyLen / 2)])
		y.SetBytes(in.PubKey[(keyLen / 2):])

		rawPubKey := ecdsa.PublicKey{curve, &x, &y}
		if ecdsa.Verify(&rawPubKey, txCopy.ID, &r, &s) == false {
			return false
		}
	}

	return true
}

func (tx Transaction) String() string {
	var lines []string

	lines = append(lines, fmt.Sprintf("--- Transaction %x:", tx.ID))
	for i, input := range tx.Inputs {
		lines = append(lines, fmt.Sprintf("     Input %d:", i))
		lines = append(lines, fmt.Sprintf("       TXID:     %x", input.ID))
		lines = append(lines, fmt.Sprintf("       Out:       %d", input.Out))
		lines = append(lines, fmt.Sprintf("       Signature: %x", input.Signature))
		lines = append(lines, fmt.Sprintf("       PubKey:    %x", input.PubKey))
	}

	for i, output := range tx.Outputs {
		lines = append(lines, fmt.Sprintf("     Output %d:", i))
		lines = append(lines, fmt.Sprintf("       Value:  %d", output.Value))
		lines = append(lines, fmt.Sprintf("       Script: %x", output.PubKeyHash))
	}

	return strings.Join(lines, "\n")
}
