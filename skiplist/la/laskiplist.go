package la

import (
	"math"
	"math/rand"

	"github.com/Hakuto4838/SkipList.git/skiplist"
)

const (
	maxLevel    = 32
	probability = 0.5
)

type laNode struct {
	key   skiplist.K
	value skiplist.V
	next  []*laNode
}

type LASkipList struct {
	head  *laNode
	level int32
	rand  *rand.Rand
	size  int32 // 數據集總元素數量（頻率基準）
}

func newNode(key skiplist.K, value skiplist.V, level int32) *laNode {
	n := &laNode{
		key:   key,
		value: value,
		next:  make([]*laNode, level+1),
	}
	return n
}

func NewLASkipList(seed int64) *LASkipList {
	return &LASkipList{
		head:  newNode(0, 0, maxLevel),
		level: 1, // 初始化為 1 層
		rand:  rand.New(rand.NewSource(seed)),
	}
}

// randomLevelWithNP 基於預測頻率計算節點高度
// np: 預測的未來出現頻率 (n * prob)
// l: 當前高度
func (sl *LASkipList) randomLevelWithNP(np float64) int32 {
	lvl := 0

	for lvl < maxLevel {
		l := lvl + 1 // 當前高度

		// 如果 np >= 2^(l-1)，保證升級
		if np >= math.Pow(2, float64(l-1)) {
			lvl++
		} else {
			// 否則有 1/2 機率升級
			if sl.rand.Float64() < probability {
				lvl++
			} else {
				break
			}
		}
	}

	return int32(lvl)
}

func (sl *LASkipList) find(key skiplist.K) (*laNode, bool) {
	cur := sl.head
	for h := sl.level; h >= 0; h-- {
		for cur.next[h] != nil && cur.next[h].key < key {
			cur = cur.next[h]
		}
		if cur.next[h] != nil && cur.next[h].key == key {
			return cur.next[h], true
		}
	}
	return nil, false
}

// PutWithNP 插入或更新 key 對應的 value，包含預測頻率
func (sl *LASkipList) PutWithNP(key skiplist.K, value skiplist.V, np float64) {
	if node, found := sl.find(key); found {
		node.value = value
		return
	}

	lvl := sl.randomLevelWithNP(np)
	newNode := newNode(key, value, lvl)
	sl.level = max(sl.level, lvl)

	curr := sl.head
	for h := sl.level; h >= 0; h-- {
		for curr.next[h] != nil && curr.next[h].key < key {
			curr = curr.next[h]
		}
		if h <= lvl {
			newNode.next[h] = curr.next[h]
			curr.next[h] = newNode
		}
	}
	sl.size++

}

// Put 實現 SkipList 介面的 Put 方法，使用傳統隨機高度
func (sl *LASkipList) Put(key skiplist.K, value skiplist.V) {
	sl.PutWithoutProb(key, value)
}

// PutWithoutProb 不帶概率的 Put 方法，使用傳統的隨機高度
func (sl *LASkipList) PutWithoutProb(key skiplist.K, value skiplist.V) {
	if node, found := sl.find(key); found {
		node.value = value
		return
	}

	lvl := sl.randomLevel()
	newNode := newNode(key, value, lvl)
	sl.level = max(sl.level, lvl)

	curr := sl.head
	for h := sl.level; h >= 0; h-- {
		for curr.next[h] != nil && curr.next[h].key < key {
			curr = curr.next[h]
		}
		if h <= lvl {
			newNode.next[h] = curr.next[h]
			curr.next[h] = newNode
		}
	}

	sl.size++
}

// 傳統的隨機高度計算方法
func (sl *LASkipList) randomLevel() int32 {
	lvl := 0
	for sl.rand.Float64() < probability && lvl < maxLevel {
		lvl++
	}
	return int32(lvl)
}

// Get 取得 key 對應的 value
func (sl *LASkipList) Get(key skiplist.K) (skiplist.V, bool) {
	node, found := sl.find(key)
	if found {
		return node.value, true
	}
	return 0, false
}

// Contains 判斷 key 是否存在
func (sl *LASkipList) Contains(key skiplist.K) bool {
	_, found := sl.find(key)
	return found
}

// Delete 刪除 key
func (sl *LASkipList) Delete(key skiplist.K) {
	curh := sl.level
	curr := sl.head

	for h := curh; h >= 0; h-- {
		for curr.next[h] != nil && curr.next[h].key < key {
			curr = curr.next[h]
		}
		if curr.next[h] != nil && curr.next[h].key == key {
			curr.next[h] = curr.next[h].next[h]
		}
	}

	newlvl := curh
	for newlvl > 0 && sl.head.next[newlvl] == nil {
		newlvl--
	}
	sl.level = newlvl
	sl.size--
}

// 輔助函數
func max(a, b int32) int32 {
	if a > b {
		return a
	}
	return b
}

func (sl *LASkipList) GetMaxStats() (int, int) {
	return int(sl.size), int(sl.level)
}

func (sl *LASkipList) GetHead() skiplist.Nodelike {
	return sl.head
}

// Node 實作 Nodelike 介面
func (n *laNode) GetKey() skiplist.K {
	return n.key
}

func (n *laNode) GetValue() skiplist.V {
	return n.value
}

func (n *laNode) GetLevel() int32 {
	return int32(len(n.next) - 1)
}

func (n *laNode) GetNextAt(level int32) skiplist.Nodelike {
	if level < 0 || level >= int32(len(n.next)) {
		return nil
	}
	if n.next[level] == nil {
		return nil
	}
	return n.next[level]
}
