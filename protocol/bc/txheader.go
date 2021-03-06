package bc

import "io"

// TxHeader contains header information for a transaction. Every
// transaction on a blockchain contains exactly one TxHeader. The ID
// of the TxHeader is the ID of the transaction. TxHeader satisfies
// the Entry interface.

func (TxHeader) typ() string { return "txheader" }
func (h *TxHeader) writeForHash(w io.Writer) {
	mustWriteForHash(w, h.Version)
	mustWriteForHash(w, h.TimeRange)
	mustWriteForHash(w, h.ResultIds)
	if h.Side {
		mustWriteForHash(w, h.Data)
	}
}

// NewTxHeader creates an new TxHeader.
func NewTxHeader(version, serializedSize uint64, data *Hash, timeRange uint64, resultIDs []*Hash, side bool) *TxHeader {
	return &TxHeader{
		Version:        version,
		SerializedSize: serializedSize,
		Data:           data,
		TimeRange:      timeRange,
		ResultIds:      resultIDs,
		Side:           side,
	}
}
