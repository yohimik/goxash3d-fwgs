package goxash3d_fwgs

import (
	"errors"
	"runtime"
	"sync/atomic"
	"time"
)

// -----------------------------
// PacketQueue: Vyukov-style bounded MPSC storing Packet directly
// - i386-safe: uses uint32 atomics
// - cacheline padding to reduce false sharing
// - adaptive spin/backoff instead of unconditional Gosched
// - optional clearing of slot on dequeue to avoid keeping references
// -----------------------------

const cacheLine = 64

type pktCell struct {
	// seq is a ticket-style sequence number.
	seq uint32
	// pad so val doesn't share cacheline with seq on some archs.
	_pad0 [cacheLine - 4]byte
	val   Packet
	// pad to avoid neighboring cell sharing (cells are in a slice though).
	_pad1 [0]byte
}

// PacketQueue is a bounded MPSC queue of Packet. Capacity must be power of two.
type PacketQueue struct {
	mask uint32
	buf  []pktCell

	// pad before head to separate cache lines
	_headPad [cacheLine]byte
	head     uint32 // consumer only

	// pad between head and tail
	_midPad [cacheLine - 4]byte

	_tailPad [cacheLine]byte
	tail     uint32 // producers use atomic.AddUint32
	// trailing pad
	_tailPad2 [cacheLine - 4]byte
}

func NewPacketQueue(capacity int) *PacketQueue {
	if capacity < 2 || (capacity&(capacity-1)) != 0 {
		panic("NewPacketQueue: capacity must be power of two and >=2")
	}
	q := &PacketQueue{
		mask: uint32(capacity - 1),
		buf:  make([]pktCell, capacity),
	}
	for i := range q.buf {
		q.buf[i].seq = uint32(i)
	}
	return q
}

var ErrPacketQueueFull = errors.New("packetqueue: full")

// smallSpinLoop: do a few iterations of "pause" (busy) before yielding.
// On Go we don't have PAUSE instruction accessportably, so we just loop.
func smallSpinLoop(iter int) {
	for i := 0; i < iter; i++ {
		// trivial no-op to keep CPU busy for a few cycles.
		// compiler will not optimize this empty loop away because it's observable by time.
	}
}

// Enqueue pushes a Packet into the queue (non-blocking). Multiple producers safe.
func (q *PacketQueue) Enqueue(p Packet) error {
	// reserve a slot (fetch-and-increment)
	pos := atomic.AddUint32(&q.tail, 1) - 1
	mask := q.mask
	cell := &q.buf[pos&mask]

	// spin / backoff
	spin := 0
	for {
		seq := atomic.LoadUint32(&cell.seq)
		if seq == pos {
			// slot free for us
			// store payload
			cell.val = p
			// publish: make visible to consumer
			atomic.StoreUint32(&cell.seq, pos+1)
			return nil
		}
		// seq < pos  => slot still not consumed (full)
		// seq > pos  => another producer raced ahead and already took this slot (shouldn't happen because pos unique)
		if seq < pos {
			// queue full
			return ErrPacketQueueFull
		}
		// backoff strategy: a few busy loops then yield
		spin++
		if spin < 8 {
			smallSpinLoop(10 << uint(spin)) // grow the busy loop a bit
		} else if (spin & 15) == 0 {
			// less frequent OS scheduler yield
			runtime.Gosched()
		} else {
			// brief sleep to avoid burning CPU in extreme contention (very rare for MPSC)
			time.Sleep(time.Microsecond)
		}
	}
}

// TryDequeue pops a Packet in consumer (single) context. Returns (pkt,true) or (zero,false).
func (q *PacketQueue) TryDequeue() (Packet, bool) {
	pos := q.head
	cell := &q.buf[pos&q.mask]
	seq := atomic.LoadUint32(&cell.seq)
	if seq == pos+1 {
		// read payload
		val := cell.val

		// Optional: clear the slot's value to avoid keeping references to underlying memory.
		// This can help GC for long-lived queues when packets hold large buffers.
		// Clear before releasing the slot to ensure producer does not see old references.
		cell.val = Packet{}

		// mark slot free for producers: seq = pos + mask + 1
		atomic.StoreUint32(&cell.seq, pos+q.mask+1)
		q.head = pos + 1
		return val, true
	}
	var zero Packet
	return zero, false
}

// DrainPackets processes available packets with handler and returns count.
func (q *PacketQueue) DrainPackets(handler func(Packet)) int {
	count := 0
	for {
		pkt, ok := q.TryDequeue()
		if !ok {
			break
		}
		handler(pkt)
		count++
	}
	return count
}

// Len returns an approximate count of items in the queue. Only approximate during concurrent Enqueue.
func (q *PacketQueue) Len() int {
	t := atomic.LoadUint32(&q.tail)
	h := q.head // consumer-only read
	if t < h {
		return 0
	}
	return int(t - h)
}
