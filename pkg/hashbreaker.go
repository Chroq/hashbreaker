package pkg

import (
	"crypto/sha256"
	"runtime"
	"sync"
	"sync/atomic"
)

const (
	Charset   = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789@"
	MaxDepth  = 8
	batchSize = 128
)

type Shard struct {
	mu sync.Mutex
	m  map[[32]byte][MaxDepth]byte
}

type MapReferential struct {
	shards []Shard
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

func decodeWord(b [MaxDepth]byte) string {
	n := 0
	for n < MaxDepth && b[n] != 0 {
		n++
	}
	return string(b[:n])
}

type batchItem struct {
	hash [32]byte
	word [MaxDepth]byte
}

type workerBatches struct {
	items [256][batchSize]batchItem
	lens  [256]int
}

func (wb *workerBatches) add(shards []Shard, hash [32]byte, word [MaxDepth]byte) {
	s := hash[0]
	idx := wb.lens[s]
	wb.items[s][idx] = batchItem{hash: hash, word: word}
	idx++
	if idx == batchSize {
		shard := &shards[s]
		shard.mu.Lock()
		for i := 0; i < batchSize; i++ {
			shard.m[wb.items[s][i].hash] = wb.items[s][i].word
		}
		shard.mu.Unlock()
		idx = 0
	}
	wb.lens[s] = idx
}

func (wb *workerBatches) flush(shards []Shard) {
	for s := range 256 {
		n := wb.lens[s]
		if n > 0 {
			shard := &shards[s]
			shard.mu.Lock()
			for i := range n {
				shard.m[wb.items[s][i].hash] = wb.items[s][i].word
			}
			shard.mu.Unlock()
			wb.lens[s] = 0
		}
	}
}

type task struct {
	depth   int
	prefLen int
	prefix  [2]byte
}

func NewMapReferential(depth int) MapReferential {
	shards := make([]Shard, 256)
	total := TotalCombinations(depth)
	shardCap := (total / 256) + (total / 2048) + 16
	for i := range 256 {
		shards[i].m = make(map[[32]byte][MaxDepth]byte, shardCap)
	}

	var tasks []task
	for d := 1; d <= depth; d++ {
		switch d {
		case 1:
			for i := 0; i < len(Charset); i++ {
				tasks = append(tasks, task{depth: 1, prefLen: 1, prefix: [2]byte{Charset[i]}})
			}
		case 2:
			for i := 0; i < len(Charset); i++ {
				tasks = append(tasks, task{depth: 2, prefLen: 1, prefix: [2]byte{Charset[i]}})
			}
		default:
			for i := 0; i < len(Charset); i++ {
				for j := 0; j < len(Charset); j++ {
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

				copy(buf[:prefLen], t.prefix[:prefLen])

				if prefLen == d {
					hash := sha256.Sum256(buf[:d])
					var w [MaxDepth]byte
					copy(w[:], buf[:d])
					wb.add(shards, hash, w)
					continue
				}

				suffixLen := d - prefLen
				for i := 0; i < suffixLen; i++ {
					indices[i] = 0
					buf[prefLen+i] = Charset[0]
				}

				for {
					hash := sha256.Sum256(buf[:d])
					var w [MaxDepth]byte
					copy(w[:], buf[:d])
					wb.add(shards, hash, w)

					pos := suffixLen - 1
					for pos >= 0 {
						indices[pos]++
						if indices[pos] < len(Charset) {
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
			wb.flush(shards)
		}()
	}

	wg.Wait()
	return MapReferential{shards: shards}
}

func (r MapReferential) Get(hash [32]byte) (string, bool) {
	if len(r.shards) == 0 {
		return "", false
	}
	shard := &r.shards[hash[0]]
	word, ok := shard.m[hash]
	if !ok {
		return "", false
	}
	return decodeWord(word), true
}

func GetHash(word string) [32]byte {
	return sha256.Sum256([]byte(word))
}
