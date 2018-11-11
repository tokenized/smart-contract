package spvnode

import (
	"context"

	"github.com/tokenized/smart-contract/pkg/wire"
)

type Queuer interface {
	Queue(context.Context, wire.Message) error
}
