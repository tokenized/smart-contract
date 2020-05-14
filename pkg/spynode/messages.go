package spynode

import (
	"sync"

	"github.com/tokenized/smart-contract/pkg/wire"

	"github.com/pkg/errors"
)

type MessageChannel struct {
	Channel chan wire.Message
	lock    sync.Mutex
	open    bool
}

func (c *MessageChannel) Add(msg wire.Message) error {
	c.lock.Lock()
	defer c.lock.Unlock()

	if !c.open {
		return errors.New("Channel closed")
	}

	c.Channel <- msg
	return nil
}

func (c *MessageChannel) Open(count int) error {
	c.lock.Lock()
	defer c.lock.Unlock()

	c.Channel = make(chan wire.Message, count)
	c.open = true
	return nil
}

func (c *MessageChannel) Close() error {
	c.lock.Lock()
	defer c.lock.Unlock()

	if !c.open {
		return errors.New("Channel closed")
	}

	close(c.Channel)
	c.open = false
	return nil
}
