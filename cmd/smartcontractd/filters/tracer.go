package filters

import (
	"bytes"
	"context"
	"encoding/binary"
	"io"

	"github.com/tokenized/pkg/bitcoin"
	"github.com/tokenized/pkg/wire"
	"github.com/tokenized/smart-contract/internal/platform/db"
	"github.com/tokenized/smart-contract/internal/platform/node"
	"github.com/tokenized/specification/dist/golang/protocol"
)

const (
	tracerStorageKey = "tracer"
)

// Tracer watches UTXO paths starting with a specified outpoint. It can be used to retrace back to
//   a specified transaction if you are expecting a later UTXO spend to come back to you.
type Tracer struct {
	traces []*traceNode
}

func NewTracer() *Tracer {
	result := Tracer{}
	return &result
}

// Count returns the number of active traces.
func (tracer *Tracer) Count() int {
	return len(tracer.traces)
}

// Clear removes all active traces.
func (tracer *Tracer) Clear() {
	tracer.traces = nil
}

func (tracer *Tracer) Save(ctx context.Context, masterDB *db.DB) error {
	// Save the cache list
	var buf bytes.Buffer

	// Write length of list
	count := uint32(len(tracer.traces))
	if err := binary.Write(&buf, protocol.DefaultEndian, &count); err != nil {
		return err
	}

	// Write items
	for _, trace := range tracer.traces {
		if err := trace.write(&buf); err != nil {
			return err
		}
	}

	node.LogVerbose(ctx, "Saving %d traces", len(tracer.traces))
	return masterDB.Put(ctx, tracerStorageKey, buf.Bytes())
}

func (tracer *Tracer) Load(ctx context.Context, masterDB *db.DB) error {
	data, err := masterDB.Fetch(ctx, tracerStorageKey)
	if err != nil {
		if err == db.ErrNotFound {
			return nil
		}
		return err
	}

	buf := bytes.NewBuffer(data)

	// Read length of list
	var count uint32
	if err := binary.Read(buf, protocol.DefaultEndian, &count); err != nil {
		return err
	}

	// Read items
	tracer.traces = make([]*traceNode, count)
	for i := range tracer.traces {
		newTrace := &traceNode{}
		if err := newTrace.read(buf); err != nil {
			return err
		}
		tracer.traces[i] = newTrace
	}

	node.LogVerbose(ctx, "Loaded %d traces", len(tracer.traces))
	return nil
}

// Add adds a new trace starting at the specified output.
func (tracer *Tracer) Add(ctx context.Context, start *wire.OutPoint) {
	newNode := traceNode{
		outpoint: *start,
	}
	tracer.traces = append(tracer.traces, &newNode)
}

// Remove removes a trace containing the specified output.
func (tracer *Tracer) Remove(ctx context.Context, start *wire.OutPoint) {
	for i, trace := range tracer.traces {
		if bytes.Equal(trace.outpoint.Hash[:], start.Hash[:]) &&
			trace.outpoint.Index == start.Index {
			tracer.traces = append(tracer.traces[:i], tracer.traces[i+1:]...)
			return
		}
	}
}

// AddTx adds the next step of any path in tracer if contained in the specified tx.
// Returns true if one of the inputs matches a monitored output.
func (tracer *Tracer) AddTx(ctx context.Context, tx *wire.MsgTx) bool {
	result := false
	for _, trace := range tracer.traces {
		if trace.addTx(tx) {
			result = true
		}
	}
	return result
}

// RevertTx reverts any outputs from any traced paths.
func (tracer *Tracer) RevertTx(ctx context.Context, txid bitcoin.Hash32) {
	for _, trace := range tracer.traces {
		trace.revertTx(txid)
	}
}

// Contains returns the hash of the output that was requested to be monitored that contains the tx
// specified.
func (tracer *Tracer) Contains(ctx context.Context, tx *wire.MsgTx) bool {
	for _, trace := range tracer.traces {
		if trace.contains(tx) {
			return true
		}
	}
	return false
}

// Retrace returns the hash of the output that was requested to be monitored that contains the tx
// specified.
// The trace is also removed.
func (tracer *Tracer) Retrace(ctx context.Context, tx *wire.MsgTx) *bitcoin.Hash32 {
	for i, trace := range tracer.traces {
		if trace.contains(tx) {
			tracer.traces = append(tracer.traces[:i], tracer.traces[i+1:]...)
			return &trace.outpoint.Hash
		}
	}
	return nil
}

type traceNode struct {
	outpoint wire.OutPoint
	children []*traceNode
}

func (node *traceNode) addTx(tx *wire.MsgTx) bool {
	result := false
	if len(node.children) == 0 {
		for _, input := range tx.TxIn {
			if bytes.Equal(input.PreviousOutPoint.Hash[:], node.outpoint.Hash[:]) &&
				input.PreviousOutPoint.Index == node.outpoint.Index {
				for index, _ := range tx.TxOut {
					newNode := traceNode{
						outpoint: wire.OutPoint{
							Hash:  *tx.TxHash(),
							Index: uint32(index),
						},
					}
					node.children = append(node.children, &newNode)
				}
				result = true
			}
		}

		return result
	}

	for _, child := range node.children {
		if child.addTx(tx) {
			result = true
		}
	}
	return result
}

func (node *traceNode) revertTx(txid bitcoin.Hash32) {
	for _, child := range node.children {
		if bytes.Equal(txid[:], child.outpoint.Hash[:]) {
			node.children = nil
			return
		}
		child.revertTx(txid)
	}
}

func (node *traceNode) contains(tx *wire.MsgTx) bool {
	if len(node.children) > 0 {
		for _, input := range tx.TxIn {
			if bytes.Equal(input.PreviousOutPoint.Hash[:], node.outpoint.Hash[:]) && input.PreviousOutPoint.Index == node.outpoint.Index {
				return true
			}
		}
	}

	for _, child := range node.children {
		if child.contains(tx) {
			return true
		}
	}
	return false
}

func (node *traceNode) write(w io.Writer) error {
	// Write outpoint
	if _, err := w.Write(node.outpoint.Hash[:]); err != nil {
		return err
	}
	if err := binary.Write(w, protocol.DefaultEndian, &node.outpoint.Index); err != nil {
		return err
	}

	// Write length of list
	count := uint32(len(node.children))
	if err := binary.Write(w, protocol.DefaultEndian, &count); err != nil {
		return err
	}

	// Write items
	for _, child := range node.children {
		if err := child.write(w); err != nil {
			return err
		}
	}

	return nil
}

func (node *traceNode) read(r io.Reader) error {
	// Read outpoint
	if _, err := io.ReadFull(r, node.outpoint.Hash[:]); err != nil {
		return err
	}
	if err := binary.Read(r, protocol.DefaultEndian, &node.outpoint.Index); err != nil {
		return err
	}

	// Read length of list
	var count uint32
	if err := binary.Read(r, protocol.DefaultEndian, &count); err != nil {
		return err
	}

	// Read items
	node.children = make([]*traceNode, count)
	for i := range node.children {
		child := &traceNode{}
		if err := child.read(r); err != nil {
			return err
		}
		node.children[i] = child
	}

	return nil
}
