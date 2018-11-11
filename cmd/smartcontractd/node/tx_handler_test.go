package node

import (
	"reflect"
	"testing"

	"github.com/tokenized/smart-contract/pkg/protocol"
)

func TestTXHandler_handle(t *testing.T) {
	tx := loadFixtureTX("tx_message_chair.txt")

	config := newTestConfig()
	repo := newTestPeerRepo(config)

	store := newMockStorage()
	txService := NewParser(repo)

	keyStore := newTestKeyStore()
	contractState := NewStateService(store)

	service := NewService(config, repo, txService, keyStore, contractState)

	h := NewTXHandler(config, service)

	ctx := newSilentContext()
	err := h.Handle(ctx, &tx)

	if err != nil {
		t.Errorf("want nil, got %v", err)
	}
}

func TestFindTokenizedOpReturn(t *testing.T) {
	t.Skip("Skipping until new fixtures exist")

	tx := loadFixtureTX("tx_message_chair.txt")

	ctx := newSilentContext()

	out, err := findTokenizedProtocol(ctx, &tx)
	if err != nil {
		t.Errorf("want nil, got %v", err)
	}

	want := protocol.NewMessage()
	want.MessageType = []byte{50, 80}

	// malformed message due to message structure change
	want.Message = []byte("arkinson's law says you should buy your girl a big chair.")

	msg, ok := out.(*protocol.Message)
	if !ok {
		t.Fatalf("Could not assert correct type : %+v", msg)
	}

	if !reflect.DeepEqual(*msg, want) {
		t.Errorf("got\n%v\nwant\n%v", *msg, want)
	}
}
