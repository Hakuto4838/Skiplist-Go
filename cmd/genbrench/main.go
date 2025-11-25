package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/Hakuto4838/SkipList.git/datastream"
)

// parseScientificNotation 解析科學記號字串（如 "1e5"）為整數
func parseScientificNotation(s string) (int, error) {
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, err
	}
	return int(f), nil
}

// formatScientific 將數字格式化為科學記號（用於檔名）
func formatScientific(n int) string {
	if n == 0 {
		return "0"
	}

	// 找出最大的 10 的冪次
	exp := 0
	temp := n
	for temp >= 10 {
		temp /= 10
		exp++
	}

	// 計算係數
	divisor := 1
	for i := 0; i < exp; i++ {
		divisor *= 10
	}
	coefficient := float64(n) / float64(divisor)

	// 如果係數是整數，就不顯示小數
	if coefficient == float64(int(coefficient)) {
		return fmt.Sprintf("%de%d", int(coefficient), exp)
	}
	return fmt.Sprintf("%.1fe%d", coefficient, exp)
}

// formatDecimal 將浮點數格式化為不含小數點的字串（用於檔名）
func formatDecimal(f float64) string {
	// 乘以 100 轉換為整數（保留兩位小數的精度）
	val := int(f * 100)
	if val%100 == 0 {
		// 如果是整數，直接返回
		return fmt.Sprintf("%d", val/100)
	} else if val%10 == 0 {
		// 如果只有一位小數，返回 X_Y 格式
		return fmt.Sprintf("%d_%d", val/100, (val%100)/10)
	} else {
		// 兩位小數，返回 X_YZ 格式
		return fmt.Sprintf("%d_%02d", val/100, val%100)
	}
}

func main() {
	var out string
	var path string
	var nStr string
	var a float64
	var b float64
	var kStr string
	var seed int64
	var phase1Ratio float64
	var deleteRatio float64
	var nums int
	var eazy bool

	flag.StringVar(&nStr, "n", "0", "number of keys for Zipf generator (支援科學記號，如 1e5)")
	flag.Float64Var(&a, "a", 1.07, "Zipf parameter a (設為 0 時使用均勻分布)")
	flag.Float64Var(&b, "b", 0.0, "Zipf parameter b (當 a > 0 時有效)")
	flag.StringVar(&kStr, "k", "0", "number of operations to generate (支援科學記號，如 1e6)")
	flag.Int64Var(&seed, "seed", time.Now().UnixNano(), "seed for generators/structures where applicable")
	flag.Float64Var(&phase1Ratio, "phase1Ratio", 0.5, "ratio of phase1 operations")
	flag.Float64Var(&deleteRatio, "deleteRatio", 0.1, "ratio of delete operations")
	flag.IntVar(&nums, "nums", 1, "number of files to generate")
	flag.StringVar(&out, "out", "", "output filename prefix (留空則自動生成)")
	flag.StringVar(&path, "path", ".", "output directory path (輸出目錄路徑)")
	flag.BoolVar(&eazy, "eazy", false, "是否使用簡單模式")
	flag.Parse()

	// 解析科學記號
	n, err := parseScientificNotation(nStr)
	if err != nil {
		fmt.Printf("解析參數 n 錯誤: %v\n", err)
		return
	}

	k, err := parseScientificNotation(kStr)
	if err != nil {
		fmt.Printf("解析參數 k 錯誤: %v\n", err)
		return
	}

	// 如果沒有指定輸出檔名，則根據參數自動生成
	if out == "" {
		out = fmt.Sprintf("bench_n%s_k%s_a%s_b%s_p1r%s_dr%s",
			formatScientific(n),
			formatScientific(k),
			formatDecimal(a),
			formatDecimal(b),
			formatDecimal(phase1Ratio),
			formatDecimal(deleteRatio))
	}

	// 確保輸出目錄存在
	if path != "." && path != "" {
		if err := os.MkdirAll(path, 0755); err != nil {
			fmt.Printf("建立輸出目錄失敗: %v\n", err)
			return
		}
	}

	fmt.Printf("生成參數:\n")
	fmt.Printf("  n (keys): %d\n", n)
	fmt.Printf("  k (operations): %d\n", k)
	fmt.Printf("  a: %.2f\n", a)
	fmt.Printf("  b: %.2f\n", b)
	fmt.Printf("  phase1Ratio: %.2f\n", phase1Ratio)
	fmt.Printf("  deleteRatio: %.2f\n", deleteRatio)
	fmt.Printf("  seed: %d\n", seed)
	fmt.Printf("  檔案數量: %d\n", nums)
	fmt.Printf("  輸出目錄: %s\n", path)
	fmt.Printf("  輸出檔名前綴: %s\n\n", out)

	for i := 0; i < nums; i++ {
		var filename string
		if nums == 1 {
			filename = fmt.Sprintf("%s.bin", out)
		} else {
			filename = fmt.Sprintf("%s_%d.bin", out, i)
		}
		outfile := filepath.Join(path, filename)
		fmt.Printf("正在生成 %s...\n", outfile)
		_, err := datastream.WriteBenchFileFromZipfV2(n, a, b, uint64(seed+int64(i)), k, phase1Ratio, deleteRatio, outfile, eazy)
		if err != nil {
			fmt.Printf("錯誤: %v\n", err)
			return
		}
	}
	fmt.Println("完成!")
}
