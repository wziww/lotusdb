package lotusdb

import (
	"errors"
	"os"
	"sync"

	"github.com/flower-corp/lotusdb/util"
)

var (
	// ErrDefaultCfNil .
	ErrDefaultCfNil = errors.New("default comumn family is nil")
)

// LotusDB .
type LotusDB struct {
	cfs     map[string]*ColumnFamily // all column families.
	lockMgr *LockMgr                 // global lock manager that guarantees consistency of read and write.
	opts    Options
	mu      sync.Mutex
}

// Open a new LotusDB instance.
func Open(opt Options) (*LotusDB, error) {
	// add dir lock? todo
	if !util.PathExist(opt.DBPath) {
		if err := os.MkdirAll(opt.DBPath, os.ModePerm); err != nil {
			return nil, err
		}
	}

	db := &LotusDB{opts: opt, cfs: make(map[string]*ColumnFamily)}
	// load default column family.
	if opt.CfOpts.CfName == "" {
		opt.CfOpts.CfName = DefaultColumnFamilyName
	}
	if _, err := db.OpenColumnFamily(opt.CfOpts); err != nil {
		return nil, err
	}
	return db, nil
}

// Close close database.
func (db *LotusDB) Close() error {
	for _, cf := range db.cfs {
		if err := cf.Close(); err != nil {
			return err
		}
	}
	return nil
}

// Put put to default column family.
func (db *LotusDB) Put(key, value []byte) error {
	return db.PutWithOptions(key, value, nil)
}

// PutWithOptions put to default column family with options.
func (db *LotusDB) PutWithOptions(key, value []byte, opt *WriteOptions) error {
	columnFamily := db.getColumnFamily(DefaultColumnFamilyName)
	if columnFamily == nil {
		return ErrDefaultCfNil
	}
	return columnFamily.PutWithOptions(key, value, opt)
}

// Get get from default column family.
func (db *LotusDB) Get(key []byte) ([]byte, error) {
	columnFamily := db.getColumnFamily(DefaultColumnFamilyName)
	if columnFamily == nil {
		return nil, ErrDefaultCfNil
	}
	return columnFamily.Get(key)
}

// Delete delete from default column family.
func (db *LotusDB) Delete(key []byte) error {
	return db.DeleteWithOptions(key, nil)
}

// DeleteWithOptions delete from default column family with options.
func (db *LotusDB) DeleteWithOptions(key []byte, opt *WriteOptions) error {
	columnFamily := db.getColumnFamily(DefaultColumnFamilyName)
	if columnFamily == nil {
		return ErrDefaultCfNil
	}
	return columnFamily.DeleteWithOptions(key, opt)
}

func (db *LotusDB) getColumnFamily(cfName string) *ColumnFamily {
	db.mu.Lock()
	defer db.mu.Unlock()
	return db.cfs[cfName]
}
