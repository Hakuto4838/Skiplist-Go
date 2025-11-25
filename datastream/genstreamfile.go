package datastream

import (
	"encoding/binary"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"sort"

	randv2 "math/rand/v2"

	"github.com/Hakuto4838/SkipList.git/skiplist"
)

// 檔案格式（LittleEndian）：
// [8]byte  Magic: "SLBENCH1"
// uint16   Version: 1
// uint16   Reserved: 0
// uint32   DistCount
// 重複 DistCount 次：
//   int64   Key
//   float64 Weight
// uint64   OpCount
// 重複 OpCount 次：
//   uint8   OperationType (0=Query,1=Insert,2=Delete)
//   int64   Key

var (
	benchMagic   = [8]byte{'S', 'L', 'B', 'E', 'N', 'C', 'H', '1'}
	benchVersion = uint16(1)
)

type BenchOp struct {
	Type OperationType
	Key  skiplist.K
}

type BenchFile struct {
	Dist map[skiplist.K]float64
	Ops  []BenchOp
}

type ZipfV2Info struct {
	Dist    map[int64]float64
	Entropy float64
}

// WriteBenchFileFromZipf 以 ZipfDataGenerator 與操作數 k 產生對應 bin 檔。
// 規則：
//   - 若 Zipf.Next() 給的 key 未曾出現過，則輸出 Insert
//   - 若已出現過，則 90% Query、5% Delete（僅當目前存在時）、其餘 Insert
//   - 搜尋與刪除僅會在該 key 至少插入過一次之後才可能出現
func WriteBenchFileFromZipf(gen *ZipfDataGenerator, k int, filename string) error {
	if gen == nil {
		return errors.New("nil ZipfDataGenerator")
	}
	if k < 0 {
		return fmt.Errorf("invalid k: %d", k)
	}

	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	// Header
	if _, err := file.Write(benchMagic[:]); err != nil {
		return err
	}
	if err := binary.Write(file, binary.LittleEndian, benchVersion); err != nil {
		return err
	}
	if err := binary.Write(file, binary.LittleEndian, uint16(0)); err != nil { // reserved
		return err
	}

	// Distribution map（使用升冪 key 輸出，確保可重現）
	dist := gen.GetKeyMap()
	keys := make([]int, 0, len(dist))
	for k := range dist {
		keys = append(keys, int(k))
	}
	sort.Ints(keys)

	if err := binary.Write(file, binary.LittleEndian, uint32(len(keys))); err != nil {
		return err
	}
	for _, ik := range keys {
		k64 := int64(ik)
		w := dist[skiplist.K(ik)]
		if err := binary.Write(file, binary.LittleEndian, k64); err != nil {
			return err
		}
		if err := binary.Write(file, binary.LittleEndian, w); err != nil {
			return err
		}
	}

	// Operations
	if err := binary.Write(file, binary.LittleEndian, uint64(k)); err != nil {
		return err
	}

	// 追蹤狀態：是否曾出現過
	everSeen := make(map[int]bool, len(keys))

	for i := 0; i < k; i++ {
		idx := gen.Next() // 0..n-1
		var op OperationType

		if !everSeen[idx] {
			op = OpInsert
			everSeen[idx] = true
		} else {
			// 已出現：90% Query、10% Insert
			r := gen.rng.Float64()
			if r < 0.90 {
				op = OpQuery
			} else {
				op = OpInsert
			}
		}

		if err := binary.Write(file, binary.LittleEndian, uint8(op)); err != nil {
			return err
		}
		if err := binary.Write(file, binary.LittleEndian, int64(idx)); err != nil {
			return err
		}
	}

	return nil
}

// writeBenchFileFromUniform 使用均勻分布產生操作序列並寫入檔案。
// 參數與邏輯與 WriteBenchFileFromZipfV2 相同，但使用均勻分布而非 Zipf 分布。
func writeBenchFileFromUniform(n int, seed uint64, k int, phase1Ratio, deleteRatio float64, filename string, simpleKey bool) (*ZipfV2Info, error) {
	phase1Size := int(float64(k) * phase1Ratio)
	info := &ZipfV2Info{
		Dist:    make(map[int64]float64, n),
		Entropy: 0.0,
	}

	if k < n {
		return nil, fmt.Errorf("k (%d) must be >= n (%d) to ensure each key appears at least once", k, n)
	}
	if phase1Size < n || phase1Size > k {
		return nil, fmt.Errorf("phase1Size (%d) must satisfy n <= phase1Size <= k", phase1Size)
	}
	if deleteRatio < 0.0 || deleteRatio > 1.0 {
		return nil, fmt.Errorf("deleteRatio (%v) must be between 0.0 and 1.0", deleteRatio)
	}

	r := randv2.New(randv2.NewPCG(seed, 0))

	file, err := os.Create(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	// Header
	if _, err := file.Write(benchMagic[:]); err != nil {
		return nil, err
	}
	if err := binary.Write(file, binary.LittleEndian, benchVersion); err != nil {
		return nil, err
	}
	if err := binary.Write(file, binary.LittleEndian, uint16(0)); err != nil { // reserved
		return nil, err
	}

	// Distribution map（均勻分布）
	// 1) 建立 rank -> key 的隨機對應（不重複）
	rankToKey := make([]int64, n)
	if simpleKey {
		for i := 0; i < n; i++ {
			rankToKey[i] = int64(i)
		}
		r.Shuffle(len(rankToKey), func(i, j int) { rankToKey[i], rankToKey[j] = rankToKey[j], rankToKey[i] })
	} else {
		check := make(map[int64]struct{})
		for i := 0; i < n; i++ {
			genKey := int64(r.Uint32())
			for _, ok := check[genKey]; ok; _, ok = check[genKey] {
				genKey = int64(r.Uint32())
			}
			rankToKey[i] = genKey
			check[genKey] = struct{}{}
		}
	}

	// 2) 均勻分布：每個 key 的機率相同
	weight := 1.0 / float64(n)

	// 3) 將 (key, weight) 對應後，依 key 升冪輸出
	if err := binary.Write(file, binary.LittleEndian, uint32(n)); err != nil {
		return nil, err
	}
	type kv struct {
		k int64
		w float64
	}
	pairs := make([]kv, n)
	for rank := 0; rank < n; rank++ {
		pairs[rank] = kv{k: rankToKey[rank], w: weight}
	}
	sort.Slice(pairs, func(i, j int) bool { return pairs[i].k < pairs[j].k })
	for _, p := range pairs {
		if err := binary.Write(file, binary.LittleEndian, int64(p.k)); err != nil {
			return nil, err
		}
		if err := binary.Write(file, binary.LittleEndian, p.w); err != nil {
			return nil, err
		}
	}

	// 組出回傳的分布 map（key->weight）
	distOut := make(map[int64]float64, n)
	for _, p := range pairs {
		distOut[p.k] = p.w
	}

	// Operations count
	if err := binary.Write(file, binary.LittleEndian, uint64(k)); err != nil {
		return nil, err
	}

	// 產生第一階段的 key 列表（長度 phase1Size）：
	// 前 n 個覆蓋所有 key，後面 phase1Size-n 個用均勻隨機補齊，最後打亂
	phase1Keys := make([]int64, phase1Size)
	for i := 0; i < n; i++ {
		phase1Keys[i] = rankToKey[i]
	}
	for i := n; i < phase1Size; i++ {
		rank := r.IntN(n) // 均勻隨機選擇 0..n-1
		phase1Keys[i] = rankToKey[rank]
	}
	r.Shuffle(len(phase1Keys), func(i, j int) { phase1Keys[i], phase1Keys[j] = phase1Keys[j], phase1Keys[i] })

	// 狀態：是否在表中
	present := make(map[int64]bool, n)

	// 逐一賦予操作（第一階段）
	for _, key := range phase1Keys {
		var op OperationType
		if !present[key] {
			op = OpInsert
			present[key] = true
		} else {
			if r.Float64() < deleteRatio {
				op = OpDelete
				present[key] = false
			} else {
				op = OpQuery
			}
		}
		if err := binary.Write(file, binary.LittleEndian, uint8(op)); err != nil {
			return nil, err
		}
		if err := binary.Write(file, binary.LittleEndian, int64(key)); err != nil {
			return nil, err
		}
	}

	// 第二階段：剩餘 k - phase1Size 個操作，均勻隨機選擇 rank 再映射 key
	for i := phase1Size; i < k; i++ {
		rank := r.IntN(n)
		key := rankToKey[rank]
		var op OperationType
		if !present[key] {
			op = OpInsert
			present[key] = true
		} else {
			if r.Float64() < deleteRatio {
				op = OpDelete
				present[key] = false
			} else {
				op = OpQuery
			}
		}
		if err := binary.Write(file, binary.LittleEndian, uint8(op)); err != nil {
			return nil, err
		}
		if err := binary.Write(file, binary.LittleEndian, int64(key)); err != nil {
			return nil, err
		}
	}

	info.Entropy = EntropyFromDist(distOut)
	info.Dist = distOut

	return info, nil
}

// WriteBenchFileFromZipfV2 使用 math/rand/v2 的 Zipf 分布產生操作序列並寫入檔案。
// 參數：
//   - n: key 數量（keys 為 0..n-1）
//   - s, v: Zipf 參數。當 s = 0 時使用均勻分布；否則需滿足 s > 1、v >= 1
//   - seed: 隨機種子
//   - k: 輸出操作數量（需 >= n，以保證每個 key 至少出現一次）
//
// 規則：
//   - 先保證每個 key 至少一次 Insert（順序會隨機洗牌）
//   - 之後的操作：若 key 已出現過，90% Query、10% Insert
//   - 檔頭與分布輸出格式同 WriteBenchFileFromZipf
func WriteBenchFileFromZipfV2(n int, s, v float64, seed uint64, k int, phase1Ratio, deleteRatio float64, filename string, simpleKey bool) (*ZipfV2Info, error) {
	phase1Size := int(float64(k) * phase1Ratio)
	info := &ZipfV2Info{
		Dist:    make(map[int64]float64, n),
		Entropy: 0.0,
	}
	if n <= 0 {
		return nil, fmt.Errorf("invalid n: %d", n)
	}
	// 特殊情況：s = 0 表示使用均勻分布
	if s == 0.0 {
		return writeBenchFileFromUniform(n, seed, k, phase1Ratio, deleteRatio, filename, simpleKey)
	}
	if s <= 1.0 || v < 1.0 {
		return nil, fmt.Errorf("invalid zipf params: s=%v must >1, v=%v must >=1", s, v)
	}
	if k < n {
		return nil, fmt.Errorf("k (%d) must be >= n (%d) to ensure each key appears at least once", k, n)
	}
	if phase1Size < n || phase1Size > k {
		return nil, fmt.Errorf("phase1Size (%d) must satisfy n <= phase1Size <= k", phase1Size)
	}
	if deleteRatio < 0.0 || deleteRatio > 1.0 {
		return nil, fmt.Errorf("deleteRatio (%v) must be between 0.0 and 1.0", deleteRatio)
	}
	r := randv2.New(randv2.NewPCG(seed, 0))
	zipf := randv2.NewZipf(r, s, v, uint64(n-1))

	file, err := os.Create(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	// Header
	if _, err := file.Write(benchMagic[:]); err != nil {
		return nil, err
	}
	if err := binary.Write(file, binary.LittleEndian, benchVersion); err != nil {
		return nil, err
	}
	if err := binary.Write(file, binary.LittleEndian, uint16(0)); err != nil { // reserved
		return nil, err
	}

	// Distribution map（以升冪 key 輸出）
	// 1) 建立 rank -> key 的隨機對應（不重複）
	rankToKey := make([]int64, n)
	if simpleKey {
		for i := 0; i < n; i++ {
			rankToKey[i] = int64(i)
		}
		r.Shuffle(len(rankToKey), func(i, j int) { rankToKey[i], rankToKey[j] = rankToKey[j], rankToKey[i] })
	} else {
		check := make(map[int64]struct{})
		for i := 0; i < n; i++ {
			genKey := int64(r.Uint32())
			for _, ok := check[genKey]; ok; _, ok = check[genKey] {
				genKey = int64(r.Uint32())
			}
			rankToKey[i] = genKey
			check[genKey] = struct{}{}
		}
	}

	// 2) 計算 Zipf 理論機率（針對 rank），並正規化
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

	// 3) 將 (key, weight) 對應後，依 key 升冪輸出
	if err := binary.Write(file, binary.LittleEndian, uint32(n)); err != nil {
		return nil, err
	}
	type kv struct {
		k int64
		w float64
	}
	pairs := make([]kv, n)
	for rank := 0; rank < n; rank++ {
		pairs[rank] = kv{k: rankToKey[rank], w: weights[rank]}
	}
	sort.Slice(pairs, func(i, j int) bool { return pairs[i].k < pairs[j].k })
	for _, p := range pairs {
		if err := binary.Write(file, binary.LittleEndian, int64(p.k)); err != nil {
			return nil, err
		}
		if err := binary.Write(file, binary.LittleEndian, p.w); err != nil {
			return nil, err
		}
	}

	// 組出回傳的分布 map（key->weight）
	distOut := make(map[int64]float64, n)
	for _, p := range pairs {
		distOut[p.k] = p.w
	}

	// Operations count
	if err := binary.Write(file, binary.LittleEndian, uint64(k)); err != nil {
		return nil, err
	}

	// 產生第一階段的 key 列表（長度 phase1Size）：
	// 前 n 個覆蓋所有 key，後面 phase1Size-n 個用 zipf(rank) 補齊，最後打亂
	phase1Keys := make([]int64, phase1Size)
	for i := 0; i < n; i++ {
		phase1Keys[i] = rankToKey[i]
	}
	for i := n; i < phase1Size; i++ {
		rank := int(zipf.Uint64())
		phase1Keys[i] = rankToKey[rank]
	}
	r.Shuffle(len(phase1Keys), func(i, j int) { phase1Keys[i], phase1Keys[j] = phase1Keys[j], phase1Keys[i] })

	// 狀態：是否在表中
	present := make(map[int64]bool, n)

	// 逐一賦予操作（第一階段）
	for _, key := range phase1Keys {
		var op OperationType
		if !present[key] {
			op = OpInsert
			present[key] = true
		} else {
			if r.Float64() < deleteRatio {
				op = OpDelete
				present[key] = false
			} else {
				op = OpQuery
			}
		}
		if err := binary.Write(file, binary.LittleEndian, uint8(op)); err != nil {
			return nil, err
		}
		if err := binary.Write(file, binary.LittleEndian, int64(key)); err != nil {
			return nil, err
		}
	}

	// 第二階段：剩餘 k - phase1Size 個操作，zipf 取 rank 再映射 key，規則同上
	for i := phase1Size; i < k; i++ {
		rank := int(zipf.Uint64())
		key := rankToKey[rank]
		var op OperationType
		if !present[key] {
			op = OpInsert
			present[key] = true
		} else {
			if r.Float64() < deleteRatio {
				op = OpDelete
				present[key] = false
			} else {
				op = OpQuery
			}
		}
		if err := binary.Write(file, binary.LittleEndian, uint8(op)); err != nil {
			return nil, err
		}
		if err := binary.Write(file, binary.LittleEndian, int64(key)); err != nil {
			return nil, err
		}
	}

	info.Entropy = EntropyFromDist(distOut)
	info.Dist = distOut

	return info, nil
}

// ReadBenchFile 讀取 bin 檔案，回傳分布與操作序列。
func ReadBenchFile(filename string) (*BenchFile, error) {
	fd, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer fd.Close()

	var magic [8]byte
	if _, err := io.ReadFull(fd, magic[:]); err != nil {
		return nil, err
	}
	if magic != benchMagic {
		return nil, fmt.Errorf("invalid magic: %q", magic)
	}
	var ver uint16
	if err := binary.Read(fd, binary.LittleEndian, &ver); err != nil {
		return nil, err
	}
	if ver != benchVersion {
		return nil, fmt.Errorf("unsupported version: %d", ver)
	}
	// reserved
	var reserved uint16
	if err := binary.Read(fd, binary.LittleEndian, &reserved); err != nil {
		return nil, err
	}

	// distribution
	var distCount uint32
	if err := binary.Read(fd, binary.LittleEndian, &distCount); err != nil {
		return nil, err
	}
	dist := make(map[skiplist.K]float64, distCount)
	for i := uint32(0); i < distCount; i++ {
		var key int64
		var weight float64
		if err := binary.Read(fd, binary.LittleEndian, &key); err != nil {
			return nil, err
		}
		if err := binary.Read(fd, binary.LittleEndian, &weight); err != nil {
			return nil, err
		}
		dist[skiplist.K(key)] = weight
	}

	// operations
	var opCount uint64
	if err := binary.Read(fd, binary.LittleEndian, &opCount); err != nil {
		return nil, err
	}
	ops := make([]BenchOp, 0, opCount)
	for i := uint64(0); i < opCount; i++ {
		var t uint8
		var key int64
		if err := binary.Read(fd, binary.LittleEndian, &t); err != nil {
			return nil, err
		}
		if err := binary.Read(fd, binary.LittleEndian, &key); err != nil {
			return nil, err
		}
		ops = append(ops, BenchOp{Type: OperationType(t), Key: skiplist.K(key)})
	}

	return &BenchFile{Dist: dist, Ops: ops}, nil
}

// ToSequenceModel 將 BenchFile 轉為可重播的 SequenceModel（以 int key）。
func (bf *BenchFile) ToSequenceModel() *SequenceModel {
	if bf == nil {
		return NewSequenceModelFromOps(nil)
	}
	ops := make([]Operation, len(bf.Ops))
	for i, op := range bf.Ops {
		ops[i] = Operation{Type: op.Type, Key: int(op.Key)}
	}
	return NewSequenceModelFromOps(ops)
}

// EntropyFromDist 計算分布的熵（單位：bit）。
// dist 的 value 應為已正規化的機率；會自動忽略 <= 0 的值。
func EntropyFromDist(dist map[int64]float64) float64 {
	h := 0.0
	for _, p := range dist {
		if p > 0 {
			h -= p * math.Log2(p)
		}
	}
	return h
}

func (info *ZipfV2Info) DistributeToCSV(writer *csv.Writer) {
	dist := info.Dist
	keys := make([]string, 0, len(dist)+2)
	probs := make([]string, 0, len(dist)+2)
	keys = append(keys, "", "")
	probs = append(probs, "", "")
	// 先收集所有 key 並排序
	sortedKeys := make([]int64, 0, len(dist))
	for k := range dist {
		sortedKeys = append(sortedKeys, k)
	}
	sort.Slice(sortedKeys, func(i, j int) bool {
		return sortedKeys[i] < sortedKeys[j]
	})

	// 按排序後的順序輸出
	for _, k := range sortedKeys {
		v := dist[k]
		keys = append(keys, fmt.Sprintf("%d", k))
		probs = append(probs, fmt.Sprintf("%f", v))
	}
	writer.Write(keys)
	writer.Write(probs)
}
