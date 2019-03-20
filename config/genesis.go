package config

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"

	log "github.com/sirupsen/logrus"

	"github.com/vapor/consensus"
	"github.com/vapor/crypto/ed25519"
	"github.com/vapor/protocol/bc"
	"github.com/vapor/protocol/bc/types"
	"github.com/vapor/protocol/vm/vmutil"
)

func commitToArguments() (res *[32]byte) {
	var fedpegPubkeys []ed25519.PublicKey
	for _, xpub := range consensus.ActiveNetParams.FedpegXPubs {
		fedpegPubkeys = append(fedpegPubkeys, xpub.PublicKey())
	}
	fedpegScript, _ := vmutil.P2SPMultiSigProgram(fedpegPubkeys, len(fedpegPubkeys))

	var buffer bytes.Buffer
	for _, address := range CommonConfig.Consensus.Signers {
		redeemContract := address.ScriptAddress()
		buffer.Write(redeemContract)
	}

	hasher := sha256.New()
	hasher.Write(fedpegScript)
	hasher.Write(buffer.Bytes())
	resSlice := hasher.Sum(nil)
	res = new([32]byte)
	copy(res[:], resSlice)
	return
}

func genesisTx() *types.Tx {

	contract, err := hex.DecodeString("00148c9d063ff74ee6d9ffa88d83aeb038068366c4c4")
	if err != nil {
		log.Panicf("fail on decode genesis tx output control program")
	}

	coinbaseInput := commitToArguments()
	txData := types.TxData{
		Version: 1,
		Inputs: []*types.TxInput{
			// Any consensus-related values that are command-line set can be added here for anti-footgun
			types.NewCoinbaseInput(coinbaseInput[:]),
			//types.NewCoinbaseInput([]byte("Information is power. -- Jan/11/2013. Computing is power. -- Apr/24/2018.")),
		},
		Outputs: []*types.TxOutput{
			types.NewTxOutput(*consensus.BTMAssetID, consensus.InitialBlockSubsidy, contract),
		},
	}

	return types.NewTx(txData)
}

func mainNetGenesisBlock() *types.Block {
	tx := genesisTx()
	txStatus := bc.NewTransactionStatus()
	if err := txStatus.SetStatus(0, false); err != nil {
		log.Panicf(err.Error())
	}
	txStatusHash, err := types.TxStatusMerkleRoot(txStatus.VerifyStatus)
	if err != nil {
		log.Panicf("fail on calc genesis tx status merkle root")
	}

	merkleRoot, err := types.TxMerkleRoot([]*bc.Tx{tx.Tx})
	if err != nil {
		log.Panicf("fail on calc genesis tx merkel root")
	}

	block := &types.Block{
		BlockHeader: types.BlockHeader{
			Version:   1,
			Height:    0,
			Timestamp: 1524549600,
			BlockCommitment: types.BlockCommitment{
				TransactionsMerkleRoot: merkleRoot,
				TransactionStatusHash:  txStatusHash,
			},
		},
		Transactions: []*types.Tx{tx},
	}
	return block
}

func testNetGenesisBlock() *types.Block {
	tx := genesisTx()
	txStatus := bc.NewTransactionStatus()
	if err := txStatus.SetStatus(0, false); err != nil {
		log.Panicf(err.Error())
	}
	txStatusHash, err := types.TxStatusMerkleRoot(txStatus.VerifyStatus)
	if err != nil {
		log.Panicf("fail on calc genesis tx status merkle root")
	}

	merkleRoot, err := types.TxMerkleRoot([]*bc.Tx{tx.Tx})
	if err != nil {
		log.Panicf("fail on calc genesis tx merkel root")
	}
	block := &types.Block{
		BlockHeader: types.BlockHeader{
			Version:   1,
			Height:    0,
			Timestamp: 1528945000,
			BlockCommitment: types.BlockCommitment{
				TransactionsMerkleRoot: merkleRoot,
				TransactionStatusHash:  txStatusHash,
			},
		},
		Transactions: []*types.Tx{tx},
	}
	return block
}

func soloNetGenesisBlock() *types.Block {
	tx := genesisTx()
	txStatus := bc.NewTransactionStatus()
	if err := txStatus.SetStatus(0, false); err != nil {
		log.Panicf(err.Error())
	}
	txStatusHash, err := types.TxStatusMerkleRoot(txStatus.VerifyStatus)
	if err != nil {
		log.Panicf("fail on calc genesis tx status merkle root")
	}

	merkleRoot, err := types.TxMerkleRoot([]*bc.Tx{tx.Tx})
	if err != nil {
		log.Panicf("fail on calc genesis tx merkel root")
	}

	block := &types.Block{
		BlockHeader: types.BlockHeader{
			Version:   1,
			Height:    0,
			Timestamp: CommonConfig.Consensus.GenesisTimestamp,
			BlockCommitment: types.BlockCommitment{
				TransactionsMerkleRoot: merkleRoot,
				TransactionStatusHash:  txStatusHash,
			},
		},
		Transactions: []*types.Tx{tx},
	}
	return block
}

// GenesisBlock will return genesis block
func GenesisBlock() *types.Block {
	return map[string]func() *types.Block{
		"main": mainNetGenesisBlock,
		"test": testNetGenesisBlock,
		"solo": soloNetGenesisBlock,
	}[consensus.ActiveNetParams.Name]()
}
