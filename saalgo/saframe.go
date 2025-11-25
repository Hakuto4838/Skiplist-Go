package saalgo

import (
	"math"
	"math/rand"
	"time"
)

// Solution 表示一個解，需要實現以下接口
type Solution interface {
	// Clone 創建當前解的深拷貝
	Clone() Solution

	// GetCost 返回當前解的成本/適應度
	GetCost() float64

	// GenerateNeighbor 生成鄰居解
	GenerateNeighbor() Solution
}

// ProgressCallback 進度回報回調函數類型
// 參數: iteration (當前迭代次數), maxIterations (最大迭代次數), temperature (當前溫度), bestCost (目前最佳成本), currentCost (當前成本)
type ProgressCallback func(iteration int, maxIterations int, temperature float64, bestCost float64, currentCost float64)

// SAConfig 模擬退火配置
type SAConfig struct {
	InitialTemp      float64          // 初始溫度
	FinalTemp        float64          // 最終溫度
	CoolingRate      float64          // 冷卻率
	Iterations       int              // 每個溫度的迭代次數
	MaxIterations    int              // 最大總迭代次數
	RandomSeed       int64            // 隨機種子
	ProgressCallback ProgressCallback // 進度回報回調函數（可選）
	ProgressInterval int              // 進度回報間隔（每 N 次迭代回報一次，0 表示不回報）
}

// DefaultConfig 返回默認配置
func DefaultConfig() *SAConfig {
	return &SAConfig{
		InitialTemp:   1000.0,
		FinalTemp:     0.1,
		CoolingRate:   0.95,
		Iterations:    100,
		MaxIterations: 10000,
		RandomSeed:    time.Now().UnixNano(),
	}
}

// SimulatedAnnealing 模擬退火算法主結構
type SimulatedAnnealing struct {
	config     *SAConfig
	bestSol    Solution
	bestCost   float64
	iterations int
}

// NewSimulatedAnnealing 創建新的模擬退火實例
func NewSimulatedAnnealing(config *SAConfig) *SimulatedAnnealing {
	if config == nil {
		config = DefaultConfig()
	}

	rand.Seed(config.RandomSeed)

	return &SimulatedAnnealing{
		config:     config,
		iterations: 0,
	}
}

// Run 執行模擬退火算法
func (sa *SimulatedAnnealing) Run(initialSolution Solution) (Solution, float64) {
	currentSol := initialSolution.Clone()
	currentCost := currentSol.GetCost()

	sa.bestSol = currentSol.Clone()
	sa.bestCost = currentCost

	temperature := sa.config.InitialTemp

	for temperature > sa.config.FinalTemp && sa.iterations < sa.config.MaxIterations {
		for i := 0; i < sa.config.Iterations; i++ {
			// 生成鄰居解
			neighborSol := currentSol.GenerateNeighbor()
			neighborCost := neighborSol.GetCost()

			// 計算成本差
			deltaCost := neighborCost - currentCost

			// 決定是否接受新解
			if sa.shouldAccept(deltaCost, temperature) {
				currentSol = neighborSol
				currentCost = neighborCost

				// 更新最佳解
				if currentCost < sa.bestCost {
					sa.bestSol = currentSol.Clone()
					sa.bestCost = currentCost
				}
			}

			sa.iterations++

			// 進度回報
			if sa.config.ProgressCallback != nil && sa.config.ProgressInterval > 0 {
				if sa.iterations%sa.config.ProgressInterval == 0 {
					sa.config.ProgressCallback(sa.iterations, sa.config.MaxIterations, temperature, sa.bestCost, currentCost)
				}
			}

			if sa.iterations >= sa.config.MaxIterations {
				break
			}
		}

		// 冷卻
		temperature *= sa.config.CoolingRate
	}

	return sa.bestSol, sa.bestCost
}

// shouldAccept 決定是否接受新解
func (sa *SimulatedAnnealing) shouldAccept(deltaCost, temperature float64) bool {
	// 如果新解更好，直接接受
	if deltaCost < 0 {
		return true
	}

	// 否則根據Metropolis準則決定
	probability := math.Exp(-deltaCost / temperature)
	return rand.Float64() < probability
}

// GetBestSolution 返回最佳解
func (sa *SimulatedAnnealing) GetBestSolution() Solution {
	return sa.bestSol
}

// GetBestCost 返回最佳成本
func (sa *SimulatedAnnealing) GetBestCost() float64 {
	return sa.bestCost
}

// GetIterations 返回迭代次數
func (sa *SimulatedAnnealing) GetIterations() int {
	return sa.iterations
}

// Reset 重置算法狀態
func (sa *SimulatedAnnealing) Reset() {
	sa.bestSol = nil
	sa.bestCost = 0
	sa.iterations = 0
}
