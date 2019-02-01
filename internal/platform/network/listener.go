package network

import (
	"context"

	"github.com/tokenized/smart-contract/pkg/wire"
)

type Listener interface {
	Handle(context.Context, wire.Message) error
}
