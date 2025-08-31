package consistenthash

import "time"

// Monitor 定义监控接口
type Monitor interface {
	// RecordDistributionStats 记录数据分布统计信息
	RecordDistributionStats(stats DistributionStats)
	// RecordNodeEvent 记录节点事件（添加、删除）
	RecordNodeEvent(eventType NodeEventType, node string, duration time.Duration)
	// RecordLookup 记录查找操作
	RecordLookup(key string, node string, found bool, duration time.Duration)
	// RecordError 记录错误
	RecordError(errorType error, operation string)
}

// NodeEventType 节点事件类型
type NodeEventType int

const (
	NodeAdded NodeEventType = iota
	NodeRemoved
)

// DistributionStats 数据分布统计信息
type DistributionStats struct {
	NodeCount      int
	VirtualNodes   int
	Mean           float64
	StdDev         float64
	CoefficientVar float64
	Timestamp      time.Time
}

// NoopMonitor 空操作的监控器实现
type NoopMonitor struct{}

func (n *NoopMonitor) RecordDistributionStats(stats DistributionStats)                              {}
func (n *NoopMonitor) RecordNodeEvent(eventType NodeEventType, node string, duration time.Duration) {}
func (n *NoopMonitor) RecordLookup(key string, node string, found bool, duration time.Duration)     {}
func (n *NoopMonitor) RecordError(errorType error, operation string)                                {}

// 默认监控器
var defaultMonitor = &NoopMonitor{}
