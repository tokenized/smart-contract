package platform

import "encoding/json"

// Convert assigns all available compatible values with matching member names from one object to
//   another.
// The dst object needs to be a pointer so that it can be written to.
// Members of these objects that are "specialized", like a struct containing only a string, need
//   to have json.Marshaler and json.UnMarshaler interfaces implemented.
func Convert(src interface{}, dst interface{}) error {
	// Marshal source object to json.
	var data []byte
	var err error
	data, err = json.Marshal(src)
	if err != nil {
		return err
	}

	// Unmarshal json back into destination object.
	err = json.Unmarshal(data, dst)
	if err != nil {
		return err
	}

	return nil
}
