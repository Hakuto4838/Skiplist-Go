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

// UniformDataGenerator 產生符合平均分布的查詢序列
// 每個索引出現機率皆相同
// n: key 數量
// seed: 隨機種子

type UniformDataGenerator struct {
	n   int
	cdf []float64
	rng *rand.Rand
}

func NewUniformDataGenerator(n int, seed int64) *UniformDataGenerator {
	rng := rand.New(rand.NewSource(seed))
	cdf := make([]float64, n)
	for i := 0; i < n; i++ {
		cdf[i] = float64(i+1) / float64(n)
	}
	return &UniformDataGenerator{
		n:   n,
		cdf: cdf,
		rng: rng,
	}
}

// Next 產生一筆查詢 (回傳索引 0~n-1)
func (u *UniformDataGenerator) Next() int {
	r := u.rng.Float64()
	lo, hi := 0, u.n-1
	for lo < hi {
		mid := (lo + hi) / 2
		if r > u.cdf[mid] {
			lo = mid + 1
		} else {
			hi = mid
		}
	}
	return lo
}

// GenerateSequence 產生指定長度的查詢序列
func (u *UniformDataGenerator) GenerateSequence(seqLen int) []int {
	seq := make([]int, seqLen)
	for i := 0; i < seqLen; i++ {
		seq[i] = u.Next()
	}
	return seq
}

// WriteSequenceToFile 產生 k 筆資料並寫入二進位檔案
func (u *UniformDataGenerator) WriteSequenceToFile(filename string, k int) error {
	seq := u.GenerateSequence(k)
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

// GetDistribute 回傳每個 key 的機率分布
func (u *UniformDataGenerator) GetDistribute() map[int]float64 {
	result := make(map[int]float64, u.n)
	for i := 0; i < u.n; i++ {
		result[i] = 1.0 / float64(u.n)
	}
	return result
}

func (u *UniformDataGenerator) DistributeToCSV(writer *csv.Writer) {
	dist := u.GetDistribute()
	keys := make([]string, 0, len(dist)+2)
	probs := make([]string, 0, len(dist)+2)
	keys = append(keys, "", "")
	probs = append(probs, "", "")
	for i := 0; i < u.n; i++ {
		keys = append(keys, fmt.Sprintf("%d", i))
		probs = append(probs, fmt.Sprintf("%f", dist[i]))
	}
	writer.Write(keys)
	writer.Write(probs)
}

func (u *UniformDataGenerator) Close() error {
	return nil
}

func (u *UniformDataGenerator) GetKeyMap() map[skiplist.K]float64 {
	result := make(map[skiplist.K]float64, u.n)
	for i := 0; i < u.n; i++ {
		result[skiplist.K(i)] = 1.0 / float64(u.n)
	}
	return result
}

func (u *UniformDataGenerator) GetCDF() []float64 {
	cdf := make([]float64, u.n)
	for i := 0; i < u.n; i++ {
		cdf[i] = float64(i+1) / float64(u.n)
	}
	return cdf
}

func (u *UniformDataGenerator) GetPDF() []float64 {
	pdf := make([]float64, u.n)
	for i := 0; i < u.n; i++ {
		pdf[i] = 1.0 / float64(u.n)
	}
	return pdf
}

func (u *UniformDataGenerator) Entropy() float64 {
	h := 0.0
	p := 1.0 / float64(u.n)
	if p > 0 {
		h = -float64(u.n) * p * math.Log2(p)
	}
	return h
}
