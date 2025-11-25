package skiplist

type K = int64
type V = float64

type SkipList interface {
	Contains(key K) bool
	Get(key K) (V, bool)
	Put(key K, value V)
	Delete(key K)
	GetHead() Nodelike
}

// Analyable 提供分析功能的介面
type Analyable interface {
	SkipList
	// GetMaxStats 獲取最大節點數和最大層級
	GetMaxStats() (maxNodes int, maxLevel int)
}

type Nodelike interface {
	GetKey() K
	GetValue() V
	GetLevel() int32
	GetNextAt(level int32) Nodelike
}
