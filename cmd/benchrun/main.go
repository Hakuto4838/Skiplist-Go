package main

import (
	"flag"
	"fmt"
	"log"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/Hakuto4838/SkipList.git/datastream"
	"github.com/Hakuto4838/SkipList.git/skiplist"
	"github.com/Hakuto4838/SkipList.git/skiplist/analyTool"
	"github.com/Hakuto4838/SkipList.git/skiplist/basic"
	"github.com/Hakuto4838/SkipList.git/skiplist/la"
	"github.com/Hakuto4838/SkipList.git/skiplist/splay"
	"github.com/olekukonko/tablewriter"
)

type laPutWithNP interface {
	PutWithNP(key skiplist.K, value skiplist.V, np float64)
}

func main() {
	// Input: either provide -file, -dir, or provide -out and generation params
	var file string
	var dir string
	var out string
	var n int
	var a float64
	var b float64
	var k int
	var seed int64

	var impls string
	var runs int
	var splayP float64
	var rebuildP float64
	var phase1Ratio float64
	var deleteRatio float64

	flag.StringVar(&file, "file", "", "existing bench streamfile (SLBENCH1 format)")
	flag.StringVar(&dir, "dir", "", "directory containing bench files to test (will test all .bin files)")
	flag.StringVar(&out, "out", "", "output path to write generated bench streamfile")
	flag.IntVar(&n, "n", 0, "number of keys for Zipf generator")
	flag.Float64Var(&a, "a", 1.07, "Zipf parameter a")
	flag.Float64Var(&b, "b", 0.0, "Zipf parameter b")
	flag.IntVar(&k, "k", 0, "number of operations to generate")
	flag.Int64Var(&seed, "seed", time.Now().UnixNano(), "seed for generators/structures where applicable")
	flag.Float64Var(&phase1Ratio, "phase1Ratio", 0.5, "ratio of phase1 operations")
	flag.Float64Var(&deleteRatio, "deleteRatio", 0.1, "ratio of delete operations")

	flag.StringVar(&impls, "impl", "all", "implementations to run: all or comma list (basic,splay,la,rebuild,gravity,falldown)")
	flag.IntVar(&runs, "runs", 5, "how many times to repeat each benchmark")
	flag.Float64Var(&splayP, "splay.p", 0.01, "probability for splay updates")
	flag.Float64Var(&rebuildP, "rebuild.p", 0.1, "probability for rebuild balancing")
	flag.Parse()

	var benchPaths []string

	// 判斷模式: -dir 優先於 -file
	if dir != "" {
		// 掃描目錄中所有 .bin 檔案
		files, err := collectBenchFilesFromDir(dir)
		if err != nil {
			log.Fatalf("scan directory %s: %v", dir, err)
		}
		if len(files) == 0 {
			log.Fatalf("no .bin files found in directory: %s", dir)
		}
		benchPaths = files
		fmt.Printf("Found %d bench files in directory: %s\n", len(benchPaths), dir)
	} else if file != "" {
		benchPaths = []string{file}
		fmt.Printf("bench_file: %s\n", file)
	} else {
		// validate generation inputs
		if out == "" {
			log.Fatalf("either -file, -dir, or -out with generation params (-n,-a,-b,-k,-seed) must be provided")
		}
		if n <= 0 || k < 0 {
			log.Fatalf("invalid -n or -k: n=%d k=%d", n, k)
		}
		fmt.Printf("generated bench_file: %s\n", out)
		if _, err := datastream.WriteBenchFileFromZipfV2(n, a, b, uint64(seed), k, phase1Ratio, deleteRatio, out, false); err != nil {
			log.Fatalf("generate bench file: %v", err)
		}
		benchPaths = []string{out}
	}

	toRun := parseImpls(impls)
	fmt.Printf("implementations to test: %s\n", strings.Join(toRun, ","))
	fmt.Println(strings.Repeat("=", 80))

	// 如果是多個檔案，匯總統計
	if len(benchPaths) > 1 {
		runBatchBenchmark(benchPaths, toRun, runs, seed, splayP, rebuildP)
	} else {
		// 單一檔案，顯示詳細結果
		runBenchmark(benchPaths[0], toRun, runs, seed, splayP, rebuildP)
	}
}

// collectBenchFilesFromDir 收集指定目錄下所有 .bin 檔案
func collectBenchFilesFromDir(dir string) ([]string, error) {
	var files []string
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && filepath.Ext(path) == ".bin" {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	// 排序檔案名稱以確保順序一致
	sort.Strings(files)
	return files, nil
}

// runBatchBenchmark 對多個 benchmark 檔案執行測試並匯總統計
func runBatchBenchmark(benchPaths []string, toRun []string, runs int, seed int64, splayP, rebuildP float64) {
	fmt.Printf("Testing %d benchmark files...\n\n", len(benchPaths))

	// 為每個實作方式收集所有檔案的統計數據
	type implStats struct {
		avgMsList []float64
		minMsList []float64
		maxMsList []float64
		opsList   []int
		stepsList []float64
		totalRuns int
	}

	allStats := make(map[string]*implStats)
	for _, impl := range toRun {
		allStats[impl] = &implStats{
			avgMsList: make([]float64, 0, len(benchPaths)),
			minMsList: make([]float64, 0, len(benchPaths)),
			maxMsList: make([]float64, 0, len(benchPaths)),
			opsList:   make([]int, 0, len(benchPaths)),
			stepsList: make([]float64, 0, len(benchPaths)),
		}
	}

	// 對每個 benchmark 檔案執行測試
	for idx, benchPath := range benchPaths {
		fmt.Printf("[%d/%d] Testing: %s\n", idx+1, len(benchPaths), filepath.Base(benchPath))

		bf, err := datastream.ReadBenchFile(benchPath)
		if err != nil {
			log.Printf("  ERROR reading bench file: %v\n", err)
			continue
		}

		fmt.Printf("  ops: %d, entropy: %.6f\n", len(bf.Ops), computeEntropy(bf.Dist))

		for _, impl := range toRun {
			fmt.Printf("  - benchmarking %s...\n", impl)
			stats := benchmarkImpl(bf, impl, runs, seed, splayP, rebuildP)

			allStats[impl].avgMsList = append(allStats[impl].avgMsList, stats.avgMs)
			allStats[impl].minMsList = append(allStats[impl].minMsList, stats.minMs)
			allStats[impl].maxMsList = append(allStats[impl].maxMsList, stats.maxMs)
			allStats[impl].opsList = append(allStats[impl].opsList, len(bf.Ops))
			if !math.IsNaN(stats.avgSteps) {
				allStats[impl].stepsList = append(allStats[impl].stepsList, stats.avgSteps)
			}
			allStats[impl].totalRuns += runs
		}
		fmt.Println()
	}

	// 計算並顯示匯總統計
	fmt.Println(strings.Repeat("=", 80))
	fmt.Println("AGGREGATE STATISTICS (across all benchmark files)")
	fmt.Println(strings.Repeat("=", 80))

	rows := make([][]string, 0, len(toRun))
	for _, impl := range toRun {
		stats := allStats[impl]
		if len(stats.avgMsList) == 0 {
			continue
		}

		// 計算平均值
		avgMs := average(stats.avgMsList)
		minMs := min(stats.minMsList)
		maxMs := max(stats.maxMsList)

		// 計算平均 ops/s
		totalOps := 0
		totalSec := 0.0
		for i, ops := range stats.opsList {
			totalOps += ops
			totalSec += stats.avgMsList[i] / 1000.0
		}
		avgThr := float64(totalOps) / totalSec

		// 計算平均 steps
		steps := "N/A"
		if len(stats.stepsList) > 0 {
			steps = fmt.Sprintf("%.6f", average(stats.stepsList))
		}

		rows = append(rows, []string{
			impl,
			fmt.Sprintf("%d", stats.totalRuns),
			fmt.Sprintf("%.3f", avgMs),
			fmt.Sprintf("%.3f", minMs),
			fmt.Sprintf("%.3f", maxMs),
			fmt.Sprintf("%.2f", avgThr),
			steps,
		})
	}

	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Impl", "Total Runs", "Avg(ms)", "Min(ms)", "Max(ms)", "Avg Ops/s", "AvgSteps"})
	table.SetAlignment(tablewriter.ALIGN_CENTER)
	table.SetAutoWrapText(false)
	table.AppendBulk(rows)
	table.Render()
}

// runBenchmark 執行單一 benchmark 檔案的測試
func runBenchmark(benchPath string, toRun []string, runs int, seed int64, splayP, rebuildP float64) {
	bf, err := datastream.ReadBenchFile(benchPath)
	if err != nil {
		log.Printf("ERROR reading bench file %s: %v", benchPath, err)
		return
	}

	fmt.Printf("bench_file: %s\n", benchPath)
	fmt.Printf("ops: %d\n", len(bf.Ops))
	fmt.Printf("entropy: %.6f\n", computeEntropy(bf.Dist))

	rows := make([][]string, 0, len(toRun))
	for _, impl := range toRun {
		fmt.Printf("benchmarking %s...\n", impl)
		stats := benchmarkImpl(bf, impl, runs, seed, splayP, rebuildP)
		thr := float64(len(bf.Ops)) / (stats.avgMs / 1000.0)
		steps := "N/A"
		if !math.IsNaN(stats.avgSteps) {
			steps = fmt.Sprintf("%.6f", stats.avgSteps)
		}
		rows = append(rows, []string{
			impl,
			fmt.Sprintf("%d", runs),
			fmt.Sprintf("%.3f", stats.avgMs),
			fmt.Sprintf("%.3f", stats.minMs),
			fmt.Sprintf("%.3f", stats.maxMs),
			fmt.Sprintf("%.2f", thr),
			steps,
		})
	}

	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Impl", "Runs", "Avg(ms)", "Min(ms)", "Max(ms)", "Ops/s", "AvgSteps"})
	table.SetAlignment(tablewriter.ALIGN_CENTER)
	table.SetAutoWrapText(false)
	table.AppendBulk(rows)
	table.Render()
}

// 輔助函數：計算平均值
func average(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

// 輔助函數：找最小值
func min(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	m := values[0]
	for _, v := range values[1:] {
		if v < m {
			m = v
		}
	}
	return m
}

// 輔助函數：找最大值
func max(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	m := values[0]
	for _, v := range values[1:] {
		if v > m {
			m = v
		}
	}
	return m
}

type benchStats struct {
	avgMs    float64
	minMs    float64
	maxMs    float64
	avgSteps float64 // from one run (structure-dependent), NaN if not analyzable
}

func benchmarkImpl(bf *datastream.BenchFile, impl string, runs int, seed int64, splayP, rebuildP float64) benchStats {
	durations := make([]float64, 0, runs)
	var sampleSteps = math.NaN()
	for i := 0; i < runs; i++ {
		// fmt.Printf("running %s %d\n", impl, i)
		sl := newImpl(impl, seed, splayP, rebuildP)
		elapsed := runOpsAndTime(sl, bf)
		durations = append(durations, float64(elapsed.Microseconds())/1000.0)
		if math.IsNaN(sampleSteps) {
			if analy, ok := sl.(skiplist.Analyable); ok {
				s, _ := analyTool.AnalyzeStep(analy, bf.Dist)
				sampleSteps = s
			}
		}
	}
	sort.Float64s(durations)
	sum := 0.0
	for _, v := range durations {
		sum += v
	}
	avg := sum / float64(len(durations))
	return benchStats{
		avgMs:    avg,
		minMs:    durations[0],
		maxMs:    durations[len(durations)-1],
		avgSteps: sampleSteps,
	}
}

func newImpl(impl string, seed int64, splayP, rebuildP float64) skiplist.SkipList {
	switch impl {
	case "basic":
		return basic.NewBasicSkipList(seed)
	case "splay":
		return splay.NewSplayList(splayP)
	case "la":
		return la.NewLASkipList(seed)
	// case "rebuild":
	// 	return rebuildsl.NewRebuildSLList(rebuildP)
	// case "gravity":
	// 	return gravity.NewGravityList()
	// case "falldown":
	// 	return falldown.NewFdList()
	default:
		log.Fatalf("unknown -impl: %s", impl)
		return nil
	}
}

func runOpsAndTime(sl skiplist.SkipList, bf *datastream.BenchFile) time.Duration {
	// 預先決定插入策略，避免每次操作都做類型斷言
	insertFunc := func(key skiplist.K) {
		val := bf.Dist[key]
		sl.Put(key, skiplist.V(val))
	}
	if laSl, ok := sl.(laPutWithNP); ok {
		n := float64(len(bf.Dist))
		insertFunc = func(key skiplist.K) {
			val := bf.Dist[key]
			laSl.PutWithNP(key, skiplist.V(val), val*n)
		}
	}

	start := time.Now()
	for _, op := range bf.Ops {
		switch op.Type {
		case datastream.OpQuery:
			sl.Get(op.Key)
		case datastream.OpInsert:
			insertFunc(op.Key)
		case datastream.OpDelete:
			sl.Delete(op.Key)
		}
	}
	return time.Since(start)
}

func parseImpls(s string) []string {
	if s == "" || s == "all" {
		return []string{"basic", "splay", "la"}
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	seen := map[string]bool{}
	for _, p := range parts {
		t := strings.TrimSpace(strings.ToLower(p))
		if t == "" || seen[t] {
			continue
		}
		switch t {
		case "basic", "splay", "la", "rebuild", "gravity", "falldown":
			out = append(out, t)
			seen[t] = true
		}
	}
	if len(out) == 0 {
		return []string{"basic", "splay", "la"}
	}
	return out
}

func computeEntropy(m map[skiplist.K]float64) float64 {
	h := 0.0
	for _, p := range m {
		if p > 0 {
			h -= p * math.Log2(p)
		}
	}
	return h
}
