package datastream

import (
	"encoding/binary"
	"encoding/csv"
	"fmt"
	"math"
	"math/rand"
	"os"

	"github.com/Hakuto4838/SkipList.git/skiplist"
)

// ZipfDataGenerator 產生符合 Zipf 分布的查詢序列
type ZipfDataGenerator struct {
	n       int
	a, b    float64
	Weights []float64
	cdf     []float64
	rng     *rand.Rand
}

func NewZipfDataGenerator(n int, a, b float64, seed int64) *ZipfDataGenerator {
	rng := rand.New(rand.NewSource(seed))
	weights := make([]float64, n)
	var sum float64
	for i := 1; i <= n; i++ {
		weights[i-1] = 1.0 / math.Pow(float64(i)+b, a)
		sum += weights[i-1]
	}
	// 正規化
	for i := range weights {
		weights[i] /= sum
	}
	rng.Shuffle(len(weights), func(i, j int) {
		weights[i], weights[j] = weights[j], weights[i]
	})
	// 建立累積分布函數 (CDF)
	cdf := make([]float64, n)
	cdf[0] = weights[0]
	for i := 1; i < n; i++ {
		cdf[i] = cdf[i-1] + weights[i]
	}
	return &ZipfDataGenerator{
		n:       n,
		a:       a,
		b:       b,
		Weights: weights,
		cdf:     cdf,
		rng:     rng,
	}
}

// Next 產生一筆查詢 (回傳索引 0~n-1)
func (z *ZipfDataGenerator) Next() int {
	r := z.rng.Float64()
	// 二分搜尋 cdf
	lo, hi := 0, z.n-1
	for lo < hi {
		mid := (lo + hi) / 2
		if r > z.cdf[mid] {
			lo = mid + 1
		} else {
			hi = mid
		}
	}
	return lo
}

// GenerateSequence 產生指定長度的查詢序列
func (z *ZipfDataGenerator) GenerateSequence(seqLen int) []int {
	seq := make([]int, seqLen)
	for i := 0; i < seqLen; i++ {
		seq[i] = z.Next()
	}
	return seq
}

// WriteSequenceToFile 產生 k 筆資料並寫入二進位檔案
func (z *ZipfDataGenerator) WriteSequenceToFile(filename string, k int) error {
	seq := z.GenerateSequence(k)
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()
	for _, v := range seq {
		if err := binary.Write(file, binary.LittleEndian, int32(v)); err != nil {
			return err
		}
	}
	return nil
}

// ReadSequenceFromFile 從二進位檔案讀取資料
type SequenceReader struct {
	seq []int
	pos int
}

func NewSequenceReaderFromFile(filename string) (*SequenceReader, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	var seq []int
	var v int32
	for {
		err := binary.Read(file, binary.LittleEndian, &v)
		if err != nil {
			break
		}
		seq = append(seq, int(v))
	}
	return &SequenceReader{seq: seq, pos: 0}, nil
}

// Next 取得下一個值，若無資料則回傳 false
func (sr *SequenceReader) Next() (int, bool) {
	if sr.pos >= len(sr.seq) {
		return 0, false
	}
	val := sr.seq[sr.pos]
	sr.pos++
	return val, true
}

// return keys, prob
func (z *ZipfDataGenerator) GetDistribute() map[int]float64 {
	result := make(map[int]float64, z.n)
	for i := 0; i < z.n; i++ {
		result[i] = z.Weights[i]
	}
	return result
}

func (z *ZipfDataGenerator) DistributeToCSV(writer *csv.Writer) {
	dist := z.GetDistribute()
	keys := make([]string, 0, len(dist)+2)
	probs := make([]string, 0, len(dist)+2)
	keys = append(keys, "", "")
	probs = append(probs, "", "")

	// Go map is not ordered, but we can assume keys are 0 to n-1
	for i := 0; i < z.n; i++ {
		keys = append(keys, fmt.Sprintf("%d", i))
		probs = append(probs, fmt.Sprintf("%f", dist[i]))
	}

	writer.Write(keys)
	writer.Write(probs)
}

func (z *ZipfDataGenerator) Close() error {
	return nil
}

func (z *ZipfDataGenerator) GetKeyMap() map[skiplist.K]float64 {
	result := make(map[skiplist.K]float64, z.n)
	for i := 0; i < z.n; i++ {
		result[skiplist.K(i)] = z.Weights[i]
	}
	return result
}

// getCDF 計算累積分布函數，並回傳一個新的 slice，避免汙染原本的 Weights
func (z *ZipfDataGenerator) GetCDF() []float64 {
	cdf := make([]float64, len(z.Weights))
	sum := 0.0
	for i, w := range z.Weights {
		sum += w
		cdf[i] = sum
	}
	return cdf
}

func (z *ZipfDataGenerator) GetPDF() []float64 {
	pdf := make([]float64, len(z.Weights))
	for i := range z.Weights {
		pdf[i] = z.Weights[i]
	}
	return pdf
}

func (z *ZipfDataGenerator) Entropy() float64 {
	h := 0.0
	for _, p := range z.Weights {
		if p > 0 {
			h -= p * math.Log2(p)
		}
	}
	return h
}
