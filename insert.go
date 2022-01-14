package chconn

import (
	"context"

	"github.com/vahid-sohrabloo/chconn/column"
	"github.com/vahid-sohrabloo/chconn/setting"
)

func commit(c *conn, b *block, columns ...column.Column) error {
	if len(columns) == 0 {
		return ErrInsertMinColumn
	}
	err := c.sendData(b, columns[0].NumRow())
	if err != nil {
		return &InsertError{
			err: err,
		}
	}

	err = b.writeColumnsBuffer(c, columns...)
	if err != nil {
		return err
	}

	err = c.sendEmptyBlock()

	if err != nil {
		return err
	}

	res, err := c.receiveAndProccessData(emptyOnProgress)
	if err != nil {
		return err
	}

	if res != nil {
		return &unexpectedPacket{expected: "serverEndOfStream", actual: res}
	}

	return nil
}

// Insert send query for insert and commit columns
func (ch *conn) Insert(ctx context.Context, query string, columns ...column.Column) error {
	return ch.InsertWithSetting(ctx, query, nil, "", columns...)
}

// Insert send query for insert and prepare insert stmt with setting option
func (ch *conn) InsertWithSetting(
	ctx context.Context,
	query string,
	settings *setting.Settings,
	queryID string,
	columns ...column.Column) error {
	err := ch.lock()
	if err != nil {
		return err
	}
	defer ch.unlock()
	ch.contextWatcher.Watch(ctx)
	defer ch.contextWatcher.Unwatch()
	var hasError bool
	defer func() {
		if hasError {
			ch.Close(context.Background())
		}
	}()
	err = ch.sendQueryWithOption(ctx, query, queryID, settings)
	if err != nil {
		hasError = true
		return err
	}

	var blockData *block
	for {
		var res interface{}
		res, err = ch.receiveAndProccessData(emptyOnProgress)
		if err != nil {
			hasError = true
			return err
		}
		if b, ok := res.(*block); ok {
			blockData = b
			break
		}

		if _, ok := res.(*Profile); ok {
			continue
		}
		if _, ok := res.(*Progress); ok {
			continue
		}
		hasError = true
		return &unexpectedPacket{expected: "serverData", actual: res}
	}

	err = blockData.initForInsert(ch)
	if err != nil {
		hasError = true
		return err
	}

	return commit(ch, blockData, columns...)
}