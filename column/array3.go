package column

import (
	"fmt"
	"reflect"
)

// Array3 is a column of Array(Array(Array(T))) ClickHouse data type
type Array3[T any] struct {
	ArrayBase
}

// NewArray create a new array column of Array(Array(Array(T))) ClickHouse data type
func NewArray3[T any](array *Array2[T]) *Array3[T] {
	a := &Array3[T]{
		ArrayBase: ArrayBase{
			dataColumn:      array,
			offsetColumn:    New[uint64](),
			arrayChconnType: "column.Array3[" + reflect.TypeOf((*T)(nil)).Elem().String() + "]",
		},
	}
	return a
}

// Data get all the data in current block as a slice.
func (c *Array3[T]) Data() [][][][]T {
	values := make([][][][]T, c.offsetColumn.numRow)
	for i := range values {
		values[i] = c.Row(i)
	}
	return values
}

// Read reads all the data in current block and append to the input.
func (c *Array3[T]) Read(value [][][][]T) [][][][]T {
	if cap(value)-len(value) >= c.NumRow() {
		value = (value)[:len(value)+c.NumRow()]
	} else {
		value = append(value, make([][][][]T, c.NumRow())...)
	}
	val := (value)[len(value)-c.NumRow():]
	for i := 0; i < c.NumRow(); i++ {
		val[i] = c.Row(i)
	}
	return value
}

// Row return the value of given row.
// NOTE: Row number start from zero
func (c *Array3[T]) Row(row int) [][][]T {
	var lastOffset uint64
	if row != 0 {
		lastOffset = c.offsetColumn.Row(row - 1)
	}
	var val [][][]T
	lastRow := c.offsetColumn.Row(row)
	for ; lastOffset < lastRow; lastOffset++ {
		val = append(val, c.dataColumn.(*Array2[T]).Row(int(lastOffset)))
	}
	return val
}

// RowAny return the value of given row.
// NOTE: Row number start from zero
func (c *Array3[T]) RowAny(row int) any {
	return c.Row(row)
}

func (c *Array3[T]) Scan(row int, dest any) error {
	switch v := dest.(type) {
	case *[][][]T:
		*v = c.Row(row)
		return nil
	case *any:
		*v = c.Row(row)
		return nil
	}
	return c.ScanValue(row, reflect.ValueOf(dest))
}

func (c *Array3[T]) ScanValue(row int, dest reflect.Value) error {
	destValue := reflect.Indirect(dest)
	if destValue.Kind() != reflect.Slice {
		return fmt.Errorf("dest must be a pointer to slice")
	}

	if destValue.Type().AssignableTo(reflect.TypeOf([][][]T{})) {
		destValue.Set(reflect.ValueOf(c.Row(row)))
		return nil
	}

	var lastOffset int
	if row != 0 {
		lastOffset = int(c.offsetColumn.Row(row - 1))
	}
	offset := int(c.offsetColumn.Row(row))
	rSlice := reflect.MakeSlice(destValue.Type(), offset-lastOffset, offset-lastOffset)
	for i, b := lastOffset, 0; i < offset; i, b = i+1, b+1 {
		err := c.dataColumn.Scan(i, rSlice.Index(b).Addr().Interface())
		if err != nil {
			return fmt.Errorf("cannot scan array item %d: %w", i, err)
		}
	}
	destValue.Set(rSlice)
	return nil
}

// Append value for insert
func (c *Array3[T]) Append(v [][][]T) {
	c.AppendLen(len(v))
	c.dataColumn.(*Array2[T]).AppendMulti(v...)
}

// AppendMulti value for insert
func (c *Array3[T]) AppendMulti(v ...[][][]T) {
	for _, v := range v {
		c.AppendLen(len(v))
		c.dataColumn.(*Array2[T]).AppendMulti(v...)
	}
}

// Array return a Array type for this column
func (c *Array2[T]) Array() *Array3[T] {
	return NewArray3(c)
}

func (c *Array3[T]) elem(arrayLevel int) ColumnBasic {
	if arrayLevel > 0 {
		panic("array level is too deep")
	}
	return c
}

func (c *Array3[T]) ToJSON(row int, ignoreDoubleQuotes bool, b []byte) []byte {
	b = append(b, '[')

	var lastOffset uint64
	if row != 0 {
		lastOffset = c.offsetColumn.Row(row - 1)
	}
	offset := c.offsetColumn.Row(row)
	first := true
	for i := lastOffset; i < offset; i++ {
		b = c.dataColumn.ToJSON(int(i), ignoreDoubleQuotes, b)

		if !first {
			b = append(b, ',')
		} else {
			first = false
		}
	}
	return append(b, ']')
}
