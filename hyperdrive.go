package hyperdrive

import (
	"time"

	"github.com/renproject/hyperdrive/v1/block"
	"github.com/renproject/hyperdrive/v1/consensus"
	"github.com/renproject/hyperdrive/v1/replica"
	"github.com/renproject/hyperdrive/v1/shard"
	"github.com/renproject/hyperdrive/v1/sig"
	"github.com/renproject/hyperdrive/v1/tx"
)

const NumHistoricalShards = 3
const NumTicksToTriggerTimeOut = 2

type Dispatcher struct {
	shard shard.Shard
}

func NewDispatcher(shard shard.Shard) replica.Dispatcher {
	return &Dispatcher{
		shard: shard,
	}
}

func (d *Dispatcher) Dispatch(action consensus.Action) {
	// TODO:
	// 1. Broadcast the action to the entire shard
}

type Hyperdrive interface {
	AcceptTick(t time.Time)
	AcceptPropose(shardHash sig.Hash, proposed block.Block)
	AcceptPreVote(shardHash sig.Hash, preVote block.SignedPreVote)
	AcceptPreCommit(shardHash sig.Hash, preCommit block.SignedPreCommit)
	AcceptShard(shard shard.Shard, blockchain block.Blockchain)
}

type hyperdrive struct {
	signer sig.Signer

	shards        map[sig.Hash]shard.Shard
	shardReplicas map[sig.Hash]replica.Replica
	shardHistory  []sig.Hash

	ticksPerShard    map[sig.Hash]int
	timeoutThreshold int
}

func New(signer sig.Signer, timeoutThreshold int) Hyperdrive {
	return &hyperdrive{
		signer: signer,

		shards:        map[sig.Hash]shard.Shard{},
		shardReplicas: map[sig.Hash]replica.Replica{},
		shardHistory:  []sig.Hash{},

		ticksPerShard:    map[sig.Hash]int{},
		timeoutThreshold: timeoutThreshold,
	}
}

func (hyperdrive *hyperdrive) AcceptTick(t time.Time) {
	// 1. Increment number of ticks seen by each shard
	for shardHash := range hyperdrive.shards {
		ticks := hyperdrive.ticksPerShard[shardHash]
		ticks++
		hyperdrive.ticksPerShard[shardHash] = ticks

		if hyperdrive.ticksPerShard[shardHash] > hyperdrive.timeoutThreshold {
			// 2. After a number of ticks send a TimedOut transition to the shard
		}
	}
}

func (hyperdrive *hyperdrive) AcceptPropose(shardHash sig.Hash, proposed block.Block) {
	// TODO:
	// 1. Verify the block is well-formed
	// 2. Verify the signatory of the block
}

func (hyperdrive *hyperdrive) AcceptPreVote(shardHash sig.Hash, preVote block.SignedPreVote) {
	// TODO:
	// 1. Verify the pre-vote is well-formed
	// 2. Verify the signatory of the pre-vote
}

func (hyperdrive *hyperdrive) AcceptPreCommit(shardHash sig.Hash, preCommit block.SignedPreCommit) {
	// TODO:
	// 1. Verify the pre-commit is well-formed
	// 2. Verify the signatory of the pre-commit
}

func (hyperdrive *hyperdrive) AcceptShard(shard shard.Shard, blockchain block.Blockchain) {
	r := replica.New(
		NewDispatcher(shard),
		hyperdrive.signer,
		tx.FIFOPool(),
		consensus.WaitForPropose(blockchain.Round(), blockchain.Height()),
		consensus.NewStateMachine(block.NewPolkaBuilder(), block.NewCommitBuilder(), shard.ConsensusThreshold()),
		consensus.NewTransitionBuffer(shard.Size()),
		blockchain,
		shard,
	)

	hyperdrive.shardReplicas[shard.Hash] = r
	hyperdrive.shardHistory = append(hyperdrive.shardHistory, shard.Hash)
	if len(hyperdrive.shardHistory) > NumHistoricalShards {
		delete(hyperdrive.shardReplicas, hyperdrive.shardHistory[0])
		hyperdrive.shardHistory = hyperdrive.shardHistory[1:]
	}

	r.Init()
}
