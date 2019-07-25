package txbuilder

import (
	"bytes"
	"encoding/binary"
	"fmt"

	"github.com/tokenized/smart-contract/pkg/bitcoin"
	"github.com/tokenized/smart-contract/pkg/wire"
)

// SigHashType represents hash type bits at the end of a signature.
type SigHashType uint32

const (
	SigHashOld          SigHashType = 0x0
	SigHashAll          SigHashType = 0x1
	SigHashNone         SigHashType = 0x2
	SigHashSingle       SigHashType = 0x3
	SigHashAnyOneCanPay SigHashType = 0x80
	SigHashForkID       SigHashType = 0x40

	// sigHashMask defines the number of bits of the hash type which is used to identify which
	//   outputs are signed.
	sigHashMask = 0x1f
)

// SigHashCache allows caching of previously calculated hashes used to calculate the signature hash
//   for signing tx inputs.
// This allows validation to re-use previous hashing computation, reducing the complexity of
//   validating SigHashAll inputs rom  O(N^2) to O(N).
type SigHashCache struct {
	hashPrevOuts []byte
	hashSequence []byte
	hashOutputs  []byte
}

// Clear resets all the hashes. This should be used if anything in the transaction changes and the
//   signatures need to be recalculated.
func (shc *SigHashCache) Clear() {
	shc.hashPrevOuts = nil
	shc.hashSequence = nil
	shc.hashOutputs = nil
}

// HashPrevOuts calculates a single hash of all the previous outputs (txid:index) referenced within
//   the specified transaction.
func (shc *SigHashCache) HashPrevOuts(tx *wire.MsgTx) []byte {
	if shc.hashPrevOuts != nil {
		return shc.hashPrevOuts
	}

	var b bytes.Buffer
	for _, in := range tx.TxIn {
		// First write out the 32-byte transaction ID.
		b.Write(in.PreviousOutPoint.Hash[:])

		// Next, we'll encode the index of the referenced output as a little endian integer.
		var buf [4]byte
		binary.LittleEndian.PutUint32(buf[:], in.PreviousOutPoint.Index)
		b.Write(buf[:])
	}

	shc.hashPrevOuts = bitcoin.DoubleSha256(b.Bytes())
	return shc.hashPrevOuts
}

// HashSequence computes an aggregated hash of each of the sequence numbers within the inputs of the
//   passed transaction.
func (shc *SigHashCache) HashSequence(tx *wire.MsgTx) []byte {
	if shc.hashSequence != nil {
		return shc.hashSequence
	}

	var b bytes.Buffer
	for _, in := range tx.TxIn {
		var buf [4]byte
		binary.LittleEndian.PutUint32(buf[:], in.Sequence)
		b.Write(buf[:])
	}

	shc.hashSequence = bitcoin.DoubleSha256(b.Bytes())
	return shc.hashSequence
}

// HashOutputs computes a hash digest of all outputs created by the transaction encoded using the
//   wire format.
func (shc *SigHashCache) HashOutputs(tx *wire.MsgTx) []byte {
	if shc.hashOutputs != nil {
		return shc.hashOutputs
	}

	var b bytes.Buffer
	for _, out := range tx.TxOut {
		out.Serialize(&b, 0, 0)
	}

	shc.hashOutputs = bitcoin.DoubleSha256(b.Bytes())
	return shc.hashOutputs
}

// signatureHash computes the hash to be signed for a transaction's input using the new, optimized
//   digest calculation algorithm defined in BIP0143:
//   https://github.com/bitcoin/bips/blob/master/bip-0143.mediawiki.
// This function makes use of pre-calculated hash fragments stored within the passed SigHashCache to
//   eliminate duplicate hashing computations when calculating the final digest, reducing the
//   complexity from O(N^2) to O(N).
// Additionally, signatures now cover the input value of the referenced unspent output. This allows
//   offline, or hardware wallets to compute the exact amount being spent, in addition to the final
//   transaction fee. In the case the wallet if fed an invalid input amount, the real sighash will
//   differ causing the produced signature to be invalid.
func signatureHash(tx *wire.MsgTx, index int, lockScript []byte, value uint64,
	hashType SigHashType, hashCache *SigHashCache) []byte {

	// As a sanity check, ensure the passed input index for the transaction is valid.
	if index > len(tx.TxIn)-1 {
		fmt.Printf("signatureHash error: index %d but %d txins",
			index, len(tx.TxIn))
		return nil
	}

	// Buffer for data to be hashed.
	var buf bytes.Buffer

	// First write out, then encode the transaction's version number.
	var bVersion [4]byte
	binary.LittleEndian.PutUint32(bVersion[:], uint32(tx.Version))
	buf.Write(bVersion[:])

	// Next write out the possibly pre-calculated hashes for the sequence
	// numbers of all inputs, and the hashes of the previous outs for all
	// outputs.
	var zeroHash [32]byte

	// If anyone can pay is active we just write zeroes for the prev outs hash.
	if hashType&SigHashAnyOneCanPay == 0 {
		buf.Write(hashCache.HashPrevOuts(tx))
	} else {
		buf.Write(zeroHash[:])
	}

	// If the sighash is anyone can pay, single, or none we write all zeroes for the sequence hash.
	if hashType&SigHashAnyOneCanPay == 0 &&
		hashType&sigHashMask != SigHashSingle &&
		hashType&sigHashMask != SigHashNone {
		buf.Write(hashCache.HashSequence(tx))
	} else {
		buf.Write(zeroHash[:])
	}

	// Next, write the outpoint being spent.
	buf.Write(tx.TxIn[index].PreviousOutPoint.Hash[:])
	var bIndex [4]byte
	binary.LittleEndian.PutUint32(bIndex[:], tx.TxIn[index].PreviousOutPoint.Index)
	buf.Write(bIndex[:])

	// Write the locking script being spent.
	wire.WriteVarBytes(&buf, 0, lockScript)

	// Next, add the input amount, and sequence number of the input being signed.
	var bAmount [8]byte
	binary.LittleEndian.PutUint64(bAmount[:], value)
	buf.Write(bAmount[:])
	var bSequence [4]byte
	binary.LittleEndian.PutUint32(bSequence[:], tx.TxIn[index].Sequence)
	buf.Write(bSequence[:])

	// If the current signature mode is single, or none, then we'll serialize and add only the
	//   target output index to the signature pre-image.
	if hashType&SigHashSingle != SigHashSingle &&
		hashType&SigHashNone != SigHashNone {
		buf.Write(hashCache.HashOutputs(tx))
	} else if hashType&sigHashMask == SigHashSingle && index < len(tx.TxOut) {
		var b bytes.Buffer
		tx.TxOut[index].Serialize(&b, 0, 0)
		buf.Write(bitcoin.DoubleSha256(b.Bytes()))
	} else {
		buf.Write(zeroHash[:])
	}

	// Finally, write out the transaction's locktime, and the sig hash type.
	var bLockTime [4]byte
	binary.LittleEndian.PutUint32(bLockTime[:], tx.LockTime)
	buf.Write(bLockTime[:])
	var bHashType [4]byte
	binary.LittleEndian.PutUint32(bHashType[:], uint32(hashType|SigHashForkID))
	buf.Write(bHashType[:])

	return bitcoin.DoubleSha256(buf.Bytes())
}
