package dpos

import (
	"fmt"
	"math/big"
	"sync"
	"sync/atomic"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/vapor/protocol/bc/types"
)

// BFT step
const (
	newRound    = uint32(0)
	prePrePared = uint32(1)
	preparing   = uint32(2)
	prepared    = uint32(3)
	committing  = uint32(4)
	committed   = uint32(5)
	done        = uint32(6)
)

type BftManager struct {
	dp       *Dpos  // DPoS object
	quorum   uint64 // 2f+1
	coinBase string

	mp      *msgPool // message pool of all future round bft messages, and not verified
	roundMp *msgPool // message pool of current round, and been verified

	// BFT state
	h           uint64              // local block chain height, protect by newRoundRWLock
	r           uint32              // local BFT round, protect by newRoundRWLock
	step        uint32              // local BFT round, protect by atomic operation
	witnessList map[string]struct{} // current witness list, rely on mining

	newRoundRWLock sync.RWMutex // RW lock for switch to new round

	blockRound uint32 // round of sealing block, no need lock
	mining     uint32 // mining or not, atomic read and write

	// callbacks
	sendBftMsg   func(types.ConsensusMsg)
	verifyBlock  func(*types.Block) error
	processBlock func(*types.Block) error
}

func newBftManager(dp *Dpos) *BftManager {
	n := dp.config.MaxSignerCount
	q := n - (n-1)/3 // N-f
	return &BftManager{
		dp:          dp,
		quorum:      q,
		mp:          newMsgPool(q, "msg pool"),
		roundMp:     newMsgPool(q, "round msg pool"),
		h:           0,
		r:           0,
		step:        newRound,
		witnessList: make(map[string]struct{}, dp.config.MaxSignerCount),
		mining:      0,
	}
}

// startPrePrepare will send pre-prepare msg and prepare msg
func (b *BftManager) startPrePrepare(block *types.Block) {
	log.Debug("Start PrePrepare")
	prePreMsg := b.makePrePrepareMsg(block, b.blockRound)

	// This node is a witness, which can seal block, no need check again
	b.sendBftMsg(prePreMsg)

	// startPrePrepare may be execute before newRound, so calling handleBftMsg to check round
	go b.handleBftMsg(prePreMsg)
}

func (b *BftManager) handleBftMsg(msg types.ConsensusMsg) error {

	msgBlkHeight, msgType, msgRound := msg.GetBlockHeight(), msg.Type(), msg.GetRound()
	msgHash, _ := msg.Hash()

	// Read lock for critical section
	b.newRoundRWLock.RLock()
	defer func() {
		b.newRoundRWLock.RUnlock()
		log.Debug("handle msg end", "hash", msgHash)
	}()
	if atomic.LoadUint32(&b.mining) == 0 {
		b.mp.addMsg(msg)
		log.Debug("HandleBftMsg: return for not mining")
		return nil
	}

	if msgBlkHeight > b.h {

		if err := b.mp.addMsg(msg); err != nil {
			log.Error("add prepare msg to msg pool error", "err", err)
			return err
		}
		if prePrepareMsg, ok := msg.(*types.PreprepareMsg); ok {
			go b.startSync(prePrepareMsg.Block)
		}
		return nil

	} else if msgBlkHeight < b.h {
		return fmt.Errorf("the height of msg is lower than current height, msg height :%d, current height : %d", msgBlkHeight, b.h)
	}

	if msgRound < b.r {
		return fmt.Errorf("the round of msg is lower than current round, msg round :%d, current round : %d", msgRound, b.r)
	} else if msgRound > b.r {

		if err := b.mp.addMsg(msg); err != nil {
			log.Error("add msg to msg pool error", "err", err)
			return err
		}

		return nil
	}

	switch msgType {
	case types.BftPreprepareMessage:
		return b.handlePrePrepareMsg(msg.(*types.PreprepareMsg))
	case types.BftPrepareMessage:
		return b.handlePrepareMsg(msg.(*types.PrepareMsg))
	case types.BftCommitMessage:
		return b.handleCommitMsg(msg.(*types.CommitMsg))
	default:
		log.Error("unknown bft message", "type", msgType.String())
		return fmt.Errorf("unknown bft message type: %s", msgType.String())
	}
}

// handlePrePrepareMsg
// WARN: msg pool has only one pre-prepare msg position, only the first correct one can be
// saved to msg pool. No need to care about you will vote for two pre-prepare msg. You only
// vote for the pre-prepare msg in msg pool.
func (b *BftManager) handlePrePrepareMsg(msg *types.PreprepareMsg) error {
	stp := atomic.LoadUint32(&b.step)
	if stp != newRound {
		log.Debug("Pre-prepare msg bft step not match", "local round", stp)
		return nil
	}
	// Verify msg
	if err := b.verifyPrePrepareMsg(msg); err != nil {
		log.Errorf("Pre-prepare msg is invalid: %v", err)
		return err
	}

	if err := b.verifyBlock(msg.Block); err != nil {
		log.Errorf("Pre-prepare's block is invalid: %v", err)
		return nil
	}

	// Add msg to round msg pool, instead of msg pool
	if err := b.roundMp.addMsg(msg); err != nil {
		log.Warn("Add pre-prepare msg failed", "error", err)
		return err
	}

	// Go to next step
	if ok := atomic.CompareAndSwapUint32(&b.step, newRound, prePrePared); ok {
		return b.startPrepare()
	}

	return nil
}

// startPrepare enter prepare step, whether
func (b *BftManager) startPrepare() error {
	log.Debug("Start Prepare")
	// check our state and make a prepare msg
	if atomic.LoadUint32(&b.step) != prePrePared {
		return nil
	}
	prePreMsg, err := b.roundMp.getPrePrepareMsg(b.h, b.r)
	if err != nil {
		log.Error("Get pre-prepare msg from msg pool in prepare", "error", err)
		return fmt.Errorf("get pre-prepare msg from msg pool, error: %s", err)
	}

	preMsg, err := b.makePrepareMsg(prePreMsg)
	if err != nil {
		return err
	}

	if err := b.roundMp.addMsg(preMsg); err != nil {
		return err
	}

	// The one of first changing step, send the msg
	if ok := atomic.CompareAndSwapUint32(&b.step, prePrePared, preparing); ok {
		b.sendMsg(preMsg)
	}

	return b.tryCommitStep()
}

// tryCommitStep check whether can enter commit step
func (b *BftManager) tryCommitStep() error {
	stp := atomic.LoadUint32(&b.step)
	if stp < preparing || stp > prepared {
		log.Debug("tryCommitStep step not match", "step", stp)
		return nil
	}

	var (
		prePreMsg   *types.PreprepareMsg
		prepareMsgs []*types.PrepareMsg
		err         error
	)
	if prepareMsgs, err = b.roundMp.getTwoThirdMajorityPrepareMsg(b.h, b.r); err != nil {
		return nil
	}
	if prePreMsg, err = b.roundMp.getPrePrepareMsg(b.h, b.r); err != nil {
		log.Error("Get pre-prepare msg from msg pool in try commit", "error", err)
		return fmt.Errorf("get pre-prepare msg from msg pool, error: %s", err)
	}
	if prePreMsg.Block.Hash() != prepareMsgs[0].BlockHash {
		log.Error("Majority prepare msg is not match with pre-prepare msg", "block in prepare",
			prepareMsgs[0].BlockHash, "block in pre-prepare", prePreMsg.Block.Hash())
		return fmt.Errorf("majority prepare msg is not match with pre-prepare msg")
	}
	atomic.CompareAndSwapUint32(&b.step, preparing, prepared)
	return b.startCommit(prePreMsg)

}

// startCommit build commit message and send it
func (b *BftManager) startCommit(prePreMsg *types.PreprepareMsg) error {
	log.Debug("Start commit")
	if atomic.LoadUint32(&b.step) != prepared {
		return nil
	}

	commitMsg, err := b.makeCommitMsg(prePreMsg)
	if err != nil {
		return err
	}
	b.roundMp.addMsg(commitMsg)

	// The one of first changing step, send the msg
	if ok := atomic.CompareAndSwapUint32(&b.step, prepared, committing); ok {
		b.sendMsg(commitMsg)
	}

	return b.tryWriteBlockStep()
}

// sendMsg only witness send bft message.
// Caller make sure has the newRoundRWLock.
func (b *BftManager) sendMsg(msg types.ConsensusMsg) {
	log.Debug("bft sendMsg start")
	if _, ok := b.witnessList[b.coinBase]; ok {
		b.sendBftMsg(msg)
	}
	log.Debug("bft sendMsg success")
}

func (b *BftManager) tryWriteBlockStep() error {
	if atomic.LoadUint32(&b.step) != committing {
		return nil
	}
	// commit消息满足数量要求后写区块
	if commitMsgs, err := b.roundMp.getTwoThirdMajorityCommitMsg(b.h, b.r); err != nil {
		return nil
	} else if ok := atomic.CompareAndSwapUint32(&b.step, committing, committed); ok {
		if prePrepareMsg, err := b.roundMp.getPrePrepareMsg(b.h, b.r); err != nil {
			log.Error("Get pre-prepare msg from msg pool in try write block", "err", err)
			return fmt.Errorf("get pre-prepare msg from msg pool, error: %s", err)
		} else {
			if err := b.processBlockWithSig(prePrepareMsg, commitMsgs); err != nil {
				log.Error("Write block to chain", "err", err)
				return fmt.Errorf("write block to chain error: %s", err)
			}
			atomic.CompareAndSwapUint32(&b.step, committed, done)
		}
	}
	return nil
}

func (b *BftManager) handlePrepareMsg(msg *types.PrepareMsg) error {
	stp := atomic.LoadUint32(&b.step)
	// 当前阶段大于preparing，则prepare消息已经没用了，直接舍弃
	if stp > preparing {
		log.Debug("prepare msg bft step not match", "local step", stp)
		return nil
	}

	if err := b.verifyPrepareMsg(msg); err != nil {
		log.Error("failed to verify prepare msg", "err", err)
		return err
	}
	if err := b.roundMp.addMsg(msg); err != nil {
		log.Error("failed to add prepare msg", "height", b.h, "round", b.r, "err", err)
		return err
	}

	return b.tryCommitStep()
}

func (b *BftManager) handleCommitMsg(msg *types.CommitMsg) error {
	stp := atomic.LoadUint32(&b.step)
	// 当前阶段大于committing，则commit消息已经没用了，直接舍弃
	if stp > committing {
		log.Debug("commit msg bft step not match", "local step", stp)
		return nil
	}
	if err := b.verifyCommitMsg(msg); err != nil {
		log.Error("failed to verify commit msg", "err", err)
		return err
	}
	if err := b.roundMp.addMsg(msg); err != nil {
		log.Error("failed to add commit msg", "height", b.h, "round", b.r, "err", err)
		return err
	}

	return b.tryWriteBlockStep()
}

// writeBlock to block chain
func (b *BftManager) processBlockWithSig(msg *types.PreprepareMsg, cmtMsg []*types.CommitMsg) error {
	block := msg.Block
	h := block.Hash()
	// Match pre-prepare msg and commit msg
	if block.Hash() != cmtMsg[0].BlockHash {
		return fmt.Errorf("writeBlockWithSig error, commit msg for block: %s, not for block: %s", cmtMsg[0].BlockHash.String(), h.String())
	}

	block.FillBftMsg(cmtMsg)

	log.Info("writeBlockWithSig", "h", b.h, "r", b.r, "hash", h.String())
	return b.processBlock(block)
}

// newRound has lock, it maybe time consuming at sometime, call it by routine
func (b *BftManager) newRound(h uint64, r uint32, witList []string) {
	log.Debug("New round switch start")
	b.newRoundRWLock.Lock()

	// Update witness list
	if b.h != h {
		b.witnessList = make(map[string]struct{})
		for _, wit := range witList {
			b.witnessList[wit] = struct{}{}
		}
	}

	// Reset state and round msg pool
	b.h = h
	b.r = r
	b.step = newRound

	// Reset round msg pool
	b.roundMp.cleanAllMessage()

	// Switch to new round finished
	b.newRoundRWLock.Unlock()

	log.Debug("New round switch finish", "h", b.h, "r", b.r, "time", time.Now().Unix())

	// New round switch finished, must return right now
	go b.importCurRoundMsg()
}

// importCurRoundMsg import consensus messages, but can not directly import to round msg pool
func (b *BftManager) importCurRoundMsg() {
	b.newRoundRWLock.RLock()
	msg := b.mp.getAllMsgOf(b.h, b.r)
	b.newRoundRWLock.RUnlock()
	for _, m := range msg {
		h, _ := m.Hash()
		log.Debug("Import Msg", "type", m.Type(), "hash", h)
		go b.handleBftMsg(m)
	}
}

func (b *BftManager) startSync(block *types.Block) {
	// 	TODO Not emergency, sync will be triggered by block hash msg
	log.Debug("Bft manager startPrePrepare sync")
}

func (b *BftManager) miningStop() {
	atomic.StoreUint32(&b.mining, 0)
	log.Debug("BFT stop mining")
}

func (b *BftManager) miningStart() {
	atomic.StoreUint32(&b.mining, 1)
	log.Debug("BFT start mining")
}

func (b *BftManager) validWitness(wit string) bool {
	_, ok := b.witnessList[wit]
	return ok
}

// cleanOldMsg clean msg pool and keep future message. cleaning only
// on height % 100 == 0.
func (b *BftManager) cleanOldMsg(h *big.Int) {
	b.mp.cleanOldMessage(h)
}