package index

import (
	"encoding/binary"
	"errors"
	"os"
)

var (
	// ErrColumnFamilyNameNil .
	ErrColumnFamilyNameNil = errors.New("column family name is nil")

	// ErrBucketNameNil .
	ErrBucketNameNil = errors.New("bucket name is nil")

	// ErrDirPathNil .
	ErrDirPathNil = errors.New("bptree dir path is nil")

	// ErrBucketNotInit .
	ErrBucketNotInit = errors.New("bucket not init")

	// ErrOptionsTypeNotMatch .
	ErrOptionsTypeNotMatch = errors.New("indexer options not match")
)

type IndexerTx interface{}

type IndexerNode struct {
	Key  []byte
	Meta *IndexerMeta
}

type IndexerType int8

const (
	BptreeBoltDB IndexerType = iota
)

const (
	indexFileSuffixName = ".index"
	metaFileSuffixName  = ".meta"
	separator           = string(os.PathSeparator)
	metaHeaderSize      = 5 + 5 + 10
)

type IndexerMeta struct {
	Value  []byte
	Fid    uint32
	Size   uint32
	Offset int64
}

type Indexer interface {
	Put(key []byte, value []byte) (err error)

	PutBatch(kv []*IndexerNode) (offset int, err error)

	Get(k []byte) (meta *IndexerMeta, err error)

	Delete(key []byte) error

	Close() (err error)

	Iter() (iter IndexerIter, err error)
}

func NewIndexer(opts IndexerOptions) (Indexer, error) {
	switch opts.GetType() {
	case BptreeBoltDB:
		boltOpts, ok := opts.(*BPTreeOptions)
		if !ok {
			return nil, ErrOptionsTypeNotMatch
		}
		return BptreeBolt(boltOpts)
	default:
		panic("unknown indexer type")
	}
}

type IndexerOptions interface {
	SetType(typ IndexerType)
	SetColumnFamilyName(cfName string)
	SetDirPath(dirPath string)
	GetType() IndexerType
	GetColumnFamilyName() string
	GetDirPath() string
}

type IndexerIter interface {
	// First moves the cursor to the first item in the bucket and returns its key and value.
	// If the bucket is empty then a nil key and value are returned.
	First() (key, value []byte)

	// Last moves the cursor to the last item in the bucket and returns its key and value.
	// If the bucket is empty then a nil key and value are returned.
	Last() (key, value []byte)

	// Seek moves the cursor to a given key and returns it.
	// If the key does not exist then the next key is used. If no keys follow, a nil key is returned.
	Seek(seek []byte) (key, value []byte)

	// Next moves the cursor to the next item in the bucket and returns its key and value.
	// If the cursor is at the end of the bucket then a nil key and value are returned.
	Next() (key, value []byte)

	// Prev moves the cursor to the previous item in the bucket and returns its key and value.
	// If the cursor is at the beginning of the bucket then a nil key and value are returned.
	Prev() (key, value []byte)

	// Close The close method must be called after the iterative behavior is over！
	Close() error
}

// EncodeMeta .
func EncodeMeta(m *IndexerMeta) []byte {
	header := make([]byte, metaHeaderSize)
	var index int
	index += binary.PutVarint(header[index:], int64(m.Fid))
	index += binary.PutVarint(header[index:], int64(m.Size))
	index += binary.PutVarint(header[index:], m.Offset)

	if m.Value != nil {
		buf := make([]byte, index+len(m.Value))
		copy(buf[:index], header[:])
		copy(buf[index:], m.Value)
		return buf
	} else {
		return header[:index]
	}
}

func DecodeMeta(buf []byte) *IndexerMeta {
	m := &IndexerMeta{}
	var index int
	fid, n := binary.Varint(buf[index:])
	m.Fid = uint32(fid)
	index += n

	size, n := binary.Varint(buf[index:])
	m.Size = uint32(size)
	index += n

	offset, n := binary.Varint(buf[index:])
	m.Offset = offset
	index += n

	m.Value = buf[index:]
	return m
}
