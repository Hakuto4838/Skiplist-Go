# LASK (Look-Ahead Skip List)

LASK 是一種跳躍列表 (Skip List) 的變體，它透過預測未來存取頻率來動態調整節點的高度。這種方法旨在優化跳躍列表的結構，使其對於具有非均勻存取模式的資料集更加高效。

## 功能

-   **基於預測的高度調整**：`PutWithNP` 函式允許在插入元素時提供一個預測的未來存取頻率 (`np`)，LASK 會根據此頻率計算節點的最適高度。
-   **標準跳躍列表操作**：支援標準的 `Put`, `Get`, `Delete`, `Contains` 操作。

## 公開函式

-   `NewLASkipList(seed int64) *LASkipList`
    -   建立一個新的 LASK 實例。`seed` 用於初始化亂數產生器。

-   `Put(key K, value V)`
    -   插入或更新一個鍵值對。此方法使用傳統的隨機方式決定節點高度。

-   `PutWithNP(key K, value V, np float64)`
    -   插入或更新一個鍵值對，並根據預測的未來存取頻率 `np` 來決定節點高度。
    -   `np` 代表預測的未來存取次數。

-   `Get(key K) (V, bool)`
    -   根據 `key` 取得對應的 `value`。如果鍵存在，返回 `value` 和 `true`；否則返回零值和 `false`。

-   `Delete(key K)`
    -   刪除指定的 `key`。

-   `Contains(key K) bool`
    -   檢查指定的 `key` 是否存在於跳躍列表中。

-   `GetMaxStats() (int, int)`
    -   返回跳躍列表中的元素總數和目前的最大層級。

-   `GetHead() skiplist.Nodelike`
    -   返回跳躍列表的頭節點。

## 私有函式

-   `randomLevelWithNP(np float64) int32`
    -   根據預測頻率 `np` 計算節點的高度。

-   `find(key K) (*laNode, bool)`
    -   在跳躍列表中尋找指定的 `key`。

-   `randomLevel() int32`
    -   使用傳統的隨機方式計算節點高度。

## 使用範例

### 基本操作

```go
package main

import (
	"fmt"
	"github.com/Hakuto4838/SkipList.git/skiplist/la"
)

func main() {
	// 使用固定的種子以獲得可預測的結果
	sl := la.NewLASkipList(42)

	// 插入一些值
	sl.Put(1, 100)
	sl.Put(5, 500)
	sl.Put(3, 300)

	// 取得一個值
	if value, found := sl.Get(5); found {
		fmt.Printf("Get(5): value=%d, found=%v\n", value, found)
	}

	// 檢查一個值是否存在
	fmt.Printf("Contains(3): %v\n", sl.Contains(3))
	fmt.Printf("Contains(99): %v\n", sl.Contains(99))

	// 刪除一個值
	sl.Delete(3)
	fmt.Printf("Contains(3) after deletion: %v\n", sl.Contains(3))
}
```

### 使用預測頻率

當您能預測某些鍵的存取頻率時，可以使用 `PutWithNP` 來優化跳躍列表的結構。

```go
package main

import (
	"fmt"
	"github.com/Hakuto4838/SkipList.git/skiplist/la"
)

func main() {
	sl := la.NewLASkipList(42)

	// 假設我們預測 key=10 將被頻繁存取，給予較高的 np
	sl.PutWithNP(10, 1000, 20.0) // np=20.0

	// 其他鍵使用較低的 np
	sl.PutWithNP(1, 100, 2.0)
	sl.PutWithNP(5, 500, 5.0)

	// 這樣，key=10 的節點可能有更高的層級，從而加快存取速度
	_, level := sl.GetMaxStats()
	fmt.Printf("Max level of the skip list: %d\n", level)
}
```
