package ops

import (
	"context"

	"time"

	"github.com/10gen/mongo-go-driver/bson"
	"github.com/10gen/mongo-go-driver/yamgo/internal"
	"github.com/10gen/mongo-go-driver/yamgo/options"
)

// Find executes a query.
//
// TODO GODRIVER-76: Document which types for interface{} are valid.
func Find(ctx context.Context, s *SelectedServer, ns Namespace, filter interface{},
	findOptions ...options.FindOption) (Cursor, error) {

	if err := ns.validate(); err != nil {
		return nil, err
	}

	command := bson.D{
		{Name: "find", Value: ns.Collection},
	}

	if filter != nil {
		command.AppendElem("filter", filter)
	}

	limit := int64(0)
	batchSize := int32(0)

	for _, option := range findOptions {
		switch name := option.FindName(); name {
		// upsert, multi, and collation are specified in each update documents
		case "cursorType":
			value := option.FindValue()
			if value == options.Tailable {
				command.AppendElem("tailable", true)
			}

			if value == options.TailableAwait {
				command.AppendElem("tailable", true)
				command.AppendElem("awaitData", true)
			}

		case "maxTimeMS":
			command.AppendElem(
				name,
				int64(option.FindValue().(time.Duration)/time.Millisecond),
			)

		// other options are specified in the top-level command document
		default:
			if name == "limit" {
				limit = int64(option.FindValue().(options.OptLimit))
			}

			if name == "batchSize" {
				batchSize = int32(option.FindValue().(options.OptBatchSize))
			}

			command.AppendElem(name, option.FindValue())
		}
	}

	if limit != 0 && batchSize != 0 && limit <= int64(batchSize) {
		command.AppendElem("singleBatch", true)
	}

	// TODO GODRIVER-27: read concern

	var result cursorReturningResult

	err := runMayUseSecondary(ctx, s, ns.DB, command, &result)
	if err != nil {
		return nil, internal.WrapError(err, "failed to execute update")
	}

	return NewCursor(&result.Cursor, batchSize, s)
}