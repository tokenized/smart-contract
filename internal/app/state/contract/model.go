package contract

type Model interface {
	GetId() string
	SetId(id string)
}

type modelImpl struct {
	id string
}

func (m *modelImpl) SetId(id string) {
	m.id = id
}
