package request

import "context"

type requestHandlerInterface interface {
	handle(context.Context, contractRequest) (*contractResponse, error)
}
