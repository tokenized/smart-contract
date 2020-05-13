package wire

import (
	"bytes"
	"fmt"
	"io"
	"unicode/utf8"

	"github.com/tokenized/smart-contract/pkg/bitcoin"

	"github.com/pkg/errors"
)

// MsgParseBlock is a more efficient version of MsgBlock. It initially just determines the complete
//   data set for the block and calculates the merkle root hash. Then allows each tx to be parsed
//   one at a time and builds them using the same memory as the block data where possible.
// The ReadMessageParse function must be used to receive a MsgParseBlock. The sole difference
//   between ReadMessageParse and ReadMessage is that it returns a MsgParseBlock for CmdBlock
//   commands.
type MsgParseBlock struct {
	Header     BlockHeader
	MerkleRoot bitcoin.Hash32 // Calculated during initial parse
	TxCount    uint64         // Total number of txs

	pver     uint32
	txOffset uint64 // Offset into data of the next tx
	data     []byte
}

// IsMerkleRootValid returns true if the decoded block's calculated merkle root hash matches the
//   header.
func (mpb *MsgParseBlock) IsMerkleRootValid() bool {
	return mpb.Header.MerkleRoot.Equal(&mpb.MerkleRoot)
}

// GetNextTx returns the next tx from the block.
func (mpb *MsgParseBlock) GetNextTx() (*MsgTx, error) {
	if uint64(len(mpb.data)) == mpb.txOffset {
		return nil, nil // No txs left to parse
	}

	size, tx, err := ReadTxBytes(mpb.data[mpb.txOffset:], mpb.pver)
	if err != nil {
		return nil, errors.Wrap(err, "read tx")
	}
	mpb.txOffset += size
	return tx, nil
}

// ResetTxs resets GetNextTx to the first tx in the block.
func (mpb *MsgParseBlock) ResetTxs() {
	mpb.txOffset = 0
}

// *************************************************************************************************
// Message interface
func (mpb *MsgParseBlock) BtcDecode(r io.Reader, pver uint32) error {
	mpb.pver = pver // Save for tx decoding later

	err := readBlockHeader(r, pver, &mpb.Header)
	if err != nil {
		return err
	}

	mpb.TxCount, err = ReadVarInt(r, pver)
	if err != nil {
		return err
	}

	// Prevent more transactions than could possibly fit into a block.
	// It would be possible to cause memory exhaustion and panics without
	// a sane upper bound on this count.
	if mpb.TxCount > maxTxPerBlock {
		str := fmt.Sprintf("too many transactions to fit into a block "+
			"[count %d, max %d]", mpb.TxCount, maxTxPerBlock)
		return messageError("MsgBlock.BtcDecode", str)
	}

	// Write all data read from this point on to the raw data for the block transactions so it can
	//   be reparsed to process each tx.
	var dataBuf bytes.Buffer
	r = io.TeeReader(r, &dataBuf)

	// Read the raw data for each tx and calculate it's hash so the merkle root hash can be
	//   calculated.
	txids := make([]*bitcoin.Hash32, 0, mpb.TxCount)
	for i := uint64(0); i < mpb.TxCount; i++ {
		txid, err := readTxId(r, pver) // Reads full tx data and returns the hash
		if err != nil {
			return err
		}
		txids = append(txids, &txid)
	}

	mpb.data = dataBuf.Bytes()

	// Calculate merkle root hash
	if mpb.TxCount == 1 {
		mpb.MerkleRoot = *txids[0] // Special case : Use hash of only tx
	} else {
		mrh := calculateMerkleLevel(txids)
		mpb.MerkleRoot = *mrh
	}
	return nil
}

func (mpb *MsgParseBlock) BtcEncode(w io.Writer, pver uint32) error {
	err := writeBlockHeader(w, pver, &mpb.Header)
	if err != nil {
		return err
	}

	err = WriteVarInt(w, pver, uint64(mpb.TxCount))
	if err != nil {
		return err
	}

	_, err = w.Write(mpb.data)
	if err != nil {
		return err
	}

	return nil
}

func (mpb *MsgParseBlock) Command() string {
	return CmdBlock
}

func (mpb *MsgParseBlock) MaxPayloadLength(pver uint32) uint32 {
	return MaxBlockPayload
}

// *************************************************************************************************

func ReadMessageParse(r io.Reader, pver uint32, btcnet BitcoinNet) (int, Message, []byte, error) {
	totalBytes := 0
	n, hdr, err := readMessageHeader(r)
	totalBytes += n
	if err != nil {
		if err == io.EOF {
			return totalBytes, nil, nil, messageTypeError("ReadMessage",
				MessageErrorConnectionClosed, err.Error())
		}
		return totalBytes, nil, nil, messageTypeError("ReadMessage", MessageErrorUndefined,
			err.Error())
	}

	// Check for messages from the wrong bitcoin network.
	if hdr.magic != btcnet {
		discardInput(r, hdr.length)
		str := fmt.Sprintf("[%v]", hdr.magic)
		return totalBytes, nil, nil, messageTypeError("ReadMessage", MessageErrorWrongNetwork, str)
	}

	// Enforce maximum message payload.
	if hdr.length > MaxMessagePayload {
		str := fmt.Sprintf("message payload is too large - header "+
			"indicates %d bytes, but max message payload is %d "+
			"bytes.", hdr.length, MaxMessagePayload)
		return totalBytes, nil, nil, messageError("ReadMessage", str)

	}

	// Check for malformed commands.
	command := hdr.command
	if !utf8.ValidString(command) {
		discardInput(r, hdr.length)
		return totalBytes, nil, nil, messageTypeError("ReadMessage", MessageErrorUnknownCommand,
			command)
	}

	var msg Message
	if command == CmdBlock { // Build MsgParseBlock instead of MsgBlock
		msg = &MsgParseBlock{}
	} else {
		// Create struct of appropriate message type based on the command.
		msg, err = makeEmptyMessage(command)
		if err != nil {
			discardInput(r, hdr.length)
			return totalBytes, nil, nil, messageTypeError("ReadMessage", MessageErrorUnknownCommand,
				command)
		}
	}

	// Check for maximum length based on the message type as a malicious client
	// could otherwise create a well-formed header and set the length to max
	// numbers in order to exhaust the machine's memory.
	mpl := msg.MaxPayloadLength(pver)
	if hdr.length > mpl {
		discardInput(r, hdr.length)
		str := fmt.Sprintf("payload exceeds max length - header "+
			"indicates %v bytes, but max payload size for "+
			"messages of type [%v] is %v.", hdr.length, command, mpl)
		return totalBytes, nil, nil, messageError("ReadMessage", str)
	}

	// Read payload.
	payload := make([]byte, hdr.length)
	n, err = io.ReadFull(r, payload)
	totalBytes += n
	if err != nil {
		return totalBytes, nil, nil, err
	}

	// Test checksum.
	checksum := bitcoin.DoubleSha256(payload)[0:4]
	if !bytes.Equal(checksum, hdr.checksum[:]) {
		str := fmt.Sprintf("payload checksum failed - header "+
			"indicates %v, but actual checksum is %v.", hdr.checksum, checksum)
		return totalBytes, nil, nil, messageError("ReadMessage", str)
	}

	// Unmarshal message.  NOTE: This must be a *bytes.Buffer since the
	// MsgVersion BtcDecode function requires it.
	if err := msg.BtcDecode(bytes.NewBuffer(payload), pver); err != nil {
		return totalBytes, nil, nil, err
	}

	return totalBytes, msg, payload, nil
}

// readTxId reads the data for a full tx and returns the double SHA256 hash of it. It must take an
//   io.Reader so that it can read from a net.Conn and wait for the data to be received. This does
//   make it less efficient though because there is no skip function, so memory must be allocated to
//   read past values that we don't actually care about.
func readTxId(r io.Reader, pver uint32) (bitcoin.Hash32, error) {
	// Tee reader into a buffer to calculate hash after reading.
	var buf bytes.Buffer // To calculate hash
	r = io.TeeReader(r, &buf)

	if _, err := readTxSize(r, pver); err != nil {
		return bitcoin.Hash32{}, errors.Wrap(err, "read tx size")
	}

	result, _ := bitcoin.NewHash32(bitcoin.DoubleSha256(buf.Bytes()))
	return *result, nil
}

// readTxSize reads the data for a full tx and returns the size that it read. It must take an
//   io.Reader so that it can read from a net.Conn and wait for the data to be received. This does
//   make it less efficient though because there is no skip function, so memory must be allocated to
//   read past values that we don't actually care about.
func readTxSize(r io.Reader, pver uint32) (uint64, error) {
	size := uint64(8) // fixed size of version and lock time
	var fourbytes [4]byte

	// Version
	if _, err := r.Read(fourbytes[:]); err != nil {
		return 0, errors.Wrap(err, "read version")
	}

	// Input count
	var countSize, count uint64
	var err error
	countSize, count, err = ReadVarIntN(r, pver)
	if err != nil {
		return 0, errors.Wrap(err, "read input count")
	}
	size += countSize

	// Prevent more input transactions than could possibly fit into a
	// message.  It would be possible to cause memory exhaustion and panics
	// without a sane upper bound on this count.
	if count > uint64(maxTxInPerMessage) {
		str := fmt.Sprintf("too many input transactions to fit into "+
			"max message size [count %d, max %d]", count,
			maxTxInPerMessage)
		return 0, messageError("readTxSize", str)
	}

	// Inputs
	for i := uint64(0); i < count; i++ {
		inputSize, err := readInputSize(r, pver)
		if err != nil {
			return 0, errors.Wrap(err, "read input size")
		}
		size += inputSize
	}

	// Output count
	countSize, count, err = ReadVarIntN(r, pver)
	if err != nil {
		return 0, errors.Wrap(err, "read output count")
	}
	size += countSize

	// Prevent more output transactions than could possibly fit into a
	// message.  It would be possible to cause memory exhaustion and panics
	// without a sane upper bound on this count.
	if count > uint64(maxTxOutPerMessage) {
		str := fmt.Sprintf("too many output transactions to fit into "+
			"max message size [count %d, max %d]", count,
			maxTxOutPerMessage)
		return 0, messageError("readTxSize", str)
	}

	// Outputs
	for i := uint64(0); i < count; i++ {
		outputSize, err := readOutputSize(r, pver)
		if err != nil {
			return 0, errors.Wrap(err, "read output size")
		}
		size += outputSize
	}

	// Lock Time
	if _, err := r.Read(fourbytes[:]); err != nil {
		return 0, errors.Wrap(err, "read lock time")
	}

	return size, nil
}

// readInputSize reads the data for a full bitcoin input and returns its size.
func readInputSize(r io.Reader, pver uint32) (uint64, error) {
	size := uint64(40) // fixed size of outpoint and sequence

	// Outpoint
	var outpoint [36]byte // 36 bytes for txid and index
	if _, err := r.Read(outpoint[:]); err != nil {
		return 0, errors.Wrap(err, "read outpoint")
	}

	// Unlocking script
	scriptSize, err := readScriptSize(r, pver)
	if err != nil {
		return 0, errors.Wrap(err, "read script")
	}
	size += scriptSize

	// Sequence
	var sequence [4]byte
	if _, err := r.Read(sequence[:]); err != nil {
		return 0, errors.Wrap(err, "read sequence")
	}

	return size, nil
}

// readInputSize reads the data for a full bitcoin output and returns its size.
func readOutputSize(r io.Reader, pver uint32) (uint64, error) {
	size := uint64(8) // fixed size of value

	// Value
	var value [8]byte
	_, err := r.Read(value[:])
	if err != nil {
		return 0, errors.Wrap(err, "read value")
	}

	// Locking script
	scriptSize, err := readScriptSize(r, pver)
	if err != nil {
		return 0, errors.Wrap(err, "read script")
	}
	size += scriptSize

	return size, nil
}

// readInputSize reads the data for a full bitcoin script and returns its size.
func readScriptSize(r io.Reader, pver uint32) (uint64, error) {
	var countSize, count uint64
	var err error
	countSize, count, err = ReadVarIntN(r, pver)
	if err != nil {
		return 0, errors.Wrap(err, "read script size")
	}

	script := make([]byte, count)
	if _, err := r.Read(script); err != nil {
		return 0, errors.Wrap(err, "read script data")
	}

	return countSize + count, nil
}

// ReadTxBytes reads a full tx and returns its size and the tx. It uses slices of the slice passed
//   in to construct the slices within the tx.
func ReadTxBytes(b []byte, pver uint32) (uint64, *MsgTx, error) {
	result := &MsgTx{}
	l := uint64(len(b))

	// Version
	if l < 4 {
		return 0, nil, errors.Wrap(io.EOF, "read version")
	}
	result.Version = int32(endian.Uint32(b[:4]))

	offset := uint64(4)

	// Input count
	var countSize, count uint64
	var err error
	countSize, count, err = ReadVarIntBytes(b[offset:], pver)
	if err != nil {
		return 0, nil, errors.Wrap(err, "read input count")
	}
	offset += countSize

	// Inputs
	result.TxIn = make([]*TxIn, 0, count)
	for i := uint64(0); i < count; i++ {
		inputSize, input, err := ReadInputBytes(b[offset:], pver)
		if err != nil {
			return 0, nil, errors.Wrap(err, "read input")
		}

		result.TxIn = append(result.TxIn, input)
		offset += inputSize
	}

	// Output count
	countSize, count, err = ReadVarIntBytes(b[offset:], pver)
	if err != nil {
		return 0, nil, errors.Wrap(err, "read output count")
	}
	offset += countSize

	// Outputs
	result.TxOut = make([]*TxOut, 0, count)
	for i := uint64(0); i < count; i++ {
		outputSize, output, err := ReadOutputBytes(b[offset:], pver)
		if err != nil {
			return 0, nil, errors.Wrap(err, "read output")
		}

		result.TxOut = append(result.TxOut, output)
		offset += outputSize
	}

	// Lock Time
	if l < offset+4 {
		return 0, nil, errors.Wrap(io.EOF, "read lock time")
	}
	result.LockTime = endian.Uint32(b[offset : offset+4])
	offset += 4

	return offset, result, nil
}

func ReadInputBytes(b []byte, pver uint32) (uint64, *TxIn, error) {
	result := &TxIn{}
	offset := uint64(0)
	l := uint64(len(b))

	// Outpoint
	if l < 36 {
		return 0, nil, errors.Wrap(io.EOF, "read outpoint")
	}
	copy(result.PreviousOutPoint.Hash[:], b[offset:offset+bitcoin.Hash32Size])
	offset += uint64(bitcoin.Hash32Size)

	result.PreviousOutPoint.Index = endian.Uint32(b[offset : offset+4])
	offset += 4

	// Unlocking script
	var scriptSize uint64
	var err error
	scriptSize, result.SignatureScript, err = ReadScriptBytes(b[offset:], pver)
	if err != nil {
		return 0, nil, errors.Wrap(err, "read script")
	}
	offset += scriptSize

	// Sequence
	if l < offset+4 {
		return 0, nil, errors.Wrap(io.EOF, "read sequence")
	}
	result.Sequence = endian.Uint32(b[offset : offset+4])
	offset += 4

	return offset, result, nil
}

func ReadOutputBytes(b []byte, pver uint32) (uint64, *TxOut, error) {
	result := &TxOut{}
	offset := uint64(0)
	l := uint64(len(b))

	// Value
	if l < 8 {
		return 0, nil, errors.Wrap(io.EOF, "read value")
	}
	result.Value = endian.Uint64(b[:8])
	offset += 8

	// Locking script
	var scriptSize uint64
	var err error
	scriptSize, result.PkScript, err = ReadScriptBytes(b[offset:], pver)
	if err != nil {
		return 0, nil, errors.Wrap(err, "read script")
	}
	offset += scriptSize

	return offset, result, nil
}

func ReadScriptBytes(b []byte, pver uint32) (uint64, []byte, error) {
	offset := uint64(0)

	countSize, count, err := ReadVarIntBytes(b, pver)
	if err != nil {
		return 0, nil, errors.Wrap(err, "read script size")
	}
	offset += countSize

	if uint64(len(b)) < offset+count {
		return 0, nil, errors.Wrap(io.EOF, "read script")
	}
	script := b[offset : offset+count]
	offset += count

	return offset, script, nil
}

// ReadVarIntBytes reads a variable length integer from a byte slice and returns it's size and value
//   as uint64s.
func ReadVarIntBytes(b []byte, pver uint32) (uint64, uint64, error) {
	discriminant := uint8(b[0])

	switch discriminant {
	case 0xff:
		if len(b) < 9 {
			return 0, 0, io.EOF
		}
		sv := endian.Uint64(b[1:9])

		// The encoding is not canonical if the value could have been
		// encoded using fewer bytes.
		min := uint64(0x100000000)
		if sv < min {
			return 0, 0, messageError("ReadVarInt", fmt.Sprintf(errNonCanonicalVarInt, sv,
				discriminant, min))
		}

		return 9, sv, nil

	case 0xfe:
		if len(b) < 5 {
			return 0, 0, io.EOF
		}
		sv := endian.Uint32(b[1:5])

		// The encoding is not canonical if the value could have been
		// encoded using fewer bytes.
		min := uint32(0x10000)
		if sv < min {
			return 0, 0, messageError("ReadVarInt", fmt.Sprintf(errNonCanonicalVarInt, sv,
				discriminant, min))
		}

		return 5, uint64(sv), nil

	case 0xfd:
		if len(b) < 3 {
			return 0, 0, io.EOF
		}
		sv := endian.Uint16(b[1:3])

		// The encoding is not canonical if the value could have been
		// encoded using fewer bytes.
		min := uint16(0xfd)
		if sv < min {
			return 0, 0, messageError("ReadVarInt", fmt.Sprintf(errNonCanonicalVarInt, sv,
				discriminant, min))
		}

		return 3, uint64(sv), nil

	default:
		return 1, uint64(discriminant), nil
	}
}
