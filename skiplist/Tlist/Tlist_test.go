package tlist

import (
	"testing"

	"github.com/Hakuto4838/SkipList.git/skiplist"
)

func TestTListBasic(t *testing.T) {
	// 創建一個 TList，span 設為 2
	tl := NewSkipList(2)

	// 測試插入 5 個元素
	for i := 1; i <= 5; i++ {
		tl.Put(skiplist.K(i), skiplist.V(i*10))
	}

	// 測試 Get 操作
	for i := 1; i <= 5; i++ {
		value, found := tl.Get(skiplist.K(i))
		if !found {
			t.Errorf("期望找到 key %d，但沒有找到", i)
		}
		if value != skiplist.V(i*10) {
			t.Errorf("key %d 的值期望為 %d，實際為 %f", i, i*10, value)
		}
	}

	// 測試 Contains 操作
	for i := 1; i <= 5; i++ {
		if !tl.Contains(skiplist.K(i)) {
			t.Errorf("期望 key %d 存在，但 Contains 返回 false", i)
		}
	}

	// 測試不存在的 key
	if tl.Contains(skiplist.K(6)) {
		t.Error("期望 key 6 不存在，但 Contains 返回 true")
	}

	// 測試 Delete 操作
	tl.Delete(skiplist.K(3))
	if tl.Contains(skiplist.K(3)) {
		t.Error("刪除 key 3 後，期望不存在，但 Contains 返回 true")
	}

	// 測試統計信息
	size, level := tl.GetMaxStats()
	t.Logf("TList 統計: size=%d, level=%d", size, level)
}
