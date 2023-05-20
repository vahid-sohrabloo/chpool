package column

// Uint16 use for UInt16 ClickHouse DataType
type Uint16 struct {
	column
	dict map[uint16]int
	keys []int
}

// NewUint16 return new Uint16 for UInt16 ClickHouse DataType
func NewUint16(nullable bool) *Uint16 {
	return &Uint16{
		dict: make(map[uint16]int),
		column: column{
			nullable:    nullable,
			colNullable: newNullable(),
			size:        Uint16Size,
		},
	}
}

// Next forward pointer to the next value. Returns false if there are no more values.
//
// Use with Value() or ValueP()
func (c *Uint16) Next() bool {
	if c.i >= c.totalByte {
		return false
	}
	c.i += Uint16Size
	return true
}

// Append value for insert
func (c *Uint16) Append(v uint16) {
	c.numRow++
	c.writerData = append(c.writerData,
		byte(v),
		byte(v>>8),
	)
}

// AppendP value for insert (for nullable column)
//
// As an alternative (for better performance), you can use `Append` to append data. and `AppendIsNil` to say this value is null or not
//
// NOTE: for alternative mode. of your value is nil you still need to append default value. You can use `AppendEmpty()` for nil values
func (c *Uint16) AppendP(v *uint16) {
	if v == nil {
		c.AppendEmpty()
		c.colNullable.Append(1)
		return
	}
	c.colNullable.Append(0)
	c.Append(*v)
}

// AppendDict add value to the dictionary (if doesn't exist on dictionary) and append key of the dictionary to keys
//
// Only use for LowCardinality data type
func (c *Uint16) AppendDict(v uint16) {
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
func (c *Uint16) AppendDictNil() {
	c.keys = append(c.keys, 0)
}

// AppendDictP add value to the dictionary (if doesn't exist on dictionary)
// and append key of the dictionary to keys (for nullable data type)
//
// As an alternative (for better performance), You can use `AppendDict()` and `AppendDictNil` instead of this function.
//
// For alternative way You shouldn't append empty value for nullable data
func (c *Uint16) AppendDictP(v *uint16) {
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
func (c *Uint16) getKeys() []int {
	return c.keys
}

// Reset all status and buffer data
//
// Reading data does not require a reset after each read. The reset will be triggered automatically.
//
// However, writing data requires a reset after each write.
func (c *Uint16) Reset() {
	c.column.Reset()
	c.keys = c.keys[:0]
	c.dict = make(map[uint16]int)
}