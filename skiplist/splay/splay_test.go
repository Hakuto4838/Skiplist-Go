package splay

import (
	"encoding/csv"
	"fmt"
	"os"
	"testing"

	"github.com/Hakuto4838/SkipList.git/datastream"
	"github.com/Hakuto4838/SkipList.git/skiplist"
	"github.com/Hakuto4838/SkipList.git/skiplist/analyTool"
	"github.com/Hakuto4838/SkipList.git/skiplist/basic"
)

func TestSplaySkipList(t *testing.T) {
	sl := NewSplayList(1)

	// 測試基本操作
	sl.Put(1, 100)
	sl.Put(2, 200)
	sl.Put(3, 300)

	analyTool.PrintSkipList(sl, 5, 10)

	// 測試 Get
	if value, found := sl.Get(1); !found || value != 100 {
		t.Errorf("Get(1) = (%f, %v), want (100, true)", value, found)
	}

	// 測試 Contains
	if !sl.Contains(2) {
		t.Error("Contains(2) = false, want true")
	}

	// 測試 Delete
	sl.Delete(2)
	if sl.Contains(2) {
		t.Error("Contains(2) = true after delete, want false")
	}
}

func TestSplayListPrint(t *testing.T) {
	sl := NewSplayList(0.5)
	for i := 0; i < 15; i++ {
		sl.Put(skiplist.K(i), skiplist.V(i))
	}

	// data := datastream.NewZipfDataGenerator(15, 1.5, 1, 42)
	// for iter := 0; iter < 10000; iter++ {
	// 	sl.Get(skiplist.K(data.Next()))
	// }
	analyTool.PrintSkipList(sl, 5, 20)
	analyTool.CountLevel(sl)
}

func TestSplayAnalysis(t *testing.T) {
	data := datastream.NewZipfDataGenerator(15, 1.5, 1, 42)
	sl := NewSplayList(0.1)

	for k := range data.GetDistribute() {
		sl.Put(skiplist.K(k), skiplist.V(123))
	}

	for range 10000 {
		sl.Get(skiplist.K(data.Next()))
	}

	analyTool.PrintSkipList(sl, 5, 10)

	keymap := make(map[skiplist.K]float64)
	for k, v := range data.GetDistribute() {
		keymap[skiplist.K(k)] = v
	}
	score, pstep := analyTool.AnalyzeStep(sl, keymap)
	fmt.Printf("score: %f\n", score)

	pstep.Print()

}

func TestSplayListCSV(t *testing.T) {
	sl := NewSplayList(0.01)
	const N = 20
	for i := 0; i < N; i++ {
		if i%10000 == 0 {
			fmt.Printf("insert %d\n", i)
		}
		sl.Put(skiplist.K(i), skiplist.V(i))
	}

	data := datastream.NewZipfDataGenerator(N, 1.5, 1, 42)
	for iter := 0; iter < 10000; iter++ {
		sl.Get(skiplist.K(data.Next()))
	}
	analyTool.PrintSkipList(sl, 20, 15)

	analyTool.CountLevel(sl)
	// sl.PrintHits()

	// 測試 PrintSkipListToCSV
	file, err := os.Create("splay_skiplist.csv")
	if err != nil {
		t.Fatalf("無法建立檔案: %s", err)
	}
	defer file.Close()
	writer := csv.NewWriter(file)
	analyTool.PrintSkipListToCSV(sl, 5, 10, writer)
	writer.Flush()

	// 測試 stepMap.PrintToCSV
	keymap := make(map[skiplist.K]float64)
	for k, v := range data.GetDistribute() {
		keymap[skiplist.K(k)] = v
	}
	_, pstep := analyTool.AnalyzeStep(sl, keymap)

	file2, err := os.Create("splay_stepmap.csv")
	if err != nil {
		t.Fatalf("無法建立檔案: %s", err)
	}
	defer file2.Close()
	writer2 := csv.NewWriter(file2)
	pstep.PrintToCSV(writer2)
	writer2.Flush()
}

func TestSplayAndBasicSeq(t *testing.T) {
	// 用來檢查 splay 與 basic 差異
	const N = 100000

	dataGen := datastream.NewZipfDataGenerator(N, 1.5, 1, 42)
	basicSL := basic.NewBasicSkipList(42)
	splaySL := NewSplayList(1)

	keymap := dataGen.GetKeyMap()

	for k, v := range keymap {
		basicSL.Put(k, v)
		splaySL.Put(k, v)
	}

	//training
	seq := dataGen.GenerateSequence(N * 10)
	for _, k := range seq {
		splaySL.Get(skiplist.K(k))
	}

	fmt.Println("basic structure")
	analyTool.PrintSkipList(basicSL, 8, 15)
	scorebasic, _ := analyTool.AnalyzeStep(basicSL, keymap)
	fmt.Printf("basic score: %f\n\n", scorebasic)

	fmt.Println("splay structure")
	analyTool.PrintSkipList(splaySL, 8, 15)
	scoresplay, _ := analyTool.AnalyzeStep(splaySL, keymap)
	fmt.Printf("splay score: %f\n\n", scoresplay)
}
