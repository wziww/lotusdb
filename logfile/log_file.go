package logfile

import (
	"errors"
	"fmt"
	"hash/crc32"
	"io"
	"os"
	"sync"

	"github.com/flower-corp/lotusdb/ioselector"
)

var (
	ErrInvalidCrc        = errors.New("logfile: invalid crc")
	ErrWriteSizeNotEqual = errors.New("write size is not equal to entry size")
)

const (
	// PathSeparator default path separator.
	PathSeparator = string(os.PathSeparator)

	// WalSuffixName log file suffix name of write ahead log.
	WalSuffixName = ".wal"

	// VLogSuffixName log file suffix name of value log.
	VLogSuffixName = ".vlog"

	InitialLogFileId = 0
)

// FileType log file of wal and value log.
type FileType int8

const (
	// WAL write ahead log.
	WAL FileType = iota

	// ValueLog value log.
	ValueLog
)

type IOType int8

const (
	FileIO IOType = iota
	MMap
)

type LogFile struct {
	sync.RWMutex
	Fid        uint32
	WriteAt    int64
	IoSelector ioselector.IOSelector
}

func OpenLogFile(path string, fid uint32, fsize int64, ftype FileType, ioType IOType) (lf *LogFile, err error) {
	lf = &LogFile{Fid: fid}
	fileName := lf.getLogFileName(path, fid, ftype)

	var selector ioselector.IOSelector
	switch ioType {
	case FileIO:
		if selector, err = ioselector.NewFileIOSelector(fileName, fsize); err != nil {
			return
		}
	case MMap:
		if selector, err = ioselector.NewMMapSelector(fileName, fsize); err != nil {
			return
		}
	default:
		panic(fmt.Sprintf("unsupported io type : %d", ioType))
	}

	lf.IoSelector = selector
	return
}

// ReadLogEntry .
func (lf *LogFile) ReadLogEntry(offset int64) (*LogEntry, int64, error) {
	// read entry header.
	headerBuf, err := lf.readBytes(offset, maxHeaderSize)
	if err != nil {
		return nil, 0, err
	}
	header, size := decodeHeader(headerBuf)

	kSize, vSize := int64(header.kSize), int64(header.vSize)
	if kSize == 0 && vSize == 0 {
		return nil, 0, io.EOF
	}

	var entrySize = size + kSize + vSize
	// read entry key and value.
	kvBuf, err := lf.readBytes(offset+size, kSize+vSize)
	if err != nil {
		return nil, 0, err
	}

	e := &LogEntry{
		Key:       kvBuf[:kSize],
		Value:     kvBuf[kSize:],
		ExpiredAt: header.expiredAt,
		Type:      header.typ,
	}
	// crc32 check.
	if crc := getEntryCrc(e, headerBuf[crc32.Size:size]); crc != header.crc32 {
		return nil, 0, ErrInvalidCrc
	}
	lf.WriteAt += entrySize
	return e, entrySize, nil
}

// Read .
func (lf *LogFile) Read(offset int64, size uint32) ([]byte, error) {
	buf := make([]byte, size)
	if _, err := lf.IoSelector.Read(buf, offset); err != nil {
		return nil, err
	}
	return buf, nil
}

// Write .
func (lf *LogFile) Write(buf []byte) error {
	n, err := lf.IoSelector.Write(buf, lf.WriteAt)
	if err != nil {
		return err
	}
	if n != len(buf) {
		return ErrWriteSizeNotEqual
	}
	lf.WriteAt += int64(n)
	return nil
}

// Sync .
func (lf *LogFile) Sync() error {
	return lf.IoSelector.Sync()
}

// Close .
func (lf *LogFile) Close() error {
	return lf.IoSelector.Close()
}

func (lf *LogFile) Delete() error {
	return lf.IoSelector.Delete()
}

func (lf *LogFile) readBytes(offset, n int64) (buf []byte, err error) {
	buf = make([]byte, n)
	_, err = lf.IoSelector.Read(buf, offset)
	return
}

func (lf *LogFile) getLogFileName(path string, fid uint32, ftype FileType) string {
	fname := path + PathSeparator + fmt.Sprintf("%09d", fid)
	switch ftype {
	case WAL:
		return fname + WalSuffixName
	case ValueLog:
		return fname + VLogSuffixName
	default:
		panic(fmt.Sprintf("unsupported log file type: %d", ftype))
	}
}
