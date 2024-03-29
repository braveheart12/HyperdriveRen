package state_test

import (
	"math/rand"
	"testing/quick"
	"time"

	"github.com/renproject/hyperdrive/block"
	"github.com/renproject/hyperdrive/sig"
	"github.com/renproject/hyperdrive/state"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/renproject/hyperdrive/state"
)

var conf = quick.Config{
	MaxCount:      256,
	MaxCountScale: 0,
	Rand:          nil,
	Values:        nil,
}

var _ = Describe("TransitionBuffer", func() {

	Context("when using Ticked", func() {
		It("should implement the State interface", func() {
			state.Ticked{}.IsTransition()
			Expect(state.Ticked{}.Round()).To(Equal(block.Round(-1)))
			Expect(state.Ticked{}.Signer()).To(Equal(sig.Signatory{}))
		})
	})

	Context("when using Proposed", func() {
		It("should implement the State interface", func() {
			state.Proposed{}.IsTransition()
			Expect(state.Proposed{}.Round()).To(Equal(block.Round(0)))
			Expect(state.Proposed{}.Signer()).To(Equal(sig.Signatory{}))
		})
	})

	Context("when using PreVoted", func() {
		It("should implement the State interface", func() {
			state.PreVoted{}.IsTransition()
			Expect(state.PreVoted{}.Round()).To(Equal(block.Round(0)))
			Expect(state.PreVoted{}.Signer()).To(Equal(sig.Signatory{}))
		})
	})

	Context("when using PreCommitted", func() {
		It("should implement the State interface", func() {
			state.PreCommitted{}.IsTransition()
			Expect(state.PreCommitted{}.Round()).To(Equal(block.Round(0)))
			Expect(state.PreCommitted{}.Signer()).To(Equal(sig.Signatory{}))
		})
	})

	Context("when only propose transitions are enqueued", func() {
		It("should enqueue the same number of times as it dequeues for a given height", func() {
			test := func(num uint8, incrementHeight uint8) bool {
				tb := NewTransitionBuffer(20)
				// cannot do (x % 0)
				if incrementHeight == 0 {
					incrementHeight++
				}

				genesisBlock := block.Genesis()
				genesis := Proposed{SignedPropose: block.SignedPropose{Propose: block.Propose{Block: genesisBlock}}}
				var height block.Height
				height = 0
				scratch := make(map[block.Height]uint8)

				for i := uint8(0); i < num; i++ {
					if i%incrementHeight == 0 {
						genesis.Block.Height++
						height++
					}
					tb.Enqueue(genesis)
					if _, ok := scratch[height]; !ok {
						scratch[height] = 0
					}
					scratch[height]++
				}

				for k, v := range scratch {
					for i := uint8(0); i < v; i++ {
						_, ok := tb.Dequeue(k)
						Expect(ok).To(Equal(true),
							"Dequeue was empty! should have %v, had %v",
							v, i)
					}
					_, ok := tb.Dequeue(k)
					Expect(ok).To(Equal(false),
						"Dequeue was Not empty when it shouldn't")
				}
				return true
			}
			Expect(quick.Check(test, &conf)).ShouldNot(HaveOccurred())
		})
	})

	Context("when random transitions are enqueued", func() {
		It("should dequeue the most relevant transition", func() {
			test := func(size int, numInputs uint8) bool {
				// size cannot be negative
				if size < 0 {
					size *= -1
				}
				size = size % 100
				tb := NewTransitionBuffer(int(size % 67))

				mock := newMock()

				for i := uint8(0); i < numInputs; i++ {
					tb.Enqueue(mock.nextTransition())
				}

				for height, mockTran := range mock.Map {
					tran, ok := tb.Dequeue(height)
					Expect(ok).To(Equal(true))
					switch tranType := tran.(type) {
					case Proposed:
						Expect(mockTran).To(Equal(mockProposed),
							"expected %v, got: %T", show(mockTran), tranType)
					default:
						Expect(false).To(Equal(true),
							"unexpected Transition type: %T FIXME!", tranType)
					}
				}
				return true
			}
			Expect(quick.Check(test, &conf)).ShouldNot(HaveOccurred())
		})
	})

	Context("when dropping transitions", func() {
		It("should remove everything below the dropped height", func() {
			tb := NewTransitionBuffer(5)

			proposed := Proposed{}
			proposed.Block.Height = 0
			tb.Enqueue(proposed)
			tb.Enqueue(proposed)
			proposed.Block.Height = 1
			tb.Enqueue(proposed)
			tb.Drop(1)
			tran, ok := tb.Dequeue(0)
			Expect(ok).To(Equal(false), "dequeued type %T", tran)
			_, ok = tb.Dequeue(1)
			Expect(ok).To(Equal(true))
		})
	})

	Context("when unsupported transitions are added", func() {
		It("should panic", func() {
			tb := NewTransitionBuffer(5)
			Expect(func() { tb.Enqueue(PreCommitted{}) }).To(Panic())
		})
	})
})

type mockInput struct {
	height block.Height
	rnd    *rand.Rand
	// This keeps track of the Transition the TransitionBuffer at the
	// given Height should return with Dequeue
	Map          map[block.Height]mockTran
	GotImmediate bool
}

type mockTran uint8

const (
	mockProposed mockTran = iota
)

func show(tran mockTran) string {
	switch tran {
	case mockProposed:
		return "Proposed"
	default:
		return "FIXME, Not a mockTran"
	}
}

func newMock() *mockInput {
	return &mockInput{
		height:       0,
		rnd:          rand.New(rand.NewSource(time.Now().UnixNano())),
		Map:          make(map[block.Height]mockTran),
		GotImmediate: false,
	}
}

func (m *mockInput) nextTransition() Transition {
	var rndTransition Transition

	// maybe increase height
	if m.rnd.Intn(6) == 1 {
		m.height = m.height + block.Height(m.rnd.Intn(4))
	}
	// maybe decrease height
	if m.rnd.Intn(6) == 1 {
		tmp := m.height - block.Height(m.rnd.Intn(4))
		if tmp > 0 {
			m.height = tmp
		}
	}

	// pick Transition
	nextTran := mockTran(m.rnd.Intn(1))

	switch nextTran {
	case mockProposed:
		genesisBlock := block.Genesis()
		gen := Proposed{SignedPropose: block.SignedPropose{Propose: block.Propose{Block: genesisBlock}}}
		gen.Block.Height = m.height
		rndTransition = gen
		// The only time I would ever Dequeue a Propose is if either
		// there already was a Propose or nothing else in the queue at
		// that height
		if _, ok := m.Map[m.height]; !ok {
			m.Map[m.height] = mockProposed
		}
	}

	return rndTransition
}
