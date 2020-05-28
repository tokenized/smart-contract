package node

import (
	"context"

	"github.com/tokenized/pkg/json"
	"github.com/tokenized/pkg/logger"
)

// Convert assigns all available compatible values with matching member names from one object to
//   another.
// The dst object needs to be a pointer so that it can be written to.
// Members of these objects that are "specialized", like a struct containing only a string, need
//   to have json.Marshaler and json.UnMarshaler interfaces implemented.
func Convert(ctx context.Context, src interface{}, dst interface{}) error {
	// Marshal source object to json.
	var data []byte
	var err error
	data, err = json.Marshal(src)
	if err != nil {
		LogDepth(ctx, logger.LevelWarn, 1, "Failed json marshal : %s", err)
		return err
	}

	// Unmarshal json back into destination object.
	err = json.Unmarshal(data, dst)
	if err != nil {
		LogDepth(ctx, logger.LevelWarn, 1, "Failed json unmarshal : %s\n%s", err, string(data))
		return err
	}

	return nil
}
