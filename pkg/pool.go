package goxash3d_fwgs

import (
	"errors"
	"runtime"
	"sync/atomic"
)

var (
	ErrPoolFull  = errors.New("pool is full")
	ErrPoolEmpty = errors.New("pool is empty")
)

// slot is one cell in the ring. seq is the Vyukov sequence number.
type slot struct {
	seq  uint32 // sequence number for vyukov algorithm
	val  uint8  // stored byte index
	_pad [3]byte
}

// BytesPool holds the bounded lock-free pool of byte indexes.
type BytesPool struct {
	mask uint32
	_0   [56]byte // pad to avoid false sharing with other vars

	head uint32 // enqueue position (producers)
	_1   [60]byte

	tail uint32 // dequeue position (consumers)
	_2   [60]byte

	slots []slot
}

// NewBytesPool creates a new BytesPool with capacity `n`. n must be a power of two and <= 256.
// The pool will be empty initially; call Prefill or Put to add indexes.
func NewBytesPool(n int) *BytesPool {
	if n <= 0 || n > 256 || (n&(n-1)) != 0 {
		panic("indexpool: capacity must be power of two and 1..256")
	}
	slots := make([]slot, n)
	for i := 0; i < n; i++ {
		// initialize seq to i so that slot is "available" to producers
		slots[i].seq = uint32(i)
	}
	return &BytesPool{
		mask:  uint32(n - 1),
		slots: slots,
	}
}

// Prefill fills the pool with the bytes 0..count-1 in order (useful to populate
// free index pool). count must be <= capacity. This is done with simple puts
// and may be used during initialization before heavy concurrency begins.
func (p *BytesPool) Prefill(count int) {
	if count < 0 || count > len(p.slots) {
		panic("indexpool: Prefill count out of range")
	}
	for i := 0; i < count; i++ {
		// busy-wait until Put succeeds (only used during init, so OK).
		for {
			if err := p.TryPut(uint8(i)); err == nil {
				break
			}
			runtime.Gosched()
		}
	}
}

// TryPut attempts to return an index (byte) back into the pool.
// On success returns nil. If the pool is full returns ErrPoolFull.
func (p *BytesPool) TryPut(v uint8) error {
	var spin int
	for {
		head := atomic.LoadUint32(&p.head)
		pos := head & p.mask
		s := &p.slots[pos]
		seq := atomic.LoadUint32(&s.seq)
		// seq == head means slot is free for producer
		if seq == head {
			// attempt to claim the head
			if atomic.CompareAndSwapUint32(&p.head, head, head+1) {
				// we own the slot; publish the value and advance its sequence
				s.val = v
				// store with release semantics: set seq to head+1
				atomic.StoreUint32(&s.seq, head+1)
				return nil
			}
			// CAS failed - retry
			continue
		}
		// If seq < head: slot has not yet been consumed (full)
		// If seq > head: some other producer is ahead
		if seq < head {
			// full
			return ErrPoolFull
		}
		// Otherwise, spin/yield briefly and retry
		spin++
		if spin&63 == 0 {
			runtime.Gosched()
		}
	}
}

// TryGet attempts to retrieve an index from the pool. On success returns the byte
// and nil error. If pool is empty returns ErrPoolEmpty.
func (p *BytesPool) TryGet() (uint8, error) {
	var spin int
	for {
		tail := atomic.LoadUint32(&p.tail)
		pos := tail & p.mask
		s := &p.slots[pos]
		seq := atomic.LoadUint32(&s.seq)
		// seq == tail+1 means slot contains data for consumer
		if seq == tail+1 {
			// attempt to claim the tail
			if atomic.CompareAndSwapUint32(&p.tail, tail, tail+1) {
				// we own the slot; read the value and mark slot as free
				v := s.val
				// set seq to tail + mask + 1 (which equals tail + capacity)
				atomic.StoreUint32(&s.seq, tail+uint32(len(p.slots)))
				return v, nil
			}
			// CAS failed - retry
			continue
		}
		// If seq <= tail: empty
		if seq <= tail {
			return 0, ErrPoolEmpty
		}
		// Otherwise, some other consumer is ahead; spin/yield
		spin++
		if spin&63 == 0 {
			runtime.Gosched()
		}
	}
}

// Capacity returns the configured capacity of the pool.
func (p *BytesPool) Capacity() int { return len(p.slots) }
