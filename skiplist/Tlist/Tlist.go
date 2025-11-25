package tlist

import (
	"github.com/Hakuto4838/SkipList.git/skiplist"
)

const (
	maxLevel    = 32
	probability = 0.5
)

type tNode struct {
	key   skiplist.K
	value skiplist.V
	next  []*tNode
	del   bool
}

type TList struct {
	head  *tNode
	level int32
	size  int32
	span  int32
}

func newNode(key skiplist.K, value skiplist.V, level int32) *tNode {
	n := &tNode{
		key:   key,
		value: value,
		next:  make([]*tNode, level+1),
		del:   false,
	}
	return n
}

func NewSkipList(span int32) *TList {
	return &TList{
		head:  newNode(-1, 0, maxLevel+1),
		level: 1, // 初始化為 1 層
		span:  span,
	}
}

func (sl *TList) pureTravel(key skiplist.K) (*tNode, bool) {
	curr := sl.head
	for level := sl.level - 1; level >= 0; level-- {
		for curr.next[level] != nil && curr.next[level].key < key {
			curr = curr.next[level]
		}
		if curr.next[level] != nil && curr.next[level].key == key {
			return curr.next[level], true
		}
	}
	return curr, false
}

func (sl *TList) buildTravel(key skiplist.K) (*tNode, bool) {
	curr := sl.head
	stepCounter := int32(0)
	var stationPointer *tNode = sl.head
	for level := sl.level - 1; level >= 0; level-- {
		for curr.next[level] != nil && curr.next[level].key < key {
			curr = curr.next[level]

			stepCounter++
			if stepCounter >= sl.span && level < maxLevel {
				// 升階判定
				if curr.next[level] == nil || curr.next[level].GetLevel() <= level {
					curr.upgrade(stationPointer)
					stationPointer = curr
					if level == sl.level-1 {
						sl.level++
					}
				}
				stepCounter = 0
			}
		}
		if curr.next[level] != nil && curr.next[level].key == key {
			return curr.next[level], true
		}
		stationPointer = curr
		stepCounter = 0
	}
	return curr, false
}

func (nd *tNode) upgrade(parent *tNode) {
	if nd.GetLevel() >= parent.GetLevel() {
		return
	}
	lvl := int(nd.GetLevel()) + 1
	nd.next = append(nd.next, parent.next[lvl])
	parent.next[lvl] = nd
}

// 實現 SkipList interface 的方法

// Put 插入或更新 key 對應的 value
func (sl *TList) Put(key skiplist.K, value skiplist.V) {
	node, found := sl.buildTravel(key)
	if found {
		node.value = value
		node.del = false
		sl.size++
		return
	}

	newNode := newNode(key, value, 0)
	newNode.next[0] = node.next[0]
	node.next[0] = newNode
	sl.size++
}

// Get 取得 key 對應的 value
func (sl *TList) Get(key skiplist.K) (skiplist.V, bool) {
	node, found := sl.buildTravel(key)
	if found {
		if node.del {
			return 0, false
		}
		return node.value, true
	}
	return 0, false
}

// Contains 判斷 key 是否存在
func (sl *TList) Contains(key skiplist.K) bool {
	node, found := sl.buildTravel(key)
	return found && !node.del
}

// Delete 刪除 key
func (sl *TList) Delete(key skiplist.K) {
	node, found := sl.buildTravel(key)
	if found {
		node.del = true
		sl.size--
	}
}

// GetHead 實現 SkipList interface
func (sl *TList) GetHead() skiplist.Nodelike {
	if sl.head == nil {
		return nil
	}
	return sl.head
}

func (nd *tNode) GetLevel() int32 {
	return int32(len(nd.next) - 1)
}

// GetKey 實現 Nodelike 介面
func (nd *tNode) GetKey() skiplist.K {
	return nd.key
}

// GetValue 實現 Nodelike 介面
func (nd *tNode) GetValue() skiplist.V {
	return nd.value
}

func (nd *tNode) GetNextAt(level int32) skiplist.Nodelike {
	if level < 0 || level >= int32(len(nd.next)) || nd.next[level] == nil {
		return nil
	}
	return nd.next[level]
}

func (sl *TList) GetMaxStats() (int, int) {
	return int(sl.size), int(sl.level)
}
