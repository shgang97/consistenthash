package consistenthash

import "errors"

// 定义包级错误
var (
	ErrNoNodesAvailable    = errors.New("no nodes available in the consistent hash ring")
	ErrEmptyKey            = errors.New("key cannot be empty")
	ErrNodeAlreadyExists   = errors.New("node already exists")
	ErrNodeDoesNotExist    = errors.New("node does not exist")
	ErrInvalidReplicaCount = errors.New("replica count must be positive")
)
