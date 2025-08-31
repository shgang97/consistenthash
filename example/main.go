package main

import (
	"fmt"
	"github.com/shgang97/consistenthash"
	"log"
	"time"
)

// SimpleMonitor 简单的监控器实现
type SimpleMonitor struct{}

func (m *SimpleMonitor) RecordDistributionStats(stats consistenthash.DistributionStats) {
	fmt.Printf("Distribution Stats: Nodes=%d, VirtualNodes=%d, Mean=%.2f, StdDev=%.2f, CV=%.3f\n",
		stats.NodeCount, stats.VirtualNodes, stats.Mean, stats.StdDev, stats.CoefficientVar)
}

func (m *SimpleMonitor) RecordNodeEvent(eventType consistenthash.NodeEventType, node string, duration time.Duration) {
	event := "added"
	if eventType == consistenthash.NodeRemoved {
		event = "removed"
	}
	fmt.Printf("Node %s %s in %v\n", node, event, duration)
}

func (m *SimpleMonitor) RecordLookup(key string, node string, found bool, duration time.Duration) {
	// 在实际生产中，这里可以记录到监控系统
}

func (m *SimpleMonitor) RecordError(errorType error, operation string) {
	log.Printf("Error in %s: %v", operation, errorType)
}

func main() {
	// 创建一致性哈希实例，使用自定义监控器
	ch := consistenthash.NewConsistentHash(
		consistenthash.WithReplicas(100),
		consistenthash.WithMonitor(&SimpleMonitor{}),
	)

	// 添加节点
	nodes := []string{"node1", "node2", "node3", "node4", "node5"}
	for _, node := range nodes {
		if err := ch.AddNode(node); err != nil {
			log.Fatalf("Failed to add node: %v", err)
		}
	}

	// 测试键查找
	testKeys := []string{"user1", "user2", "productA", "order100", "sessionXYZ"}
	for _, key := range testKeys {
		node, err := ch.Get(key)
		if err != nil {
			log.Printf("Error getting node for key %s: %v", key, err)
			continue
		}
		fmt.Printf("Key '%s' is assigned to node: %s\n", key, node)
	}

	// 查看数据分布情况
	stats := ch.CalculateDistributionStats()
	fmt.Printf("Distribution quality: %.3f (lower is better)\n", stats.CoefficientVar)

	// 添加新节点并观察变化
	fmt.Println("\nAdding node6...")
	ch.AddNode("node6")

	for _, key := range testKeys {
		node, _ := ch.Get(key)
		fmt.Printf("Key '%s' is now assigned to node: %s\n", key, node)
	}

	// 查看新的数据分布情况
	stats = ch.CalculateDistributionStats()
	fmt.Printf("New distribution quality: %.3f (lower is better)\n", stats.CoefficientVar)
}
