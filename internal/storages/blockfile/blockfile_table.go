package blockfile

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"

	"github.com/sirupsen/logrus"
)

type BlockTable struct {
	items uint64

	name        string
	path        string
	maxFileSize uint32 // Max file size for data-files

	head   *os.File            // File descriptor for the data head of the table
	index  *os.File            // File description
	files  map[uint32]*os.File // open files
	headId uint32              // number of the currently active head file
	tailId uint32              // number of the earliest file

	headBytes  uint32 // Number of bytes written to the head file
	itemOffset uint32 // Offset (number of discarded items)

	logger logrus.FieldLogger
	lock   sync.RWMutex // Mutex protecting the data file descriptors
}

type indexEntry struct {
	filenum uint32 // stored as uint16 ( 2 bytes)
	offset  uint32 // stored as uint32 ( 4 bytes)
}

const indexEntrySize = 6

// unmarshallBinary deserializes binary b into the rawIndex entry.
func (i *indexEntry) unmarshalBinary(b []byte) error {
	i.filenum = uint32(binary.BigEndian.Uint16(b[:2]))
	i.offset = binary.BigEndian.Uint32(b[2:6])
	return nil
}

// marshallBinary serializes the rawIndex entry into binary.
func (i *indexEntry) marshallBinary() []byte {
	b := make([]byte, indexEntrySize)
	binary.BigEndian.PutUint16(b[:2], uint16(i.filenum))
	binary.BigEndian.PutUint32(b[2:6], i.offset)
	return b
}

func newTable(path string, name string, maxFilesize uint32, logger logrus.FieldLogger) (*BlockTable, error) {
	if err := os.MkdirAll(path, 0755); err != nil {
		return nil, err
	}
	idxName := fmt.Sprintf("%s.ridx", name)
	offsets, err := openBlockFileForAppend(filepath.Join(path, idxName))
	if err != nil {
		return nil, err
	}
	table := &BlockTable{
		index:       offsets,
		files:       make(map[uint32]*os.File),
		name:        name,
		path:        path,
		maxFileSize: maxFilesize,
		logger:      logger,
	}
	if err := table.repair(); err != nil {
		return nil, err
	}
	return table, nil
}

func (b *BlockTable) repair() error {
	buffer := make([]byte, indexEntrySize)

	stat, err := b.index.Stat()
	if err != nil {
		return err
	}
	if stat.Size() == 0 {
		if _, err := b.index.Write(buffer); err != nil {
			return err
		}
	}
	if remainder := stat.Size() % indexEntrySize; remainder != 0 {
		err := truncateBlockFile(b.index, stat.Size()-remainder)
		if err != nil {
			return err
		}
	}
	if stat, err = b.index.Stat(); err != nil {
		return err
	}
	offsetsSize := stat.Size()

	// Open the head file
	var (
		firstIndex  indexEntry
		lastIndex   indexEntry
		contentSize int64
		contentExp  int64
	)
	// Read index zero, determine what file is the earliest
	// and what item offset to use
	_, err = b.index.ReadAt(buffer, 0)
	if err != nil {
		return err
	}
	err = firstIndex.unmarshalBinary(buffer)
	if err != nil {
		return err
	}

	b.tailId = firstIndex.filenum
	b.itemOffset = firstIndex.offset

	_, err = b.index.ReadAt(buffer, offsetsSize-indexEntrySize)
	if err != nil {
		return err
	}
	err = lastIndex.unmarshalBinary(buffer)
	if err != nil {
		return err
	}
	b.head, err = b.openFile(lastIndex.filenum, openBlockFileForAppend)
	if err != nil {
		return err
	}
	if stat, err = b.head.Stat(); err != nil {
		return err
	}
	contentSize = stat.Size()

	// Keep truncating both files until they come in sync
	contentExp = int64(lastIndex.offset)

	for contentExp != contentSize {
		b.logger.WithFields(logrus.Fields{
			"indexed": contentExp,
			"stored":  contentSize,
		}).Warn("Truncating dangling head")
		if contentExp < contentSize {
			if err := truncateBlockFile(b.head, contentExp); err != nil {
				return err
			}
		}
		if contentExp > contentSize {
			b.logger.WithFields(logrus.Fields{
				"indexed": contentExp,
				"stored":  contentSize,
			}).Warn("Truncating dangling indexes")
			offsetsSize -= indexEntrySize
			_, err = b.index.ReadAt(buffer, offsetsSize-indexEntrySize)
			if err != nil {
				return err
			}
			var newLastIndex indexEntry
			err = newLastIndex.unmarshalBinary(buffer)
			if err != nil {
				return err
			}
			// We might have slipped back into an earlier head-file here
			if newLastIndex.filenum != lastIndex.filenum {
				// Release earlier opened file
				b.releaseFile(lastIndex.filenum)
				if b.head, err = b.openFile(newLastIndex.filenum, openBlockFileForAppend); err != nil {
					return err
				}
				if stat, err = b.head.Stat(); err != nil {
					// TODO, anything more we can do here?
					// A data file has gone missing...
					return err
				}
				contentSize = stat.Size()
			}
			lastIndex = newLastIndex
			contentExp = int64(lastIndex.offset)
		}
	}
	// Ensure all reparation changes have been written to disk
	if err := b.index.Sync(); err != nil {
		return err
	}
	if err := b.head.Sync(); err != nil {
		return err
	}
	// Update the item and byte counters and return
	b.items = uint64(b.itemOffset) + uint64(offsetsSize/indexEntrySize-1) // last indexEntry points to the end of the data file
	b.headBytes = uint32(contentSize)
	b.headId = lastIndex.filenum

	// Close opened files and preopen all files
	if err := b.preopen(); err != nil {
		return err
	}
	b.logger.WithFields(logrus.Fields{
		"items": b.items,
		"size":  b.headBytes,
	}).Debug("Chain freezer table opened")
	return nil
}

// truncate discards any recent data above the provided threshold number.
func (b *BlockTable) truncate(items uint64) error {
	b.lock.Lock()
	defer b.lock.Unlock()

	existing := atomic.LoadUint64(&b.items)
	if existing <= items {
		return nil
	}

	b.logger.WithFields(logrus.Fields{
		"items": existing,
		"limit": items,
	}).Warn("Truncating block file")
	if err := truncateBlockFile(b.index, int64(items+1)*indexEntrySize); err != nil {
		return err
	}
	// Calculate the new expected size of the data file and truncate it
	buffer := make([]byte, indexEntrySize)
	if _, err := b.index.ReadAt(buffer, int64(items*indexEntrySize)); err != nil {
		return err
	}
	var expected indexEntry
	err := expected.unmarshalBinary(buffer)
	if err != nil {
		return err
	}

	// We might need to truncate back to older files
	if expected.filenum != b.headId {
		// If already open for reading, force-reopen for writing
		b.releaseFile(expected.filenum)
		newHead, err := b.openFile(expected.filenum, openBlockFileForAppend)
		if err != nil {
			return err
		}
		// Release any files _after the current head -- both the previous head
		// and any files which may have been opened for reading
		b.releaseFilesAfter(expected.filenum, true)
		// Set back the historic head
		b.head = newHead
		atomic.StoreUint32(&b.headId, expected.filenum)
	}
	if err := truncateBlockFile(b.head, int64(expected.offset)); err != nil {
		return err
	}
	// All data files truncated, set internal counters and return
	atomic.StoreUint64(&b.items, items)
	atomic.StoreUint32(&b.headBytes, expected.offset)

	return nil
}

func (b *BlockTable) Retrieve(item uint64) ([]byte, error) {
	b.lock.RLock()

	if b.index == nil || b.head == nil {
		b.lock.RUnlock()
		return nil, fmt.Errorf("closed")
	}
	if atomic.LoadUint64(&b.items) <= item {
		b.lock.RUnlock()
		return nil, fmt.Errorf("out of bounds")
	}
	if uint64(b.itemOffset) > item {
		b.lock.RUnlock()
		return nil, fmt.Errorf("out of bounds")
	}
	startOffset, endOffset, filenum, err := b.getBounds(item - uint64(b.itemOffset))
	if err != nil {
		b.lock.RUnlock()
		return nil, err
	}
	dataFile, exist := b.files[filenum]
	if !exist {
		b.lock.RUnlock()
		return nil, fmt.Errorf("missing data file %d", filenum)
	}
	blob := make([]byte, endOffset-startOffset)
	if _, err := dataFile.ReadAt(blob, int64(startOffset)); err != nil {
		b.lock.RUnlock()
		return nil, err
	}
	b.lock.RUnlock()

	return blob, nil
}

func (b *BlockTable) Append(item uint64, blob []byte) error {
	b.lock.RLock()
	if b.index == nil || b.head == nil {
		b.lock.RUnlock()
		return fmt.Errorf("closed")
	}
	if atomic.LoadUint64(&b.items) != item {
		b.lock.RUnlock()
		return fmt.Errorf("appending unexpected item: want %d, have %d", b.items, item)
	}
	bLen := uint32(len(blob))
	if b.headBytes+bLen < bLen ||
		b.headBytes+bLen > b.maxFileSize {
		b.lock.RUnlock()
		b.lock.Lock()
		nextID := atomic.LoadUint32(&b.headId) + 1
		// We open the next file in truncated mode -- if this file already
		// exists, we need to start over from scratch on it
		newHead, err := b.openFile(nextID, openBlockFileTruncated)
		if err != nil {
			b.lock.Unlock()
			return err
		}
		// Close old file, and reopen in RDONLY mode
		b.releaseFile(b.headId)
		_, err = b.openFile(b.headId, openBlockFileForReadOnly)
		if err != nil {
			return err
		}

		// Swap out the current head
		b.head = newHead
		atomic.StoreUint32(&b.headBytes, 0)
		atomic.StoreUint32(&b.headId, nextID)
		b.lock.Unlock()
		b.lock.RLock()
	}

	defer b.lock.RUnlock()
	if _, err := b.head.Write(blob); err != nil {
		return err
	}
	newOffset := atomic.AddUint32(&b.headBytes, bLen)
	idx := indexEntry{
		filenum: atomic.LoadUint32(&b.headId),
		offset:  newOffset,
	}
	// Write indexEntry
	_, _ = b.index.Write(idx.marshallBinary())

	atomic.AddUint64(&b.items, 1)
	return nil
}

func (b *BlockTable) getBounds(item uint64) (uint32, uint32, uint32, error) {
	buffer := make([]byte, indexEntrySize)
	var startIdx, endIdx indexEntry
	if _, err := b.index.ReadAt(buffer, int64((item+1)*indexEntrySize)); err != nil {
		return 0, 0, 0, err
	}
	if err := endIdx.unmarshalBinary(buffer); err != nil {
		return 0, 0, 0, err
	}
	if item != 0 {
		if _, err := b.index.ReadAt(buffer, int64(item*indexEntrySize)); err != nil {
			return 0, 0, 0, err
		}
		if err := startIdx.unmarshalBinary(buffer); err != nil {
			return 0, 0, 0, err
		}
	} else {
		// the first reading
		return 0, endIdx.offset, endIdx.filenum, nil
	}
	if startIdx.filenum != endIdx.filenum {
		return 0, endIdx.offset, endIdx.filenum, nil
	}
	return startIdx.offset, endIdx.offset, endIdx.filenum, nil
}

func (b *BlockTable) preopen() (err error) {
	b.releaseFilesAfter(0, false)

	for i := b.tailId; i < b.headId; i++ {
		if _, err = b.openFile(i, openBlockFileForReadOnly); err != nil {
			return err
		}
	}
	b.head, err = b.openFile(b.headId, openBlockFileForAppend)
	return err
}

func (b *BlockTable) openFile(num uint32, opener func(string) (*os.File, error)) (f *os.File, err error) {
	var exist bool
	if f, exist = b.files[num]; !exist {
		name := fmt.Sprintf("%s.%04d.rdat", b.name, num)
		f, err = opener(filepath.Join(b.path, name))
		if err != nil {
			return nil, err
		}
		b.files[num] = f
	}
	return f, err
}

// Close closes all opened files.
func (b *BlockTable) Close() error {
	b.lock.Lock()
	defer b.lock.Unlock()

	var errs []error
	if err := b.index.Close(); err != nil {
		errs = append(errs, err)
	}
	b.index = nil

	for _, f := range b.files {
		if err := f.Close(); err != nil {
			errs = append(errs, err)
		}
	}
	b.head = nil

	if errs != nil {
		return fmt.Errorf("%v", errs)
	}
	return nil
}

func (b *BlockTable) releaseFilesAfter(num uint32, remove bool) {
	for fnum, f := range b.files {
		if fnum > num {
			delete(b.files, fnum)
			f.Close()
			if remove {
				os.Remove(f.Name())
			}
		}
	}
}

func (b *BlockTable) releaseFile(num uint32) {
	if f, exist := b.files[num]; exist {
		delete(b.files, num)
		f.Close()
	}
}

func truncateBlockFile(file *os.File, size int64) error {
	if err := file.Truncate(size); err != nil {
		return err
	}
	// Seek to end for append
	if _, err := file.Seek(0, io.SeekEnd); err != nil {
		return err
	}
	return nil
}

func openBlockFileForAppend(filename string) (*os.File, error) {
	file, err := os.OpenFile(filename, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return nil, err
	}
	// Seek to end for append
	if _, err = file.Seek(0, io.SeekEnd); err != nil {
		return nil, err
	}
	return file, nil
}

func openBlockFileTruncated(filename string) (*os.File, error) {
	return os.OpenFile(filename, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
}

func openBlockFileForReadOnly(filename string) (*os.File, error) {
	return os.OpenFile(filename, os.O_RDONLY, 0644)
}
