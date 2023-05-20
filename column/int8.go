package column

// Int8 use for Int8 ClickHouse DataType
type Int8 struct {
	column
	val  int8
	dict map[int8]int
	keys []int
}

// NewInt8 return new Int8 for Int8 ClickHouse DataType
func NewInt8(nullable bool) *Int8 {
	return &Int8{
		dict: make(map[int8]int),
		column: column{
			nullable:    nullable,
			colNullable: newNullable(),
			size:        Int8Size,
		},
	}
}

// Next forward pointer to the next value. Returns false if there are no more values.
//
// Use with Value() or ValueP()
func (c *Int8) Next() bool {
	if c.i >= c.totalByte {
		return false
	}
	c.val = int8(c.b[c.i])
	c.i += c.size
	return true
}

// Value of current pointer
//
// Use with Next()
func (c *Int8) Value() int8 {
	return c.val
}

// Row return the value of given row
// NOTE: Row number start from zero
func (c *Int8) Row(row int) int8 {
	return int8(c.b[row])
}

// RowP return the value of given row for nullable data
// NOTE: Row number start from zero
//
// As an alternative (for better performance), you can use `Row()` to get a value and `ValueIsNil()` to check if it is null.
//
func (c *Int8) RowP(row int) *int8 {
	if c.colNullable.b[row] == 1 {
		return nil
	}
	val := int8(c.b[row])
	return &val
}

// ReadAll read all value in this block and append to the input slice
func (c *Int8) ReadAll(value *[]int8) {
	for _, v := range c.b {
		*value = append(*value, int8(v))
	}
}

// Fill slice with value and forward the pointer by the length of the slice
//
// NOTE: A slice that is longer than the remaining data is not safe to pass.
func (c *Int8) Fill(value []int8) {
	for i := range value {
		value[i] = int8(c.b[c.i])
		c.i += c.size
	}
}

// ValueP Value of current pointer for nullable data
//
// As an alternative (for better performance), you can use `Value()` to get a value and `ValueIsNil()` to check if it is null.
//
// Use with Next()
func (c *Int8) ValueP() *int8 {
	if c.colNullable.b[c.i-c.size] == 1 {
		return nil
	}
	val := c.val
	return &val
}

// ReadAllP read all value in this block and append to the input slice (for nullable data)
//
// As an alternative (for better performance), you can use `ReadAll()` to get a values and `ReadAllNil()` to check if they are null.
func (c *Int8) ReadAllP(value *[]*int8) {
	for i := 0; i < c.totalByte; i += c.size {
		if c.colNullable.b[i] != 0 {
			*value = append(*value, nil)
			continue
		}
		val := int8(c.b[i])
		*value = append(*value, &val)
	}
}

// FillP slice with value and forward the pointer by the length of the slice (for nullable data)
//
// As an alternative (for better performance), you can use `Fill()` to get a values and `FillNil()` to check if they are null.
//
// NOTE: A slice that is longer than the remaining data is not safe to pass.
func (c *Int8) FillP(value []*int8) {
	for i := range value {
		if c.colNullable.b[c.i] == 1 {
			c.i += c.size
			value[i] = nil
			continue
		}
		val := int8(c.b[c.i])
		value[i] = &val
		c.i += c.size
	}
}

// Append value for insert
func (c *Int8) Append(v int8) {
	c.numRow++
	c.writerData = append(c.writerData,
		byte(v),
	)
}

// AppendEmpty append empty value for insert
func (c *Int8) AppendEmpty() {
	c.numRow++
	c.writerData = append(c.writerData,
		0,
	)
}

// AppendP value for insert (for nullable column)
//
// As an alternative (for better performance), you can use `Append` to append data. and `AppendIsNil` to say this value is null or not
//
// NOTE: for alternative mode. of your value is nil you still need to append default value. You can use `AppendEmpty()` for nil values
func (c *Int8) AppendP(v *int8) {
	if v == nil {
		c.AppendEmpty()
		c.colNullable.Append(1)
		return
	}
	c.colNullable.Append(0)
	c.Append(*v)
}

// AppendDict add value to dictionary and append keys
//
// Only use for LowCardinality data type
func (c *Int8) AppendDict(v int8) {
	key, ok := c.dict[v]
	if !ok {
		key = len(c.dict)
		c.dict[v] = key
		c.Append(v)
	}
	if c.nullable {
		c.keys = append(c.keys, key+1)
	} else {
		c.keys = append(c.keys, key)
	}
}

// AppendDictNil add nil key for LowCardinality nullable data type
func (c *Int8) AppendDictNil() {
	c.keys = append(c.keys, 0)
}

// AppendDictP add value to dictionary and append keys (for nullable data type)
//
// As an alternative (for better performance), You can use `AppendDict()` and `AppendDictNil` instead of this function.
//
// For alternative way You shouldn't append empty value for nullable data
func (c *Int8) AppendDictP(v *int8) {
	if v == nil {
		c.keys = append(c.keys, 0)
		return
	}
	key, ok := c.dict[*v]
	if !ok {
		key = len(c.dict)
		c.dict[*v] = key
		c.Append(*v)
	}
	c.keys = append(c.keys, key+1)
}

// Keys current keys for LowCardinality data type
func (c *Int8) getKeys() []int {
	return c.keys
}

// Reset all status and buffer data
//
// Reading data does not require a reset after each read. The reset will be triggered automatically.
//
// However, writing data requires a reset after each write.
func (c *Int8) Reset() {
	c.column.Reset()
	c.keys = c.keys[:0]
	c.dict = make(map[int8]int)
}