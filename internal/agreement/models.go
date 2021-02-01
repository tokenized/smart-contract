package agreement

import (
	"github.com/tokenized/specification/dist/golang/actions"
	"github.com/tokenized/specification/dist/golang/protocol"
)

// NewAgreement defines what we require when creating an agreement record.
type NewAgreement struct {
	Chapters    []*actions.ChapterField     `json:"Chapters,omitempty"`
	Definitions []*actions.DefinedTermField `json:"Definitions,omitempty"`
	CreatedAt   protocol.Timestamp          `json:"CreatedAt,omitempty"`
	Timestamp   protocol.Timestamp          `json:"Timestamp,omitempty"`
}

// UpdateAgreement defines what information may be provided to modify an existing
// agreement. All fields are optional so clients can send just the fields they want
// changed. It uses pointer fields so we can differentiate between a field that
// was not provided and a field that was provided as explicitly blank. Normally
// we do not want to use pointers to basic types but we make exceptions around
// marshalling/unmarshalling.
type UpdateAgreement struct {
	Chapters    *[]*actions.ChapterField     `json:"Chapters,omitempty"`
	Definitions *[]*actions.DefinedTermField `json:"Definitions,omitempty"`
	Revision    *uint32                      `json:"Revision,omitempty"`
	Timestamp   *protocol.Timestamp          `json:"Timestamp,omitempty"`
}
