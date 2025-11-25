package basic

import (
	"testing"

	"github.com/Hakuto4838/SkipList.git/datastream"
	"github.com/Hakuto4838/SkipList.git/skiplist"
	"github.com/Hakuto4838/SkipList.git/skiplist/analyTool"
)

func TestBasicSkipListInterface(t *testing.T) {
	var _ skiplist.SkipList = (*BasicSkipList)(nil)
	var _ skiplist.Analyable = (*BasicSkipList)(nil)
	var _ skiplist.Nodelike = (*basicNode)(nil)
}

func TestBasicSkipListBasic(t *testing.T) {
	sl := NewBasicSkipList(42)
	sl.Put(1, 100)
	sl.Put(2, 200)
	sl.Put(3, 300)

	analyTool.PrintSkipList(sl, 5, 10)
}

func TestBigZipf(t *testing.T) {
	data := datastream.NewZipfDataGenerator(100000, 1.5, 1, 42)
	sl := NewBasicSkipList(42)
	for k, v := range data.GetKeyMap() {
		sl.Put(k, v)
	}
	for range 100000 {
		sl.Get(skiplist.K(data.Next()))
	}
	analyTool.PrintSkipList(sl, 5, 10)
}
