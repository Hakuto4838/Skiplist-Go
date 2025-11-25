package analyTool

import (
	"encoding/csv"
	"fmt"
	"sort"

	"github.com/Hakuto4838/SkipList.git/skiplist"
)

type StepMap map[skiplist.K]int

// FindStep 計算找到指定 key 的總步數和各層步數
func FindStep(sl skiplist.Analyable, key skiplist.K) (step int, level []int) {
	cur := sl.GetHead()
	if cur == nil {
		return 0, []int{}
	}

	totalSteps := 0

	// 獲取最大層級
	_, maxLevel := sl.GetMaxStats()
	stepsPerLevel := make([]int, maxLevel+1)

	// 從最高層開始搜尋
	for h := maxLevel; h >= 0; h-- {
		levelSteps := 0

		// 在當前層級水平移動
		for cur != nil {
			nextNode := cur.GetNextAt(int32(h))
			if nextNode == nil || nextNode.GetKey() >= key {
				break
			}
			cur = nextNode
			levelSteps++
		}

		// 如果找到目標 key，記錄步數並返回
		if cur != nil {
			nextNode := cur.GetNextAt(int32(h))
			if nextNode != nil && nextNode.GetKey() == key {
				levelSteps++ // 加上最後一步
				stepsPerLevel[h] = levelSteps
				totalSteps += levelSteps

				return totalSteps, stepsPerLevel[:maxLevel+1]
			}
		}

		stepsPerLevel[h] = levelSteps
		totalSteps += levelSteps + 1 // 加上向下移動
	}

	// 如果沒找到，返回搜尋過程中的總步數
	return totalSteps, stepsPerLevel[:maxLevel+1]
}

// AnalyzeStep 根據 map 提供的 key 出現機率計算平均搜尋步數
func AnalyzeStep(sl skiplist.Analyable, keys map[skiplist.K]float64) (float64, StepMap) {
	if len(keys) == 0 {
		return 0.0, nil
	}

	step := StepMap{}

	var totalExpectedSteps float64
	var totalProbability float64

	// 遞迴搜尋所有node，若存在key則計算期望步數
	var dfs func(node skiplist.Nodelike, level int, steps int)
	dfs = func(node skiplist.Nodelike, level int, steps int) {
		if node == nil {
			return
		}

		if node.GetLevel() == int32(level) { // 初次到來，計算期望步數
			if _, ok := keys[node.GetKey()]; ok {
				// fmt.Printf("key found in keys map: %d, steps: %d, probability: %f\n", node.GetKey(), steps, keys[node.GetKey()])
				totalExpectedSteps += float64(steps) * keys[node.GetKey()]
				totalProbability += keys[node.GetKey()]
				// fmt.Printf("totalExpectedSteps: %f, totalProbability: %f\n", totalExpectedSteps, totalProbability)
				step[node.GetKey()] = steps
			} else {
				fmt.Printf("warning: key not found in keys map: %d\n", node.GetKey())
			}
		}
		if level > 0 { // 下降也算一步
			dfs(node, level-1, steps+1)
		}

		nextNode := node.GetNextAt(int32(level))
		if nextNode != nil && nextNode.GetLevel() == int32(level) {
			// 若下一個節點高度較高，則不屬於本次走訪
			dfs(nextNode, level, steps+1)
		}
	}

	_, maxLevel := sl.GetMaxStats()
	head := sl.GetHead()
	if head != nil {
		dfs(head, maxLevel, 0)
	}

	// 返回平均步數
	if totalProbability > 0 {
		return totalExpectedSteps / totalProbability, step
	}
	return 0.0, step
}

// PrintSkipList 打印 skip list 的結構
func PrintSkipList(sl skiplist.Analyable, maxLevel, maxNodes int) {
	_, actualMaxLevel := sl.GetMaxStats()
	maxLevel = min(maxLevel, actualMaxLevel)
	output := make([]string, maxLevel+1)

	for i := maxLevel; i >= 0; i-- {
		output[i] = fmt.Sprintf("level %d : ", i)
	}

	node := sl.GetHead()
	if node == nil {
		fmt.Println("Skip list 為空")
		return
	}

	count := 0
	for ; node != nil && count < maxNodes; count++ {
		lv := int(node.GetLevel())
		for i := range output {
			if i <= lv {
				output[i] += fmt.Sprintf("%3d ->", node.GetKey())
			} else {
				output[i] += "    ->"
			}
		}
		nextNode := node.GetNextAt(0)
		if nextNode != nil {
			node = nextNode
		} else {
			break
		}
	}

	for i := maxLevel; i >= 0; i-- {
		fmt.Println(output[i])
	}
}

// PrintSkipListToCSV 將 skip list 的結構輸出到 CSV
func PrintSkipListToCSV(sl skiplist.Analyable, maxLevel, maxNodes int, writer *csv.Writer) {
	_, actualMaxLevel := sl.GetMaxStats()
	maxLevel = min(maxLevel, actualMaxLevel)

	// 遍歷所有節點，建立一個包含所有 key 的排序列表
	var allKeys []skiplist.K
	node := sl.GetHead()
	for node != nil {
		allKeys = append(allKeys, node.GetKey())
		node = node.GetNextAt(0)
	}

	if len(allKeys) > maxNodes {
		allKeys = allKeys[:maxNodes]
	}

	for i := maxLevel; i >= 0; i-- {
		row := []string{fmt.Sprintf("level %d", i)}
		currentKeyIndex := 0
		node := sl.GetHead()
		for node != nil && currentKeyIndex < len(allKeys) {
			if node.GetKey() == allKeys[currentKeyIndex] {
				if int(node.GetLevel()) >= i {
					row = append(row, fmt.Sprintf("%d", node.GetKey()))
				} else {
					row = append(row, "")
				}
				node = node.GetNextAt(0)
				currentKeyIndex++
			} else if node.GetKey() < allKeys[currentKeyIndex] {
				node = node.GetNextAt(0)
			} else {
				// This case should ideally not happen if allKeys are from the skiplist
				row = append(row, "")
				currentKeyIndex++
			}
		}
		writer.Write(row)
	}
}

// PrintLink 打印 skip list 的連結結構
func PrintLink(sl skiplist.Analyable, maxLevel, maxNodes int) {
	head := sl.GetHead()
	if head == nil {
		fmt.Println("Skip list 為空")
		return
	}

	maxLevel = min(maxLevel, int(head.GetLevel()))

	for i := maxLevel; i >= 0; i-- {
		if int(head.GetLevel()) < i {
			continue
		}
		fmt.Printf("level %d : ", i)
		node := head
		count := 0
		for node != nil && count < maxNodes {
			fmt.Printf("%d ->", node.GetKey())
			nextNode := node.GetNextAt(int32(i))
			if nextNode != nil {
				node = nextNode
			} else {
				break
			}
			count++
		}
		fmt.Println()
	}
}

// CheckStruct 檢查 skip list 的結構是否正確
func CheckStruct(sl skiplist.Analyable) bool {
	_, maxLevel := sl.GetMaxStats()
	list := make([]skiplist.Nodelike, maxLevel+1)

	node := sl.GetHead()
	if node == nil {
		return true
	}

	for i := range list {
		list[i] = node
	}

	nextNode := node.GetNextAt(0)
	if nextNode != nil {
		node = nextNode
	} else {
		return true
	}

	for node != nil {
		nodelv := node.GetLevel()
		if nodelv > int32(maxLevel) {
			fmt.Printf("nodelv > level, nodelv: %d, level: %d\n", nodelv, maxLevel)
			return false
		}
		for i := 1; i <= int(nodelv); i++ {
			if i < len(list) {
				nextAtLevel := list[i].GetNextAt(int32(i))
				if nextAtLevel != node {
					fmt.Printf("list[%d] != node, list[%d]: %d, node: %d\n", i, i, nextAtLevel.GetKey(), node.GetKey())
					return false
				}
				list[i] = node
			}
		}
		nextNode := node.GetNextAt(0)
		if nextNode != nil {
			node = nextNode
		} else {
			break
		}
	}

	return true
}

// min 輔助函數
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func (mp StepMap) Print() {
	out := make([][2]int, 0)
	for k, v := range mp {
		out = append(out, [2]int{int(k), v})
	}

	sort.Slice(out, func(i, j int) bool {
		return out[i][0] < out[j][0]
	})

	for _, v := range out {
		fmt.Printf("%2d  ", v[0])
	}
	fmt.Println()
	for _, v := range out {
		fmt.Printf("%2d  ", v[1])
	}
}

func (mp StepMap) PrintToCSV(writer *csv.Writer) {
	out := make([][2]int, 0)
	for k, v := range mp {
		out = append(out, [2]int{int(k), v})
	}

	sort.Slice(out, func(i, j int) bool {
		return out[i][0] < out[j][0]
	})
	steps := make([]string, len(out)+2)

	for i, v := range out {
		steps[i+2] = fmt.Sprintf("%d", v[1])
	}

	writer.Write(steps)
}

func CountLevel(sl skiplist.Analyable) []int {
	maxNodes, maxLevel := sl.GetMaxStats()
	levelCounts := make([]int, maxLevel)

	// 遍歷所有節點，計算每層的節點數量
	head := sl.GetHead()
	current := head.GetNextAt(0) // 從第一個實際節點開始（跳過head）

	for current != nil {
		nodeLevel := current.GetLevel()
		// 該節點存在於level 0 到 nodeLevel-1 的所有層
		for i := int32(0); i <= nodeLevel; i++ {
			if int(i) < len(levelCounts) {
				levelCounts[i]++
			}
		}
		current = current.GetNextAt(0)
	}

	// 印出每層的節點數量
	fmt.Printf("層級節點統計 (總節點數: %d, 最高層級: %d):\n", maxNodes, maxLevel)
	for i := maxLevel - 1; i >= 0; i-- {
		fmt.Printf("Level %2d: %d 個節點\n", i, levelCounts[i])
	}

	return levelCounts
}

func BetterPrintSkipListToCSV(sl skiplist.Analyable, writer *csv.Writer) {
	_, actualMaxLevel := sl.GetMaxStats()

	outstr := make([][]string, actualMaxLevel+1)
	var dfs func(node skiplist.Nodelike, level int)
	dfs = func(node skiplist.Nodelike, level int) {
		if node == nil {
			return
		}
		if node.GetLevel() == int32(level) {
			for i := range outstr {
				if i <= level {
					outstr[i] = append(outstr[i], fmt.Sprintf("%d", node.GetKey()))
				} else {
					outstr[i] = append(outstr[i], "")
				}
			}
		}
		if level > 0 {
			dfs(node, level-1)
		}
		nextNode := node.GetNextAt(int32(level))
		if nextNode != nil && nextNode.GetLevel() == int32(level) {
			dfs(nextNode, level)
		}
	}
	head := sl.GetHead()
	if head != nil {
		dfs(head, actualMaxLevel)
	}

	for i := len(outstr) - 1; i >= 0; i-- {
		row := make([]string, len(outstr[i])+1)
		row[0] = fmt.Sprintf("level %d", i)
		copy(row[1:], outstr[i])
		writer.Write(row)
	}
}
