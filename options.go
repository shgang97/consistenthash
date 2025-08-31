package consistenthash

// Option 定义配置函数的类型
type Option func(*ConsistentHash)

// WithHashFunc 允许自定义哈希函数
func WithHashFunc(fn HashFunc) Option {
	return func(ch *ConsistentHash) {
		if fn != nil {
			ch.hashFunc = fn
		}
	}
}

// WithReplicas 允许自定义虚拟节点数量
func WithReplicas(replicas int) Option {
	return func(ch *ConsistentHash) {
		if replicas > 0 {
			ch.replicas = replicas
		}
	}
}

// WithMonitor 设置监控器
func WithMonitor(monitor Monitor) Option {
	return func(ch *ConsistentHash) {
		if monitor != nil {
			ch.monitor = monitor
		}
	}
}
