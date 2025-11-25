package datastream

import "github.com/Hakuto4838/SkipList.git/skiplist"

// DataStream 定義資料流的介面
type DataStream interface {
	Close() error
	Next() int
	GetDistribute() map[int]float64
	GetKeyMap() map[skiplist.K]float64
	GetCDF() []float64
	GetPDF() []float64
	Entropy() float64
}

// OperationType 表示操作種類
type OperationType uint8

const (
	OpQuery OperationType = iota
	OpInsert
	OpDelete
)

func (t OperationType) String() string {
	switch t {
	case OpQuery:
		return "Query"
	case OpInsert:
		return "Insert"
	case OpDelete:
		return "Delete"
	default:
		return "Unknown"
	}
}

// Operation 表示一筆操作
type Operation struct {
	Type OperationType
	Key  int
}

// SequenceModel 以既有的 Operation 序列提供順序重播
type SequenceModel struct {
	ops []Operation
	pos int
}

// NewSequenceModelFromOps 由外部供給的操作序列建立模型
func NewSequenceModelFromOps(ops []Operation) *SequenceModel {
	cp := make([]Operation, len(ops))
	copy(cp, ops)
	return &SequenceModel{ops: cp}
}

// Next 回傳下一筆操作，若結束則回傳零值與 false
func (m *SequenceModel) Next() (Operation, bool) {
	if m.pos >= len(m.ops) {
		return Operation{}, false
	}
	op := m.ops[m.pos]
	m.pos++
	return op, true
}

// NextN 回傳接下來 n 筆（或直到結束）的操作
func (m *SequenceModel) NextN(n int) []Operation {
	if n <= 0 || m.pos >= len(m.ops) {
		return nil
	}
	end := m.pos + n
	if end > len(m.ops) {
		end = len(m.ops)
	}
	out := m.ops[m.pos:end]
	m.pos = end
	// 回傳淺拷貝避免外部修改底層切片
	cp := make([]Operation, len(out))
	copy(cp, out)
	return cp
}

// Reset 游標重置到起點
func (m *SequenceModel) Reset() { m.pos = 0 }
