package pkg

import (
	"crypto/sha256"
	"encoding/binary"
	"runtime"
	"sync"
	"sync/atomic"
)

const (
	Charset   = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789@"
	MaxDepth  = 8
	batchSize = 64
)

type FlatShard struct {
	tags   []uint16
	hashes [][32]byte
	words  [][MaxDepth]byte
	mask   uint32
	mu     sync.Mutex
}

type MapReferential struct {
	shards   *[256]FlatShard
	maxDepth int
}

func TotalCombinations(depth int) int {
	total := 0
	curr := 1
	n := len(Charset)
	for d := 1; d <= depth; d++ {
		curr *= n
		total += curr
	}
	return total
}

func nextPowerOf2(n uint32) uint32 {
	if n <= 1 {
		return 1
	}
	n--
	n |= n >> 1
	n |= n >> 2
	n |= n >> 4
	n |= n >> 8
	n |= n >> 16
	return n + 1
}

type batchItem struct {
	hash [32]byte
	word [MaxDepth]byte
}

type workerBatches struct {
	items [256][batchSize]batchItem
	lens  [256]uint8
}

func (wb *workerBatches) add(shards *[256]FlatShard, hash [32]byte, word [MaxDepth]byte) {
	s := hash[0]
	idx := wb.lens[s]
	wb.items[s][idx] = batchItem{hash: hash, word: word}
	idx++
	if idx == batchSize {
		shard := &shards[s]
		shard.mu.Lock()
		mask := shard.mask
		tags := shard.tags
		hashes := shard.hashes
		words := shard.words

		for i := range batchSize {
			item := &wb.items[s][i]
			h := item.hash
			tag := binary.LittleEndian.Uint16(h[1:3])
			if tag == 0 {
				tag = 1
			}
			slot := binary.LittleEndian.Uint32(h[3:7]) & mask
			for {
				t := tags[slot]
				if t == 0 || (t == tag && hashes[slot] == h) {
					tags[slot] = tag
					hashes[slot] = h
					words[slot] = item.word
					break
				}
				slot = (slot + 1) & mask
			}
		}
		shard.mu.Unlock()
		idx = 0
	}
	wb.lens[s] = idx
}

func (wb *workerBatches) flush(shards *[256]FlatShard) {
	for s := range 256 {
		n := wb.lens[s]
		if n > 0 {
			shard := &shards[s]
			shard.mu.Lock()
			mask := shard.mask
			tags := shard.tags
			hashes := shard.hashes
			words := shard.words

			for i := range n {
				item := &wb.items[s][i]
				h := item.hash
				tag := binary.LittleEndian.Uint16(h[1:3])
				if tag == 0 {
					tag = 1
				}
				slot := binary.LittleEndian.Uint32(h[3:7]) & mask
				for {
					t := tags[slot]
					if t == 0 || (t == tag && hashes[slot] == h) {
						tags[slot] = tag
						hashes[slot] = h
						words[slot] = item.word
						break
					}
					slot = (slot + 1) & mask
				}
			}
			shard.mu.Unlock()
			wb.lens[s] = 0
		}
	}
}

type task struct {
	prefix  [2]byte
	depth   int
	prefLen int
}

func NewMapReferential(depth int) MapReferential {
	if depth < 1 {
		depth = 0
	}
	precomputeDepth := min(depth, 4)

	var shards [256]FlatShard
	if precomputeDepth > 0 {
		totalComb := uint32(TotalCombinations(precomputeDepth))
		entriesPerShard := (totalComb / 256) + 1
		targetCap := uint32(float64(entriesPerShard) / 0.50)
		capPow2 := nextPowerOf2(targetCap)
		if capPow2 < 32 {
			capPow2 = 32
		}

		for i := range 256 {
			shards[i].tags = make([]uint16, capPow2)
			shards[i].hashes = make([][32]byte, capPow2)
			shards[i].words = make([][MaxDepth]byte, capPow2)
			shards[i].mask = capPow2 - 1
		}

		nCharset := len(Charset)
		var tasks []task
		for d := 1; d <= precomputeDepth; d++ {
			switch d {
			case 1:
				for i := range nCharset {
					tasks = append(tasks, task{depth: 1, prefLen: 1, prefix: [2]byte{Charset[i]}})
				}
			case 2:
				for i := range nCharset {
					tasks = append(tasks, task{depth: 2, prefLen: 1, prefix: [2]byte{Charset[i]}})
				}
			default:
				for i := range nCharset {
					for j := range nCharset {
						tasks = append(tasks, task{depth: d, prefLen: 2, prefix: [2]byte{Charset[i], Charset[j]}})
					}
				}
			}
		}

		numWorkers := runtime.GOMAXPROCS(0)
		var taskCounter atomic.Uint32
		totalTasks := uint32(len(tasks))

		var wg sync.WaitGroup
		for w := 0; w < numWorkers; w++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				var wb workerBatches
				var buf [MaxDepth]byte
				var indices [MaxDepth]int

				for {
					curr := taskCounter.Add(1) - 1
					if curr >= totalTasks {
						break
					}
					t := &tasks[curr]
					d := t.depth
					prefLen := t.prefLen

					for i := d; i < MaxDepth; i++ {
						buf[i] = 0
					}
					copy(buf[:prefLen], t.prefix[:prefLen])

					if prefLen == d {
						hash := sha256.Sum256(buf[:d])
						wb.add(&shards, hash, buf)
						continue
					}

					suffixLen := d - prefLen
					for i := range suffixLen {
						indices[i] = 0
						buf[prefLen+i] = Charset[0]
					}

					for {
						hash := sha256.Sum256(buf[:d])
						wb.add(&shards, hash, buf)

						pos := suffixLen - 1
						for pos >= 0 {
							indices[pos]++
							if indices[pos] < nCharset {
								buf[prefLen+pos] = Charset[indices[pos]]
								break
							}
							indices[pos] = 0
							buf[prefLen+pos] = Charset[0]
							pos--
						}
						if pos < 0 {
							break
						}
					}
				}
				wb.flush(&shards)
			}()
		}

		wg.Wait()
	}

	return MapReferential{
		shards:   &shards,
		maxDepth: precomputeDepth,
	}
}

// GetRaw performs an inlined, zero-allocation tag-probed lookup in the flat hash table.
func (r *MapReferential) GetRaw(hash [32]byte) [MaxDepth]byte {
	if r.shards != nil {
		shard := &r.shards[hash[0]]
		mask := shard.mask
		tag := binary.LittleEndian.Uint16(hash[1:3])
		if tag == 0 {
			tag = 1
		}
		slot := binary.LittleEndian.Uint32(hash[3:7]) & mask
		tags := shard.tags
		for {
			t := tags[slot]
			if t == tag {
				if shard.hashes[slot] == hash {
					return shard.words[slot]
				}
			} else if t == 0 {
				break
			}
			slot = (slot + 1) & mask
		}
	}
	return [MaxDepth]byte{}
}

// DecodeWord converts a fixed [MaxDepth]byte array into a Go string by trimming trailing null bytes.
func DecodeWord(b [MaxDepth]byte) string {
	n := 0
	for n < MaxDepth && b[n] != 0 {
		n++
	}
	return string(b[:n])
}

// Get returns the cracked string representation of the hash, or ("", false) if not found.
func (r *MapReferential) Get(hash [32]byte) (string, bool) {
	raw := r.GetRaw(hash)
	if raw == ([MaxDepth]byte{}) {
		return "", false
	}
	return DecodeWord(raw), true
}
