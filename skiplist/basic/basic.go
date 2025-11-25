package basic

import (
	"math/rand"

	"github.com/Hakuto4838/SkipList.git/skiplist"
)

const (
	maxLevel    = 32
	probability = 0.5
)

type basicNode struct {
	key   skiplist.K
	value skiplist.V
	next  []*basicNode
}

type BasicSkipList struct {
	head  *basicNode
	level int32
	rand  *rand.Rand
	size  int32
}

func NewBasicSkipList(seed int64) *BasicSkipList {
	return &BasicSkipList{
		head:  newNode(-1, 0, maxLevel),
		level: 1,
		rand:  rand.New(rand.NewSource(seed)),
		size:  0,
	}
}

func (sl *BasicSkipList) find(key skiplist.K) *basicNode {
	cur := sl.head
	for h := sl.level; h >= 0; h-- {
		for cur.next[h] != nil && cur.next[h].key < key {
			cur = cur.next[h]
		}
		if cur.next[h] != nil && cur.next[h].key == key {
			return cur.next[h]
		}
	}
	return nil
}

func newNode(key skiplist.K, value skiplist.V, level int32) *basicNode {
	return &basicNode{
		key:   key,
		value: value,
		next:  make([]*basicNode, level+1),
	}
}

func (sl *BasicSkipList) randomLevel() int32 {
	lvl := 0
	for sl.rand.Float64() < probability && lvl < maxLevel {
		lvl++
	}
	return int32(lvl)
}

func (sl *BasicSkipList) Put(key skiplist.K, value skiplist.V) {
	cur := sl.find(key)
	if cur != nil {
		cur.value = value
		return
	}
	lvl := sl.randomLevel()
	cur = newNode(key, value, lvl)
	sl.level = max(sl.level, lvl)
	curr := sl.head
	for h := sl.level; h >= 0; h-- {
		for curr.next[h] != nil && curr.next[h].key < key {
			curr = curr.next[h]
		}
		if h <= lvl {
			cur.next[h] = curr.next[h]
			curr.next[h] = cur
		}
		sl.size++
	}
}

func (sl *BasicSkipList) Get(key skiplist.K) (skiplist.V, bool) {
	cur := sl.find(key)
	if cur != nil {
		return cur.value, true
	}
	return 0, false
}

func (sl *BasicSkipList) Contains(key skiplist.K) bool {
	return sl.find(key) != nil
}

func (sl *BasicSkipList) Delete(key skiplist.K) {
	curr := sl.head
	for h := sl.level; h >= 0; h-- {
		for curr.next[h] != nil && curr.next[h].key < key {
			curr = curr.next[h]
		}
		if curr.next[h] != nil && curr.next[h].key == key {
			curr.next[h] = curr.next[h].next[h]
		}
	}
	sl.size--
}

func (sl *BasicSkipList) GetHead() skiplist.Nodelike {
	return sl.head
}

func (sl *BasicSkipList) GetMaxStats() (int, int) {
	return int(sl.size), int(sl.level)
}

func (nd *basicNode) GetKey() skiplist.K {
	return nd.key
}

func (nd *basicNode) GetValue() skiplist.V {
	return nd.value
}

func (nd *basicNode) GetLevel() int32 {
	return int32(len(nd.next) - 1)
}

func (nd *basicNode) GetNextAt(level int32) skiplist.Nodelike {
	if level < 0 || level >= int32(len(nd.next)) {
		return nil
	}
	if nd.next[level] == nil {
		return nil
	}
	return nd.next[level]
}
