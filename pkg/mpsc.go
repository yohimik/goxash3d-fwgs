package goxash3d_fwgs

import (
	"errors"
	"runtime"
	"sync/atomic"
	"time"
	"unsafe"
)

var (
	ErrFull   = errors.New("mpscqueue: full")
	ErrClosed = errors.New("mpscqueue: closed")
)

// cacheLinePad to avoid false sharing on 32-bit (64 bytes).
type cacheLinePad [16]uint32

// cell is one ring slot. seq is a uint32 sequence number used by Vyukov algorithm.
// val is stored as an unsafe.Pointer to avoid interface{} atomicity and writer races.
// For convenience the public API accepts interface{} and converts to unsafe.Pointer.
// For the absolute fastest path, change `interface{}` handling to store concrete types
// (e.g., *Packet) and avoid boxing.
type cell struct {
	seq uint32
	// 32-bit aligned pointer on 386; using unsafe.Pointer so we can store any pointer-sized value.
	val unsafe.Pointer
}

// MPSCQueue is a bounded non-blocking MPSC queue.
type MPSCQueue struct {
	_headPad cacheLinePad
	head     uint32 // consumer index (only touched by consumer)

	_tailPad cacheLinePad
	tail     uint32 // producer index (atomic add)

	mask uint32
	buf  []cell

	closed uint32 // 0 = open, 1 = closed (atomic)
}

// NewMPSCQueue creates a new queue with power-of-two capacity >= 2.
func NewMPSCQueue(capacity int) *MPSCQueue {
	if capacity < 2 || (capacity&(capacity-1)) != 0 {
		panic("mpscqueue: capacity must be a power of two and >= 2")
	}
	q := &MPSCQueue{
		mask: uint32(capacity - 1),
		buf:  make([]cell, capacity),
	}
	for i := range q.buf {
		q.buf[i].seq = uint32(i)
	}
	return q
}

// Close marks queue closed. Producers will get ErrClosed on Enqueue attempts.
// Consumer can still drain remaining items.
func (q *MPSCQueue) Close() {
	atomic.StoreUint32(&q.closed, 1)
}

func (q *MPSCQueue) isClosed() bool {
	return atomic.LoadUint32(&q.closed) != 0
}

// helper to convert interface{} to unsafe.Pointer (nil safe)
func ifaceToPtr(v interface{}) unsafe.Pointer {
	if v == nil {
		return nil
	}
	return unsafe.Pointer(&v)
}

// helper to convert pointer stored back to interface{}.
// Note: because we store the temporary address of an interface value above we must
// reconstruct the original value. To avoid double-allocation in hot paths, replace
// queue storage with concrete pointers (e.g., *Packet) and cast directly.
func ptrToIface(p unsafe.Pointer) interface{} {
	if p == nil {
		return nil
	}
	// p points to a temporary interface value; read it.
	return *(*interface{})(p)
}

// Enqueue attempts a non-blocking push. Returns ErrFull if the queue is full, ErrClosed if closed.
// This is the ultra-fast path intended for hot producers; it never blocks or syscalls.
func (q *MPSCQueue) Enqueue(v interface{}) error {
	if q.isClosed() {
		return ErrClosed
	}
	// reserve slot
	pos := atomic.AddUint32(&q.tail, 1) - 1
	idx := pos & q.mask
	cell := &q.buf[idx]
	// spin once or twice waiting for slot to be available
	seq := atomic.LoadUint32(&cell.seq)
	// expected sequence for a free slot is pos
	if seq != pos {
		// fast-fail: determine queue full by comparing distance between tail and head
		// If seq < pos it's not ready. We attempt a bounded spin to avoid immediate failure
		// for transient contention.
		spins := 0
		for seq != pos {
			spins++
			if spins < 4 {
				runtime.Gosched()
				seq = atomic.LoadUint32(&cell.seq)
				continue
			}
			// final check: if seq < pos then slot hasn't been consumed -> full
			if seq < pos {
				return ErrFull
			}
			// otherwise reload and retry a few times
			seq = atomic.LoadUint32(&cell.seq)
		}
	}
	// slot owned; store value (as pointer) and publish by setting seq = pos+1
	ptr := ifaceToPtr(v)
	atomic.StorePointer(&cell.val, ptr)
	atomic.StoreUint32(&cell.seq, pos+1)
	return nil
}

// EnqueueSpin is a utility producers can use if they prefer waiting with backoff.
// It returns ErrClosed if queue closed while spinning.
func (q *MPSCQueue) EnqueueSpin(v interface{}, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for {
		err := q.Enqueue(v)
		if err == nil {
			return nil
		}
		if err == ErrClosed {
			return ErrClosed
		}
		// backoff strategy: yield a bit then retry until deadline
		runtime.Gosched()
		if time.Now().After(deadline) {
			return ErrFull
		}
	}
}

// TryDequeue is the non-blocking consumer pop. Returns (item, true) if success.
// If empty returns (nil, false). Consumer is expected to be single-threaded.
func (q *MPSCQueue) TryDequeue() (interface{}, bool) {
	pos := q.head
	idx := pos & q.mask
	cell := &q.buf[idx]
	seq := atomic.LoadUint32(&cell.seq)
	if seq == pos+1 {
		// slot ready
		p := atomic.LoadPointer(&cell.val)
		// clear val for GC and reuse
		atomic.StorePointer(&cell.val, nil)
		// publish slot as free: seq = pos + mask + 1
		atomic.StoreUint32(&cell.seq, pos+q.mask+1)
		q.head = pos + 1
		return ptrToIface(p), true
	}
	return nil, false
}

// Drain calls handler for each available item and returns number processed.
// This reduces per-item overhead by avoiding repeated Go-level loops in user code.
func (q *MPSCQueue) Drain(handler func(interface{})) int {
	count := 0
	for {
		item, ok := q.TryDequeue()
		if !ok {
			break
		}
		handler(item)
		count++
	}
	return count
}

// Len returns approximate number of items in queue.
func (q *MPSCQueue) Len() int {
	t := atomic.LoadUint32(&q.tail)
	h := atomic.LoadUint32(&q.head)
	return int(t - h)
}

// Cap returns capacity.
func (q *MPSCQueue) Cap() int { return len(q.buf) }
