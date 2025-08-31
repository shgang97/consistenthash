package consistenthash

import (
	"crypto/md5"
	"fmt"
	"github.com/cespare/xxhash/v2"
	"sort"
	"strconv"
	"sync"
	"time"
)

// HashFunc 定义哈希函数类型
type HashFunc func(data []byte) uint64

// ConsistentHash 一致性哈希结构体
type ConsistentHash struct {
	hashFunc     HashFunc
	replicas     int
	circle       map[uint64]string
	sortedHashes []uint64
	nodes        map[string]bool
	mu           sync.RWMutex
	monitor      Monitor
}

const (
	// DefaultReplicas 默认虚拟节点数量
	DefaultReplicas = 160
)

// NewConsistentHash 创建一致性哈希实例
func NewConsistentHash(opts ...Option) *ConsistentHash {
	ch := &ConsistentHash{
		replicas: DefaultReplicas,
		hashFunc: xxhash.Sum64, // 使用高性能的xxhash作为默认哈希函数
		circle:   make(map[uint64]string),
		nodes:    make(map[string]bool),
		monitor:  defaultMonitor,
	}

	// 应用所有选项
	for _, opt := range opts {
		opt(ch)
	}

	return ch
}

// AddNode 添加节点到一致性哈希环
func (ch *ConsistentHash) AddNode(node string) error {
	if node == "" {
		return ErrEmptyKey
	}

	start := time.Now()
	ch.mu.Lock()
	defer ch.mu.Unlock()

	// 检查节点是否已存在
	if ch.nodes[node] {
		ch.monitor.RecordError(ErrNodeAlreadyExists, "AddNode")
		return ErrNodeAlreadyExists
	}

	ch.nodes[node] = true

	// 为节点创建虚拟节点
	for i := 0; i < ch.replicas; i++ {
		virtualKey := ch.generateVirtualKey(node, i)
		hash := ch.hashFunc([]byte(virtualKey))

		// 检查哈希冲突
		if existingNode, exists := ch.circle[hash]; exists {
			// 处理哈希冲突（概率极低，但需要处理）
			virtualKey = ch.handleHashCollision(node, i, hash, existingNode)
			hash = ch.hashFunc([]byte(virtualKey))
		}

		ch.circle[hash] = node
		ch.sortedHashes = append(ch.sortedHashes, hash)
	}

	// 对虚拟节点哈希值进行排序
	sort.Slice(ch.sortedHashes, func(i, j int) bool {
		return ch.sortedHashes[i] < ch.sortedHashes[j]
	})

	ch.monitor.RecordNodeEvent(NodeAdded, node, time.Since(start))
	return nil
}

// generateVirtualKey 生成虚拟节点键
func (ch *ConsistentHash) generateVirtualKey(node string, index int) string {
	return node + "#" + strconv.Itoa(index)
}

// handleHashCollision 处理哈希冲突
func (ch *ConsistentHash) handleHashCollision(node string, index int, hash uint64, existingNode string) string {
	// 使用不同的策略生成新的虚拟键
	for attempt := 1; attempt <= 5; attempt++ {
		newVirtualKey := fmt.Sprintf("%s#%d#%d", node, index, attempt)
		newHash := ch.hashFunc([]byte(newVirtualKey))

		if _, exists := ch.circle[newHash]; !exists {
			return newVirtualKey
		}
	}

	// 如果多次尝试仍有冲突，使用更复杂的方法
	hasher := md5.New()
	hasher.Write([]byte(node))
	hasher.Write([]byte(strconv.Itoa(index)))
	hasher.Write([]byte(strconv.FormatUint(hash, 10)))
	return fmt.Sprintf("%x", hasher.Sum(nil))
}

// RemoveNode 从一致性哈希环中移除节点
func (ch *ConsistentHash) RemoveNode(node string) error {
	if node == "" {
		return ErrEmptyKey
	}

	start := time.Now()
	ch.mu.Lock()
	defer ch.mu.Unlock()

	if !ch.nodes[node] {
		ch.monitor.RecordError(ErrNodeDoesNotExist, "RemoveNode")
		return ErrNodeDoesNotExist
	}

	delete(ch.nodes, node)

	// 重建哈希环，移除该节点的所有虚拟节点
	newHashes := make([]uint64, 0, len(ch.sortedHashes))
	for _, hash := range ch.sortedHashes {
		if ch.circle[hash] != node {
			newHashes = append(newHashes, hash)
		} else {
			delete(ch.circle, hash)
		}
	}

	ch.sortedHashes = newHashes

	ch.monitor.RecordNodeEvent(NodeRemoved, node, time.Since(start))
	return nil
}

// Get 根据key获取对应的节点
func (ch *ConsistentHash) Get(key string) (string, error) {
	if key == "" {
		ch.monitor.RecordError(ErrEmptyKey, "Get")
		return "", ErrEmptyKey
	}

	start := time.Now()
	ch.mu.RLock()
	defer ch.mu.RUnlock()

	if len(ch.circle) == 0 {
		ch.monitor.RecordError(ErrNoNodesAvailable, "Get")
		return "", ErrNoNodesAvailable
	}

	hash := ch.hashFunc([]byte(key))
	idx := ch.findNodeIndex(hash)
	node := ch.circle[ch.sortedHashes[idx]]

	ch.monitor.RecordLookup(key, node, true, time.Since(start))
	return node, nil
}

// findNodeIndex 在排序的哈希列表中查找节点的索引
func (ch *ConsistentHash) findNodeIndex(hash uint64) int {
	// 使用二分查找找到第一个大于等于key哈希的虚拟节点
	idx := sort.Search(len(ch.sortedHashes), func(i int) bool {
		return ch.sortedHashes[i] >= hash
	})

	// 如果找不到(说明key的哈希值大于所有虚拟节点)，则使用第一个虚拟节点(环状结构)
	if idx == len(ch.sortedHashes) {
		idx = 0
	}

	return idx
}

// GetNodes 获取所有真实节点
func (ch *ConsistentHash) GetNodes() []string {
	ch.mu.RLock()
	defer ch.mu.RUnlock()

	nodes := make([]string, 0, len(ch.nodes))
	for node := range ch.nodes {
		nodes = append(nodes, node)
	}

	return nodes
}

// GetNodeCount 获取节点数量
func (ch *ConsistentHash) GetNodeCount() int {
	ch.mu.RLock()
	defer ch.mu.RUnlock()

	return len(ch.nodes)
}

// GetVirtualNodeCount 获取虚拟节点数量
func (ch *ConsistentHash) GetVirtualNodeCount() int {
	ch.mu.RLock()
	defer ch.mu.RUnlock()

	return len(ch.sortedHashes)
}

// CalculateDistributionStats 计算数据分布统计信息
func (ch *ConsistentHash) CalculateDistributionStats() DistributionStats {
	ch.mu.RLock()
	defer ch.mu.RUnlock()

	// 统计每个真实节点负责的虚拟节点数量
	nodeCounts := make(map[string]int)
	for _, node := range ch.circle {
		nodeCounts[node]++
	}

	// 计算统计数据
	total := float64(len(ch.sortedHashes))
	nodeCount := len(nodeCounts)

	if nodeCount == 0 {
		return DistributionStats{
			NodeCount:    0,
			VirtualNodes: 0,
			Timestamp:    time.Now(),
		}
	}

	mean := total / float64(nodeCount)
	variance := 0.0

	for _, count := range nodeCounts {
		diff := float64(count) - mean
		variance += diff * diff
	}
	variance /= float64(nodeCount)

	stdDev := sqrt(variance)
	coefVar := stdDev / mean

	stats := DistributionStats{
		NodeCount:      nodeCount,
		VirtualNodes:   int(total),
		Mean:           mean,
		StdDev:         stdDev,
		CoefficientVar: coefVar,
		Timestamp:      time.Now(),
	}

	ch.monitor.RecordDistributionStats(stats)
	return stats
}

// sqrt 计算平方根（Go标准库没有直接提供float64的平方根函数）
func sqrt(x float64) float64 {
	// 使用牛顿迭代法求平方根
	if x == 0 {
		return 0
	}

	z := x / 2
	for i := 0; i < 10; i++ {
		z -= (z*z - x) / (2 * z)
	}
	return z
}
