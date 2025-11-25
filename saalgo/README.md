# 模擬退火算法框架

這是一個通用的模擬退火算法框架，可以用於解決各種優化問題。

## 功能特點

- **通用性**: 通過接口設計，可以應用於任何優化問題
- **可配置**: 提供豐富的參數配置選項
- **易用性**: 簡潔的API設計，易於使用和擴展
- **高效性**: 實現了標準的模擬退火算法

## 核心組件

### Solution 接口

要使用此框架，您的解需要實現 `Solution` 接口：

```go
type Solution interface {
    Clone() Solution           // 創建解的深拷貝
    GetCost() float64         // 返回解的成本/適應度
    GenerateNeighbor() Solution // 生成鄰居解
}
```

### SAConfig 配置

```go
type SAConfig struct {
    InitialTemp    float64 // 初始溫度
    FinalTemp      float64 // 最終溫度
    CoolingRate    float64 // 冷卻率
    Iterations     int     // 每個溫度的迭代次數
    MaxIterations  int     // 最大總迭代次數
    RandomSeed     int64   // 隨機種子
}
```

## 使用方法

### 1. 實現 Solution 接口

```go
type MySolution struct {
    // 您的解數據
    data []int
    cost float64
}

func (s *MySolution) Clone() saalgo.Solution {
    dataCopy := make([]int, len(s.data))
    copy(dataCopy, s.data)
    return &MySolution{
        data: dataCopy,
        cost: s.cost,
    }
}

func (s *MySolution) GetCost() float64 {
    return s.cost
}

func (s *MySolution) GenerateNeighbor() saalgo.Solution {
    // 實現鄰居生成邏輯
    // 例如：隨機交換兩個元素
    neighbor := s.Clone().(*MySolution)
    // ... 鄰居生成邏輯
    return neighbor
}
```

### 2. 創建和運行算法

```go
// 使用默認配置
sa := saalgo.NewSimulatedAnnealing(nil)

// 或使用自定義配置
config := &saalgo.SAConfig{
    InitialTemp:   1000.0,
    FinalTemp:     0.1,
    CoolingRate:   0.95,
    Iterations:    100,
    MaxIterations: 10000,
}
sa := saalgo.NewSimulatedAnnealing(config)

// 創建初始解
initialSolution := &MySolution{...}

// 運行算法
bestSolution, bestCost := sa.Run(initialSolution)
```

## 參數調優指南

### 溫度參數

- **InitialTemp**: 初始溫度應該足夠高，使得在開始時有較大概率接受較差的解
- **FinalTemp**: 最終溫度應該足夠低，使得在結束時幾乎只接受更好的解
- **CoolingRate**: 冷卻率控制溫度下降速度，通常設置為 0.8-0.99

### 迭代參數

- **Iterations**: 每個溫度的迭代次數，影響搜索深度
- **MaxIterations**: 最大總迭代次數，防止無限運行

## 示例

### 數值優化問題

```go
// 最小化函數 f(x) = x^2 + 2*x + 1
type NumberSolution struct {
    value float64
    cost  float64
}

func (ns *NumberSolution) GenerateNeighbor() saalgo.Solution {
    neighborValue := ns.value + (rand.Float64()-0.5)*2.0
    return NewNumberSolution(neighborValue)
}
```

### 旅行商問題 (TSP)

```go
// 使用2-opt交換生成鄰居
func (tsp *TSPSolution) GenerateNeighbor() saalgo.Solution {
    path := make([]int, len(tsp.path))
    copy(path, tsp.path)
    
    // 隨機選擇兩個位置進行交換
    i := rand.Intn(len(path))
    j := rand.Intn(len(path))
    path[i], path[j] = path[j], path[i]
    
    return NewTSPSolution(path, calculateTSPCost(path))
}
```

## 運行演示

```bash
go run cmd/sa_demo/main.go
```

## 注意事項

1. **解的表示**: 確保您的解表示方法適合問題特性
2. **鄰居生成**: 鄰居生成策略對算法性能影響很大
3. **成本計算**: 成本函數應該準確反映解的好壞
4. **參數調優**: 不同問題可能需要不同的參數設置

## 擴展

您可以擴展此框架以支持：

- 自適應溫度調度
- 多目標優化
- 並行搜索
- 自定義接受準則 