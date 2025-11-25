package analyTool

import (
	"encoding/csv"
	"fmt"
	"strconv"

	"github.com/Hakuto4838/SkipList.git/skiplist"
)

var LOGIC0 = 0.0000001

func DominationToCSV(csvWriter *csv.Writer, sl skiplist.Analyable) error {
	maxNodes, maxLevel := sl.GetMaxStats()
	// 先以 level 為第一維度，再以 node 為第二維度
	domin := make([][]float64, maxLevel+1)
	for i := 0; i <= maxLevel; i++ {
		domin[i] = make([]float64, maxNodes+1)
	}

	var countDomain func(nd skiplist.Nodelike, level int) float64
	countDomain = func(nd skiplist.Nodelike, level int) float64 {
		result := float64(0)
		next := nd.GetNextAt(int32(level))
		if next != nil {
			nextlvl := int(next.GetLevel())
			if nextlvl == level {
				result += countDomain(next, level)
			}
		}

		if level > 0 {
			result += countDomain(nd, level-1)
		}
		if level == int(nd.GetLevel()) {
			result += float64(nd.GetValue())
		}

		if result == 0 {
			result = LOGIC0
		}

		if int(nd.GetKey()) != -1 {
			domin[level][int(nd.GetKey())+1] = result
		} else {
			domin[level][0] = result
		}
		return result
	}
	countDomain(sl.GetHead(), maxLevel)
	printProbToCSV(csvWriter, domin)
	printRightProbCSV(csvWriter, domin)
	return nil
}

func printProbToCSV(csvWriter *csv.Writer, domin [][]float64) error {
	n := len(domin)
	for i := n - 1; i >= 0; i-- {
		strRow := make([]string, len(domin[i])+1)
		strRow[0] = fmt.Sprintf("level %d", i)
		for j := 0; j < len(domin[i]); j++ {
			v := domin[i][j]
			if v == 0 {
				strRow[j+1] = ""
			} else {
				strRow[j+1] = strconv.FormatFloat(v, 'f', 4, 64)
			}
		}
		csvWriter.Write(strRow)
	}
	csvWriter.Write([]string{"", "", "", "", "", "", ""})
	return nil
}

func printRightProbCSV(csvWriter *csv.Writer, domin [][]float64) error {
	level := len(domin)
	nodes := len(domin[0])

	getLevel := func(node int) int {
		lvl := 0
		for lvl < level && domin[lvl][node] != 0 {
			lvl++
		}
		return lvl - 1
	}
	for i := level - 1; i >= 0; i-- {
		for j := 0; j < nodes; j++ {
			if domin[i][j] == 0 {
				continue
			}
			var right float64
			var down float64
			k := j + 1
			for k < nodes && domin[i][k] == 0 {
				k++
			}
			if k < nodes && getLevel(k) == i {
				right = domin[i][k]
			} else {
				right = LOGIC0
			}

			if i > 0 {
				down = domin[i-1][j]
			} else {
				down = 0
			}

			if right == LOGIC0 {
				domin[i][j] = LOGIC0
			} else {
				domin[i][j] = right / (down + right)
			}
		}
	}
	return printProbToCSV(csvWriter, domin)
}
