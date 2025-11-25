package splay

import (
	"math/rand"

	// "sync"

	"github.com/Hakuto4838/SkipList.git/skiplist"
)

const MAX_LEVEL = 32 // 根據實際需求調整

type SplayNode struct {
	key       skiplist.K
	value     skiplist.V
	zeroLevel int32
	topLevel  int32
	selfhits  int32
	next      [MAX_LEVEL + 1]*SplayNode
	hits      [MAX_LEVEL + 1]int32
	deleted   bool
}

type SplayList struct {
	m         int32      // 動態計數器，記錄目前操作次數，供 balancing phase 使用
	zeroLevel int32      // 記錄當前 zero level
	head      *SplayNode // 頭節點
	p         float64    // 平衡條件的常數，初始化後不變
	size      int32      // 記錄當前節點數
}

func NewSplayList(p float64) *SplayList {
	head := &SplayNode{topLevel: MAX_LEVEL, zeroLevel: MAX_LEVEL - 1}
	for i := 0; i <= MAX_LEVEL; i++ {
		head.next[i] = nil
	}
	return &SplayList{
		head:      head,
		zeroLevel: MAX_LEVEL - 1,
		p:         p,
	}
}

// contains 函式
func (list *SplayList) Contains(key skiplist.K) bool {
	node := list.find(key)
	if node == nil {
		return false
	}
	list.tryUpdate(key)
	return !node.deleted
}

// find 函式
func (list *SplayList) find(key skiplist.K) *SplayNode {
	pred := list.head
	var succ *SplayNode
	for level := int32(MAX_LEVEL - 1); level >= list.zeroLevel; level-- {
		list.updateUpToLevel(pred, level)
		succ = pred.next[level]
		if succ == nil {
			continue
		}
		list.updateUpToLevel(succ, level)
		for succ != nil && succ.key < key {
			pred = succ
			succ = pred.next[level]
			if succ == nil {
				break
			}
			list.updateUpToLevel(succ, level)
		}
		if succ != nil && succ.key == key {
			return succ
		}
	}
	return nil
}

// updateUpToLevel 函式
func (list *SplayList) updateUpToLevel(node *SplayNode, level int32) {
	if node == nil {
		return
	}
	if node.zeroLevel <= level {
		return
	}
	for node.zeroLevel > level {
		node.hits[node.zeroLevel-1] = 0
		node.next[node.zeroLevel-1] = node.next[node.zeroLevel]
		node.zeroLevel--
	}
}

// getHits 函式：計算節點在指定層的 hits
func getHits(node *SplayNode, h int32) int32 {
	if node.zeroLevel > h {
		return node.selfhits
	}
	return node.selfhits + node.hits[h]
}

// update 函式：根據論文偽碼實作 balancing phase
func (list *SplayList) update(key skiplist.K) {
	list.m++

	pred := list.head
	pred.hits[MAX_LEVEL]++
	var prepred, curr *SplayNode
	for level := int32(MAX_LEVEL - 1); level >= list.zeroLevel; level-- {
		list.updateUpToLevel(pred, level)
		prepred = pred
		curr = pred.next[level]
		list.updateUpToLevel(curr, level)
		if curr == nil || curr.key > key { //走一步就過頭
			pred.hits[level]++
			continue
		}

		found := false
		for curr != nil && curr.key <= key {
			list.updateUpToLevel(curr, level)

			if curr.next[level] == nil || curr.next[level].key > key {
				if curr.key == key {
					found = true
					curr.selfhits++
				} else {
					curr.hits[level]++
				}
				break
			}

			//ascent condition+
			curh := curr.topLevel
			if curh+1 < MAX_LEVEL && curh < prepred.topLevel && prepred.hits[curh+1]-prepred.hits[curh] > list.getAscentThreshold(curh, list.m) {
				for curh+1 < MAX_LEVEL && curh < prepred.topLevel && prepred.hits[curh+1]-prepred.hits[curh] > list.getAscentThreshold(curh, list.m) {
					curr.topLevel++
					curh++
					curr.hits[curh] = prepred.hits[curh] - prepred.hits[curh-1] - curr.selfhits
					curr.next[curh] = prepred.next[curh]
					prepred.hits[curh] = prepred.hits[curh-1]
					prepred.next[curh] = curr
				}
				prepred = curr
				pred = curr
				curr = pred.next[level]
				continue // 升級後無需判定降級

				//descend condition
			} else if curr.topLevel == level && curr.next[level] != nil && curr.next[level].key <= key &&
				getHits(curr, level)+getHits(pred, level) <= list.getDescentThreshold(level, list.m) {
				currZero := list.zeroLevel
				if level == currZero {
					//擴張
					list.zeroLevel--
				}
				list.updateUpToLevel(curr, level-1)
				list.updateUpToLevel(pred, level-1)

				pred.hits[level] += getHits(curr, level)
				curr.hits[level] = 0
				pred.next[level] = curr.next[level]
				curr.next[level] = nil
				curr.topLevel--
				curr = pred.next[level]
				continue
			}
			//沒上升也沒下降
			pred = curr
			curr = pred.next[level]
		}
		if found {
			return
		}
	} //end for

	// 調試用：若需要驗證 hits 可手動呼叫 CheckHits()
}

func (list *SplayList) getAscentThreshold(h int32, M int32) int32 {
	return M / (1 << (MAX_LEVEL - 1 - h))
}

func (list *SplayList) getDescentThreshold(h int32, M int32) int32 {
	return M / (1 << (MAX_LEVEL - h))
}

// Put 方法：插入或更新節點
func (list *SplayList) Put(key skiplist.K, value skiplist.V) {
	// 先嘗試找到現有的節點
	node := list.find(key)

	if node != nil {
		// 找到節點，檢查是否被標記為已刪除
		if node.deleted {
			list.size++
		}

		node.deleted = false
		node.value = value
		list.tryUpdate(key)
	} else {
		// 沒有找到節點，建立新節點
		list.insertNewNode(key, value)
		list.update(key) //必定更新
		list.size++
	}

}

// Delete 方法：標記刪除節點
func (list *SplayList) Delete(key skiplist.K) {
	node := list.find(key)
	if node != nil {
		node.deleted = true
		list.tryUpdate(key)
		list.size--
	}
}

// Get 方法：獲取節點值
func (list *SplayList) Get(key skiplist.K) (skiplist.V, bool) {
	node := list.find(key)
	if node == nil {
		return 0, false
	}
	list.tryUpdate(key)

	if node.deleted {
		return 0, false
	}

	return node.value, true
}

// insertNewNode 方法：插入新節點
func (list *SplayList) insertNewNode(key skiplist.K, value skiplist.V) {
	zLevel := list.zeroLevel
	// 建立新節點
	newNode := &SplayNode{
		key:       key,
		value:     value,
		topLevel:  zLevel,
		zeroLevel: zLevel,
		selfhits:  1, // 新節點的 hit 為 1
		deleted:   false,
	}

	// 初始化 next 陣列
	for i := 0; i <= MAX_LEVEL; i++ {
		newNode.next[i] = nil
	}

	// 尋找插入位置並更新連結
	list.insertNode(newNode, zLevel)
}

// insertNode 方法：在指定層級插入節點
func (list *SplayList) insertNode(newNode *SplayNode, level int32) {
	// 從最高層開始尋找插入位置
	pred := list.head

	for h := int32(MAX_LEVEL); h >= list.zeroLevel; h-- {
		// 更新 pred 到當前層級
		list.updateUpToLevel(pred, h)

		// 尋找插入位置
		curr := pred.next[h]
		// 向右尋找插入位置
		for curr != nil && curr.key < newNode.key {
			pred = curr
			curr = pred.next[h]
			if curr != nil {
				list.updateUpToLevel(curr, h)
			}
		}

		// 在新節點的 zeroLevel 以下皆需連接，以維持 SkipList 結構
		if h <= level {
			newNode.next[h] = curr
			pred.next[h] = newNode
		}
	}
}

func (list *SplayList) GetHead() skiplist.Nodelike {
	list.UpdateAllLvl()
	return list.head
}

func (list *SplayList) GetMaxStats() (maxNodes int, maxLevel int) {
	list.UpdateAllLvl()
	return int(list.size), int(MAX_LEVEL - list.zeroLevel)
}

func (n *SplayNode) GetKey() skiplist.K {
	return n.key
}

func (n *SplayNode) GetValue() skiplist.V {
	return n.value
}

func (n *SplayNode) GetLevel() int32 {
	return n.topLevel - n.zeroLevel
}

func (n *SplayNode) GetNextAt(level int32) skiplist.Nodelike {
	trueLevel := level + n.zeroLevel
	if trueLevel < 0 || trueLevel > n.topLevel {
		return nil
	}
	if n.next[trueLevel] == nil {
		return nil
	}
	return n.next[trueLevel]
}

func (sl *SplayList) UpdateAllLvl() {
	// 更新所有層級的 zeroLevel
	h := sl.zeroLevel
	node := sl.head
	for node != nil {
		sl.updateUpToLevel(node, h)
		node = node.next[h]
	}
}

func (sl *SplayList) tryUpdate(key skiplist.K) {
	if rand.Float64() > sl.p {
		return
	}
	sl.update(key)
}

// // ---------------- hits 檢查工具 ----------------

// // isGoodhit 透過遞迴檢查每層 hits 的總和是否等於下層加總（含 selfhits）
// func (node *SplayNode) isGoodhit(level int32) bool {
// 	if level <= node.zeroLevel {
// 		return true
// 	}

// 	sum := int32(0)
// 	if node.isGoodhit(level - 1) {
// 		sum += node.hits[level-1] + node.selfhits
// 	} else {
// 		return false
// 	}

// 	curr := node.next[level-1]
// 	for curr != nil && curr.topLevel == level-1 {
// 		if curr.isGoodhit(level - 1) {
// 			sum += curr.hits[level-1] + curr.selfhits
// 		} else {
// 			return false
// 		}
// 		curr = curr.next[level-1]
// 	}
// 	return sum == node.hits[level]+node.selfhits
// }

// // CheckHits 用來驗證整棵 SplayList 的 hits 是否正確
// func (l *SplayList) CheckHits() bool {
// 	if l.head.isGoodhit(MAX_LEVEL) {
// 		return true
// 	}
// 	// 若 hits 不正確，輸出 skip list 結構與 hits 方便除錯
// 	analyTool.PrintSkipList(l, 32, 10000)
// 	fmt.Println("--------------------------------")
// 	l.PrintHits()
// 	panic("hits is not good")
// }

// // PrintHits 輸出各層 hits 及 selfhits，僅供偵錯使用
// func (l *SplayList) PrintHits() {
// 	curr := l.head
// 	n := int(curr.GetLevel())
// 	line := make([]string, n+1)
// 	lvl := int(curr.zeroLevel)
// 	selfhit := "Level S : "

// 	for i := 0; i <= n; i++ {
// 		line[i] = fmt.Sprintf("Level %d : ", i)
// 	}

// 	for curr != nil {
// 		// 確保 zeroLevel 位置更新
// 		l.updateUpToLevel(curr, int32(lvl))

// 		selfhit += fmt.Sprintf("%3d ->", curr.selfhits)
// 		for i := 0; i <= n; i++ {
// 			if curr.GetLevel() >= int32(i) {
// 				line[i] += fmt.Sprintf("%3d ->", curr.hits[i+lvl])
// 			} else {
// 				line[i] += "    ->"
// 			}
// 		}

// 		curr = curr.next[lvl]
// 	}

// 	for i := n; i >= 0; i-- {
// 		fmt.Println(line[i])
// 	}
// 	fmt.Println(selfhit)
// }
