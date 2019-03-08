package dpos

import (
	"errors"
	"fmt"

	"github.com/vapor/protocol/bc"
	"github.com/vapor/protocol/bc/types"
)

func (bft *BftManager) makePrePrepareMsg(block *types.Block, round uint32) *types.PreprepareMsg {
	msg := &types.PreprepareMsg{
		Round: round,
		Block: block,
	}
	return msg
}

func (bft *BftManager) makePrepareMsg(prePreMsg *types.PreprepareMsg) (*types.PrepareMsg, error) {

	msg := &types.PrepareMsg{
		Round:       prePreMsg.Round,
		PrepareAddr: bft.coinBase,
		BlockHeight: prePreMsg.Block.Height,
		BlockHash:   prePreMsg.Block.Hash(),
		PrepareSig:  nil,
	}

	/*
		if sig, err := bft.dp.signFn(accounts.Account{Address: bft.coinBase}, msg.Hash().Bytes()); err != nil {
			log.Error("Make prepare msg failed", "error", err)
			return nil, fmt.Errorf("makePrepareMsg, error: %s", err)
		} else {
			msg.PrepareSig = make([]byte, len(sig))
			copy(msg.PrepareSig, sig)
			return msg, nil
		}
	*/
	return msg, nil
}

func (bft *BftManager) makeCommitMsg(prePreMsg *types.PreprepareMsg) (*types.CommitMsg, error) {
	msg := &types.CommitMsg{
		Round:       prePreMsg.Round,
		Commiter:    bft.coinBase,
		BlockHeight: prePreMsg.Block.Height,
		BlockHash:   prePreMsg.Block.Hash(),
		CommitSig:   nil,
	}
	/*
		if sig, err := bft.dp.signFn(accounts.Account{Address: bft.coinBase}, msg.Hash().Bytes()); err != nil {
			log.Error("Make commit msg failed", "error", err)
			return nil, fmt.Errorf("makeCommitMsg, error: %s", err)
		} else {
			msg.CommitSig = make([]byte, len(sig))
			copy(msg.CommitSig, sig)
			return msg, nil
		}
	*/
	return msg, nil
}

func (bft *BftManager) verifyPrePrepareMsg(msg *types.PreprepareMsg) error {
	// Nothing to verify
	return nil
}

func (bft *BftManager) verifyPrepareMsg(msg *types.PrepareMsg) error {
	var emptyHash bc.Hash
	if msg.BlockHash == emptyHash {
		return fmt.Errorf("prepare msg's block hash is empty")
	}

	// Sender is witness
	if !bft.validWitness(msg.PrepareAddr) {
		return fmt.Errorf("prepare sender is not witness: %s", msg.PrepareAddr)
	}

	// Verify signature
	data, _ := msg.Hash()
	if !bft.verifySig(msg.PrepareAddr, data.Bytes(), msg.PrepareSig) {
		return fmt.Errorf("prepare smg signature is invalid")
	}
	return nil
}

func (bft *BftManager) verifyCommitMsg(msg *types.CommitMsg) error {
	var emptyHash bc.Hash
	if msg.BlockHash == emptyHash {
		return fmt.Errorf("commit msg's block hash is empty")
	}

	// Sender is witness
	if !bft.validWitness(msg.Commiter) {
		return fmt.Errorf("commiter is not witness: %s", msg.Commiter)
	}

	// Verify signature
	data, _ := msg.Hash()
	if !bft.verifySig(msg.Commiter, data.Bytes(), msg.CommitSig) {
		return fmt.Errorf("commiter signature is invalid")
	}

	return nil
}

func (bft *BftManager) VerifyCmtMsgOf(block *types.BlockHeader) error {
	cmtMsges := block.CmtMsges
	if uint64(len(cmtMsges)) < bft.quorum {
		return fmt.Errorf("too less commit msg, len = %d", len(cmtMsges))
	}

	// Build witness cache
	witCaches := make(map[string]struct{})
	Witnesses := []string{}
	//for _, wit := range block.Witnesses() {
	for _, wit := range Witnesses {
		witCaches[wit] = struct{}{}
	}

	// Check each commit msg
	for _, m := range cmtMsges {
		if block.Hash() != m.BlockHash {
			return errors.New("commit msg hash not match with block hash")
		}

		if _, ok := witCaches[m.Commiter]; !ok {
			return errors.New("committer is not a valid witness")
		}
		data, _ := m.Hash()
		if !bft.verifySig(m.Commiter, data.Bytes(), m.CommitSig) {
			return errors.New("commit msg's signature is error")
		}
	}

	return nil
}

func (bft *BftManager) verifySig(sender string, data []byte, sig []byte) bool {
	return true
}
