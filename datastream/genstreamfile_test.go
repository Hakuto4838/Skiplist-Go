package datastream

import (
	"fmt"
	"math"
	"path/filepath"
	"testing"

	"github.com/Hakuto4838/SkipList.git/skiplist"
)

func TestWriteAndReadBenchFileFromZipf(t *testing.T) {
	n := 8
	a := 1.2
	b := 0.0
	seed := int64(42)
	k := 200

	gen := NewZipfDataGenerator(n, a, b, seed)
	if gen == nil {
		t.Fatalf("NewZipfDataGenerator returned nil")
	}

	tmp := t.TempDir()
	file := filepath.Join(tmp, "bench.bin")

	if err := WriteBenchFileFromZipf(gen, k, file); err != nil {
		t.Fatalf("WriteBenchFileFromZipf error: %v", err)
	}

	bf, err := ReadBenchFile(file)
	if err != nil {
		t.Fatalf("ReadBenchFile error: %v", err)
	}

	// 驗證分布 map
	exp := gen.GetKeyMap()
	if len(bf.Dist) != len(exp) {
		t.Fatalf("dist len mismatch: got %d, want %d", len(bf.Dist), len(exp))
	}
	for kexp, vexp := range exp {
		vgot, ok := bf.Dist[kexp]
		if !ok {
			t.Fatalf("missing key in dist: %v", kexp)
		}
		if !floatAlmostEqual(vgot, vexp, 1e-12) {
			t.Fatalf("weight mismatch for key %v: got %v, want %v", kexp, vgot, vexp)
		}
	}

	// 驗證操作序列
	if len(bf.Ops) != k {
		t.Fatalf("ops len mismatch: got %d, want %d", len(bf.Ops), k)
	}
	seen := map[int]bool{}
	for i, op := range bf.Ops {
		idx := int(op.Key)
		if !seen[idx] {
			if op.Type != OpInsert {
				t.Fatalf("op[%d] first occurrence must be Insert, got %v", i, op.Type)
			}
			seen[idx] = true
		} else {
			if !(op.Type == OpQuery || op.Type == OpInsert) {
				t.Fatalf("op[%d] must be Query or Insert after seen, got %v", i, op.Type)
			}
		}
	}

	// 驗證 ToSequenceModel
	m := bf.ToSequenceModel()
	count := 0
	for {
		op, ok := m.Next()
		if !ok {
			break
		}
		fmt.Println(op)
		count++
	}
	if count != k {
		t.Fatalf("sequence model length mismatch: got %d, want %d", count, k)
	}
}

func floatAlmostEqual(a, b, eps float64) bool {
	return math.Abs(a-b) <= eps
}

func TestWriteAndReadBenchFileFromZipfV2(t *testing.T) {
	n := 8
	s := 1.2
	v := 1.0
	var seed uint64 = 42
	k := 200

	tmp := t.TempDir()
	file := filepath.Join(tmp, "bench_v2.bin")

	phase1Ratio := 0.5
	deleteRatio := 0.1
	if _, err := WriteBenchFileFromZipfV2(n, s, v, seed, k, phase1Ratio, deleteRatio, file, false); err != nil {
		t.Fatalf("WriteBenchFileFromZipfV2 error: %v", err)
	}

	bf, err := ReadBenchFile(file)
	if err != nil {
		t.Fatalf("ReadBenchFile error: %v", err)
	}

	// 驗證分布 map（Zipf 理論分布，權重集合一致，但 key 由 rank 映射而來）
	if len(bf.Dist) != n {
		t.Fatalf("dist len mismatch: got %d, want %d", len(bf.Dist), n)
	}
	// 計算期望權重（由 rank 0..n-1 推導）
	weights := make([]float64, n)
	var sumW float64
	for i := 0; i < n; i++ {
		w := 1.0 / math.Pow(v+float64(i), s)
		weights[i] = w
		sumW += w
	}
	for i := 0; i < n; i++ {
		weights[i] /= sumW
	}
	// 蒐集實際權重，檢查是否可一一對應到理論權重
	used := make([]bool, n)
	for _, got := range bf.Dist {
		matched := false
		for j := 0; j < n; j++ {
			if used[j] {
				continue
			}
			if floatAlmostEqual(got, weights[j], 1e-12) {
				used[j] = true
				matched = true
				break
			}
		}
		if !matched {
			t.Fatalf("unexpected weight in dist: %v", got)
		}
	}
	for j := 0; j < n; j++ {
		if !used[j] {
			t.Fatalf("expected weight not found: %v", weights[j])
		}
	}

	// 驗證操作序列數量與覆蓋（每個分布中的 key 至少出現一次）
	if len(bf.Ops) != k {
		t.Fatalf("ops len mismatch: got %d, want %d", len(bf.Ops), k)
	}
	distKeys := make(map[int64]struct{}, len(bf.Dist))
	for kx := range bf.Dist {
		distKeys[int64(kx)] = struct{}{}
	}
	seenKeys := make(map[int64]struct{})
	for _, op := range bf.Ops {
		seenKeys[int64(op.Key)] = struct{}{}
	}
	for kx := range distKeys {
		if _, ok := seenKeys[kx]; !ok {
			t.Fatalf("key %d did not appear in ops at least once", kx)
		}
	}
}

// TestReadBenchFileKeysCoverage 測試讀取檔案後，分布中的每個 key 是否至少在操作序列中出現一次
func TestReadBenchFileKeysCoverage(t *testing.T) {
	// 讀取檔案
	bf, err := ReadBenchFile("gravity_test.bin")
	if err != nil {
		t.Fatalf("ReadBenchFile failed: %v", err)
	}

	n := len(bf.Dist)

	// 驗證分布中的每個 key 至少在操作序列中出現一次
	distKeys := make(map[skiplist.K]struct{}, len(bf.Dist))
	for k := range bf.Dist {
		distKeys[k] = struct{}{}
	}

	seenKeys := make(map[skiplist.K]struct{})
	for _, op := range bf.Ops {
		seenKeys[op.Key] = struct{}{}
	}

	for k := range distKeys {
		if _, ok := seenKeys[k]; !ok {
			t.Errorf("key %d from distribution did not appear in operations", k)
		}
	}

	// 驗證至少有 n 個不同的 key 出現
	if len(seenKeys) < n {
		t.Errorf("expected at least %d unique keys in ops, got %d", n, len(seenKeys))
	}
}
