package main

import (
	"fmt"

	"github.com/Hakuto4838/SkipList.git/datastream"
	"github.com/Hakuto4838/SkipList.git/skiplist"
	"github.com/Hakuto4838/SkipList.git/skiplist/analyTool"
	"github.com/Hakuto4838/SkipList.git/skiplist/basic"
	"github.com/Hakuto4838/SkipList.git/skiplist/la"
	"github.com/Hakuto4838/SkipList.git/skiplist/rebuildsl"
	"github.com/Hakuto4838/SkipList.git/skiplist/splay"
)

func insertSequential(sl skiplist.SkipList, gen *datastream.ZipfDataGenerator) {
	for k, v := range gen.GetKeyMap() {
		sl.Put(k, v)
	}
}

func testOne(name string, sl skiplist.SkipList, kmap map[skiplist.K]float64) {
	fmt.Printf("=== %s ===\n", name)
	// analyTool.CountLevel(sl.(skiplist.Analyable))
	score, _ := analyTool.AnalyzeStep(sl.(skiplist.Analyable), kmap)
	fmt.Printf("score: %.6f\n\n", score)
	analyTool.PrintSkipList(sl.(skiplist.Analyable), 8, 35)
}

func main() {
	const n = 900
	const seed = 42

	// Zipf distribution for analysis
	gen := datastream.NewZipfDataGenerator(n, 1.07, 1.0, seed)
	kmap := gen.GetKeyMap()

	// Basic
	basicSL := basic.NewBasicSkipList(seed)
	insertSequential(basicSL, gen)
	testOne("basic", basicSL, kmap)

	// Splay, 先插入再進行 training 查詢
	splaySL := splay.NewSplayList(1)
	insertSequential(splaySL, gen)

	// 執行隨機查詢作為 training（Zipf 模擬熱點）
	seq := gen.GenerateSequence(n * 10) // 20 倍查詢量
	for _, idx := range seq {
		splaySL.Get(skiplist.K(idx))
	}

	testOne("splay", splaySL, kmap)

	// LA SkipList
	laSL := la.NewLASkipList(seed)
	// 依 Zipf 權重插入並帶入預估頻率 np=n*prob
	for k, prob := range kmap {
		np := float64(n) * prob
		laSL.PutWithNP(k, prob*float64(n), np)
	}
	testOne("la", laSL, kmap)

	// Rebuild
	rebuildSL := rebuildsl.NewRebuildSLList(0.1)
	insertSequential(rebuildSL, gen)
	for _, idx := range seq {
		rebuildSL.Get(skiplist.K(idx))
	}
	rebuildSL.ForceBalance(32)
	testOne("rebuild", rebuildSL, kmap)

}
