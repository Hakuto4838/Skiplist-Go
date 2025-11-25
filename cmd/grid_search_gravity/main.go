package main

import (
	"encoding/csv"
	"flag"
	"fmt"
	"log"
	"math"
	"os"
	"path/filepath"
	"time"

	"github.com/Hakuto4838/SkipList.git/datastream"
	"github.com/Hakuto4838/SkipList.git/skiplist"
	"github.com/Hakuto4838/SkipList.git/skiplist/gravity"
)

// evaluateCost 評估給定參數的執行時間成本
// 返回所有 benchmark 文件在所有運行中的總時間（取平均後的總和）
func evaluateCost(z, threshold float64, benchFiles []*datastream.BenchFile, runs int) float64 {
	var totalMs float64

	for _, bf := range benchFiles {
		var fileMs float64
		// 對每個檔案運行 runs 次並取平均
		for i := 0; i < runs; i++ {
			gl := gravity.NewGravityListWithParams(z, threshold)

			start := time.Now()
			for _, op := range bf.Ops {
				switch op.Type {
				case datastream.OpQuery:
					gl.Get(op.Key)
				case datastream.OpInsert:
					gl.Put(op.Key, skiplist.V(op.Key))
				case datastream.OpDelete:
					gl.Delete(op.Key)
				}
			}
			elapsed := time.Since(start)
			fileMs += float64(elapsed.Microseconds()) / 1000.0
		}
		// 累加每個檔案的平均時間（這樣可以找到對所有 benchmark 綜合表現最好的參數）
		totalMs += fileMs / float64(runs)
	}

	// 返回所有檔案的總時間（每個檔案取平均後加總）
	return totalMs
}

func main() {
	// 命令行參數
	var benchPath string
	var benchDir string
	var zMin, zMax, zStep float64
	var thMin, thMax, thStep float64
	var runs int
	var outputCSV string
	var outputFormat string

	flag.StringVar(&benchPath, "bench", "", "單一 benchmark 檔案路徑")
	flag.StringVar(&benchDir, "benchdir", "", "包含多個 benchmark 檔案的目錄 (使用所有 .bin 檔案)")
	flag.Float64Var(&zMin, "zmin", 0.5, "z 的最小值")
	flag.Float64Var(&zMax, "zmax", 3.0, "z 的最大值")
	flag.Float64Var(&zStep, "zstep", 0.25, "z 的步長")
	flag.Float64Var(&thMin, "thmin", 0.1, "tryThreshold 的最小值")
	flag.Float64Var(&thMax, "thmax", 0.9, "tryThreshold 的最大值")
	flag.Float64Var(&thStep, "thstep", 0.08, "tryThreshold 的步長")
	flag.IntVar(&runs, "runs", 3, "每組參數運行的次數（取平均值）")
	flag.StringVar(&outputCSV, "csv", "", "輸出 CSV 檔案路徑（選填，用於生成熱力圖）")
	flag.StringVar(&outputFormat, "format", "text", "輸出格式: text 或 csv-only")
	flag.Parse()

	// 收集 benchmark 檔案
	var benchFiles []string

	if benchDir != "" {
		// 使用目錄中的所有 .bin 檔案
		files, err := filepath.Glob(filepath.Join(benchDir, "*.bin"))
		if err != nil {
			log.Fatalf("掃描目錄失敗: %v", err)
		}
		if len(files) == 0 {
			log.Fatalf("目錄中找不到 .bin 檔案: %s", benchDir)
		}
		benchFiles = files
	} else if benchPath != "" {
		// 使用單一檔案
		benchFiles = []string{benchPath}
	} else {
		log.Fatal("請提供 -bench 或 -benchdir 參數")
	}

	// 讀取所有 benchmark 檔案
	loadedBenchmarks := make([]*datastream.BenchFile, 0, len(benchFiles))
	totalOps := 0

	if outputFormat != "csv-only" {
		fmt.Printf("=== Gravity SkipList 網格搜索參數優化 ===\n\n")
		fmt.Printf("載入 %d 個 benchmark 檔案...\n", len(benchFiles))
	}

	for _, fpath := range benchFiles {
		bf, err := datastream.ReadBenchFile(fpath)
		if err != nil {
			log.Fatalf("讀取 benchmark 檔案失敗 %s: %v", fpath, err)
		}
		loadedBenchmarks = append(loadedBenchmarks, bf)
		totalOps += len(bf.Ops)
		if outputFormat != "csv-only" {
			fmt.Printf("  - %s: %d 操作\n", filepath.Base(fpath), len(bf.Ops))
		}
	}

	if outputFormat != "csv-only" {
		fmt.Printf("\n總計: %d 檔案, %d 操作\n", len(loadedBenchmarks), totalOps)
		fmt.Printf("\n搜索範圍:\n")
		fmt.Printf("  z: [%.2f, %.2f], 步長: %.3f\n", zMin, zMax, zStep)
		fmt.Printf("  tryThreshold: [%.2f, %.2f], 步長: %.3f\n", thMin, thMax, thStep)
		fmt.Printf("  每組運行次數: %d\n", runs)
	}

	// 計算總測試數
	zCount := int(math.Round((zMax-zMin)/zStep)) + 1
	thCount := int(math.Round((thMax-thMin)/thStep)) + 1
	totalTests := zCount * thCount

	if outputFormat != "csv-only" {
		fmt.Printf("\n總測試組合數: %d × %d = %d\n", zCount, thCount, totalTests)
		estimatedMinutes := float64(totalTests) * 0.4 / 60.0 // 粗估每次 0.4 秒
		fmt.Printf("預估時間: %.1f 分鐘\n\n", estimatedMinutes)
	}

	// CSV 輸出準備
	var csvFile *os.File
	var csvWriter *csv.Writer
	if outputCSV != "" {
		var err error
		csvFile, err = os.Create(outputCSV)
		if err != nil {
			log.Fatalf("無法創建 CSV 檔案: %v", err)
		}
		defer csvFile.Close()

		csvWriter = csv.NewWriter(csvFile)
		defer csvWriter.Flush()

		// 寫入標頭
		csvWriter.Write([]string{"z", "tryThreshold", "cost_ms"})
	}

	// 執行網格搜索
	bestZ := zMin
	bestThreshold := thMin
	bestCost := math.Inf(1)
	testCount := 0

	startTime := time.Now()

	for z := zMin; z <= zMax+0.0001; z += zStep { // 加小量避免浮點誤差
		for th := thMin; th <= thMax+0.0001; th += thStep {
			testCount++
			cost := evaluateCost(z, th, loadedBenchmarks, runs)

			// CSV 輸出
			if csvWriter != nil {
				csvWriter.Write([]string{
					fmt.Sprintf("%.6f", z),
					fmt.Sprintf("%.6f", th),
					fmt.Sprintf("%.3f", cost),
				})
			}

			// 文字輸出
			if outputFormat != "csv-only" {
				progress := float64(testCount) / float64(totalTests) * 100.0
				fmt.Printf("[%6.2f%%] z=%.4f, th=%.4f → %7.3f ms", progress, z, th, cost)

				if cost < bestCost {
					fmt.Printf(" ✓ 新最佳!")
					bestZ = z
					bestThreshold = th
					bestCost = cost
				}
				fmt.Println()
			}

			// 更新最佳解
			if cost < bestCost {
				bestZ = z
				bestThreshold = th
				bestCost = cost
			}
		}
	}

	elapsed := time.Since(startTime)

	if outputFormat != "csv-only" {
		fmt.Printf("\n=== 搜索完成 ===\n")
		fmt.Printf("總耗時: %.2f 分鐘 (%.1f 秒)\n", elapsed.Minutes(), elapsed.Seconds())
		fmt.Printf("\n最佳參數（對所有 benchmark 綜合表現最佳）:\n")
		fmt.Printf("  z            = %.6f\n", bestZ)
		fmt.Printf("  tryThreshold = %.6f\n", bestThreshold)
		if len(loadedBenchmarks) > 1 {
			fmt.Printf("  總成本       = %.3f ms (所有 %d 個 benchmark 的總和)\n", bestCost, len(loadedBenchmarks))
			fmt.Printf("  平均成本     = %.3f ms (每個 benchmark)\n", bestCost/float64(len(loadedBenchmarks)))
		} else {
			fmt.Printf("  成本         = %.3f ms\n", bestCost)
		}

		// 與預設參數比較
		fmt.Printf("\n與預設參數 (z=1.8, tryThreshold=0.5) 比較:\n")
		defaultCost := evaluateCost(1.8, 0.5, loadedBenchmarks, runs)
		if len(loadedBenchmarks) > 1 {
			fmt.Printf("  預設總成本: %.3f ms (所有 %d 個 benchmark)\n", defaultCost, len(loadedBenchmarks))
		} else {
			fmt.Printf("  預設成本: %.3f ms\n", defaultCost)
		}
		improvement := (defaultCost - bestCost) / defaultCost * 100.0
		if improvement > 0 {
			fmt.Printf("  改善: %.2f%% ✓\n", improvement)
		} else {
			fmt.Printf("  變化: %.2f%%\n", improvement)
		}

		// 使用建議
		fmt.Printf("\n使用方式:\n")
		fmt.Printf("  在您的程式碼中使用:\n")
		fmt.Printf("    gl := gravity.NewGravityListWithParams(%.6f, %.6f)\n", bestZ, bestThreshold)

		if outputCSV != "" {
			fmt.Printf("\nCSV 結果已保存至: %s\n", outputCSV)
			fmt.Printf("可以使用 Python/Excel 生成熱力圖進行分析\n")
		}
	}
}

