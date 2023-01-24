package chconn

import (
	"bytes"
	"fmt"

	"github.com/vahid-sohrabloo/chconn/v3/column"
	"github.com/vahid-sohrabloo/chconn/v3/internal/helper"
	"github.com/vahid-sohrabloo/chconn/v3/internal/readerwriter"
)

// Column contains details of ClickHouse column
type chColumn struct {
	ChType []byte
	Name   []byte
}

type block struct {
	Columns    []chColumn
	NumRows    uint64
	NumColumns uint64
	info       blockInfo
	// only use for compress data
	compressWriter *readerwriter.Writer
}

func newBlock() *block {
	return &block{
		compressWriter: readerwriter.NewWriter(),
	}
}

func (block *block) reset() {
	block.compressWriter.Reset()
	block.Columns = block.Columns[:0]
	block.NumRows = 0
	block.NumColumns = 0
}

func (block *block) read(ch *conn) error {
	if _, err := ch.reader.ByteString(); err != nil { // temporary table
		return &readError{"block: temporary table", err}
	}
	ch.reader.SetCompress(ch.compress)
	defer ch.reader.SetCompress(false)

	var err error
	err = block.info.read(ch.reader)
	if err != nil {
		return err
	}

	block.NumColumns, err = ch.reader.Uvarint()
	if err != nil {
		return &readError{"block: read NumColumns", err}
	}

	block.NumRows, err = ch.reader.Uvarint()
	if err != nil {
		return &readError{"block: read NumRows", err}
	}
	return nil
}

func (block *block) readColumns(ch *conn) error {
	ch.reader.SetCompress(ch.compress)
	defer ch.reader.SetCompress(false)
	block.Columns = make([]chColumn, block.NumColumns)

	for i := uint64(0); i < block.NumColumns; i++ {
		col, err := block.nextColumn(ch)
		if err != nil {
			return err
		}
		block.Columns[i] = col
	}
	return nil
}

func (block *block) readColumnsData(ch *conn, needValidateData bool, columns ...column.ColumnBasic) error {
	ch.reader.SetCompress(ch.compress)
	defer ch.reader.SetCompress(false)
	for _, col := range columns {
		err := col.HeaderReader(ch.reader, true, ch.serverInfo.Revision)
		if err != nil {
			return fmt.Errorf("read column header: %w", err)
		}
		if needValidateData {
			if errValidate := col.Validate(); errValidate != nil {
				return fmt.Errorf("validate %q: %w", col.Name(), errValidate)
			}
		}
		err = col.ReadRaw(int(block.NumRows), ch.reader)
		if err != nil {
			return fmt.Errorf("read data %q: %w", col.Name(), err)
		}
	}
	return nil
}

func (block *block) reorderColumns(columns []column.ColumnBasic) ([]column.ColumnBasic, error) {
	for i, c := range block.Columns {
		// check if already sorted
		if bytes.Equal(columns[i].Name(), block.Columns[i].Name) {
			continue
		}
		index, col := findColumn(columns, c.Name)
		if col == nil {
			return nil, &ColumnNotFoundError{
				Column: string(c.Name),
			}
		}
		columns[index] = columns[i]
		columns[i] = col
	}
	return columns, nil
}

func findColumn(columns []column.ColumnBasic, name []byte) (int, column.ColumnBasic) {
	for i, col := range columns {
		if bytes.Equal(col.Name(), name) {
			return i, col
		}
	}
	return 0, nil
}

func (block *block) nextColumn(ch *conn) (chColumn, error) {
	col := chColumn{}
	var err error
	if col.Name, err = ch.reader.ByteString(); err != nil {
		return col, &readError{"block: read column name", err}
	}
	if col.ChType, err = ch.reader.ByteString(); err != nil {
		return col, &readError{"block: read column type", err}
	}
	if ch.serverInfo.Revision >= helper.DbmsMinProtocolWithCustomSerialization {
		customSerialization, err := ch.reader.ReadByte()
		if err != nil {
			return col, &readError{"block: read custom serialization", err}
		}
		if customSerialization == 1 {
			return col, &readError{"block: custom serialization not supported", nil}
		}
	}
	return col, nil
}

func (block *block) write(ch *conn, numRows int, columns ...column.ColumnBasic) error {
	writeTo := ch.writer
	if ch.compress {
		block.compressWriter.Reset()
		writeTo = block.compressWriter
	}
	block.info.write(writeTo)
	// NumColumns
	writeTo.Uvarint(block.NumColumns)
	// NumRows
	writeTo.Uvarint(uint64(numRows))

	if len(columns) > 0 {
		numRows := columns[0].NumRow()
		for i, column := range block.Columns {
			if numRows != columns[i].NumRow() {
				return &NumberWriteError{
					FirstNumRow: numRows,
					NumRow:      columns[i].NumRow(),
					Column:      string(column.Name),
					FirstColumn: string(block.Columns[0].Name),
				}
			}

			writeTo.ByteString(column.Name)
			writeTo.ByteString(column.ChType)

			if ch.serverInfo.Revision >= helper.DbmsMinProtocolWithCustomSerialization {
				writeTo.Uint8(0)
			}

			columns[i].HeaderWriter(writeTo)
			columns[i].Write(writeTo)
			if !ch.config.UseWriteBuffer && !ch.compress {
				if err := ch.flushWriteData(); err != nil {
					return &writeError{"block: write block data for column " + string(column.Name), err}
				}
			}
		}
	}
	if ch.compress {
		if err := ch.compressWriter.Compress(writeTo.Output); err != nil {
			return &writeError{"block: compress block", err}
		}
		ch.writer.Output = append(ch.writer.Output, ch.compressWriter.Data...)
	}

	return nil
}

type blockInfo struct {
	field1      uint64
	isOverflows uint8
	field2      uint64
	bucketNum   int32
	num3        uint64
}

func (info *blockInfo) read(r *readerwriter.Reader) error {
	var err error
	if info.field1, err = r.Uvarint(); err != nil {
		return &readError{"blockInfo: read field1", err}
	}
	if info.isOverflows, err = r.ReadByte(); err != nil {
		return &readError{"blockInfo: read isOverflows", err}
	}
	if info.field2, err = r.Uvarint(); err != nil {
		return &readError{"blockInfo: read field2", err}
	}
	if info.bucketNum, err = r.Int32(); err != nil {
		return &readError{"blockInfo: read bucketNum", err}
	}
	if info.num3, err = r.Uvarint(); err != nil {
		return &readError{"blockInfo: read num3", err}
	}
	return nil
}

func (info *blockInfo) write(w *readerwriter.Writer) {
	w.Uvarint(1)
	w.Uint8(info.isOverflows)
	w.Uvarint(2)

	if info.bucketNum == 0 {
		info.bucketNum = -1
	}
	w.Int32(info.bucketNum)
	w.Uvarint(0)
}
