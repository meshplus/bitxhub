package etcdraft

import (
	"fmt"
	"go.uber.org/atomic"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/coreos/etcd/pkg/fileutil"
	"github.com/coreos/etcd/raft"
	"github.com/coreos/etcd/raft/raftpb"
	"github.com/coreos/etcd/snap"
	"github.com/coreos/etcd/wal"
	"github.com/coreos/etcd/wal/walpb"
	"github.com/meshplus/bitxhub-kit/storage"
	"github.com/meshplus/bitxhub-kit/storage/leveldb"
	"github.com/pkg/errors"
)

// MaxSnapshotFiles defines max number of etcd/raft snapshot files to retain
// on filesystem. Snapshot files are read from newest to oldest, until first
// intact file is found. The more snapshot files we keep around, the more we
// mitigate the impact of a corrupted snapshots. This is exported for testing
// purpose. This MUST be greater equal than 1.
var MaxSnapshotFiles = 5

var restart atomic.Bool

var appliedDbKey = []byte("applied")

// MemoryStorage is currently backed by etcd/raft.MemoryStorage. This interface is
// defined to expose dependencies of fsm so that it may be swapped in the
// future. TODO(jay) Add other necessary methods to this interface once we need
// them in implementation, e.g. ApplySnapshot.
type MemoryStorage interface {
	raft.Storage
	Append(entries []raftpb.Entry) error
	SetHardState(st raftpb.HardState) error
	CreateSnapshot(i uint64, cs *raftpb.ConfState, data []byte) (raftpb.Snapshot, error)
	Compact(compactIndex uint64) error
	ApplySnapshot(snap raftpb.Snapshot) error
}

// RaftStorage encapsulates storages needed for etcd/raft data, i.e. memory, wal
type RaftStorage struct {
	SnapshotCatchUpEntries uint64

	walDir  string
	snapDir string

	lg   raft.Logger
	ram  MemoryStorage
	wal  *wal.WAL
	snap *snap.Snapshotter
	db   storage.Storage // Persistent storage for last-applied raft index
	// a queue that keeps track of indices of snapshots on disk
	snapshotIndex []uint64
}

// CreateStorage attempts to create a storage to persist etcd/raft data.
// If data presents in specified disk, they are loaded to reconstruct storage state.
func CreateStorage(
	lg raft.Logger,
	walDir string,
	snapDir string,
	dbDir string,
	ram MemoryStorage,
) (*RaftStorage, storage.Storage, error) {

	sn, err := createSnapshotter(snapDir)
	if err != nil {
		return nil, nil, fmt.Errorf("create snapshotter failed: %w", err)
	}

	snapshot, err := sn.Load()
	if err != nil {
		if err == snap.ErrNoSnapshot {
			lg.Debugf("No snapshot found at %s", snapDir)
		} else {
			return nil, nil, errors.Errorf("failed to load snapshot: %s", err)
		}
	} else {
		// snapshot found
		lg.Debugf("Loaded snapshot at Term %d and Index %d, Nodes: %+v", snapshot.Metadata.Term, snapshot.Metadata.Index, snapshot.Metadata.ConfState.Nodes)
	}

	w, st, ents, err := createOrReadWAL(lg, walDir, snapshot)
	if err != nil {
		return nil, nil, errors.Errorf("failed to create or read WAL: %s", err)
	}

	if snapshot != nil {
		lg.Debugf("Applying snapshot to raft MemoryStorage")
		if err := ram.ApplySnapshot(*snapshot); err != nil {
			return nil, nil, errors.Errorf("Failed to apply snapshot to memory: %s", err)
		}
	}

	lg.Debugf("Setting HardState to {Term: %d, Commit: %d}", st.Term, st.Commit)
	// MemoryStorage.SetHardState always returns nil
	if err := ram.SetHardState(st); err != nil {
		panic(err)
	}

	lg.Debugf("Appending %d entries to memory storage", len(ents))
	// MemoryStorage.Append always return nil
	if err := ram.Append(ents); err != nil {
		panic(err)
	}

	db, err := leveldb.New(dbDir)
	if err != nil {
		return nil, nil, errors.Errorf("Failed to new leveldb: %s", err)
	}
	return &RaftStorage{
		lg:            lg,
		ram:           ram,
		wal:           w,
		snap:          sn,
		walDir:        walDir,
		snapDir:       snapDir,
		db:            db,
		snapshotIndex: ListSnapshots(lg, snapDir),
	}, db, nil
}

// ListSnapshots returns a list of RaftIndex of snapshots stored on disk.
// If a file is corrupted, rename the file.
func ListSnapshots(logger raft.Logger, snapDir string) []uint64 {
	dir, err := os.Open(snapDir)
	if err != nil {
		logger.Errorf("Failed to open snapshot directory %s: %s", snapDir, err)
		return nil
	}
	defer dir.Close()

	filenames, err := dir.Readdirnames(-1)
	if err != nil {
		logger.Errorf("Failed to read snapshot files: %s", err)
		return nil
	}

	snapfiles := []string{}
	for i := range filenames {
		if strings.HasSuffix(filenames[i], ".snap") {
			snapfiles = append(snapfiles, filenames[i])
		}
	}
	sort.Strings(snapfiles)

	var snapshots []uint64
	for _, snapfile := range snapfiles {
		fpath := filepath.Join(snapDir, snapfile)
		s, err := snap.Read(fpath)
		if err != nil {
			logger.Errorf("Snapshot file %s is corrupted: %s", fpath, err)
			continue
		}
		snapshots = append(snapshots, s.Metadata.Index)
	}

	return snapshots
}

func createSnapshotter(snapDir string) (*snap.Snapshotter, error) {
	if !fileutil.Exist(snapDir) {
		if err := os.MkdirAll(snapDir, os.ModePerm); err != nil {
			return nil, errors.Errorf("failed to mkdir '%s' for snapshot: %s", snapDir, err)
		}
	}
	return snap.New(snapDir), nil
}

func createOrReadWAL(lg raft.Logger, walDir string, snapshot *raftpb.Snapshot) (w *wal.WAL, st raftpb.HardState, ents []raftpb.Entry, err error) {
	if !wal.Exist(walDir) {
		lg.Infof("No WAL data found, creating new WAL at path '%s'", walDir)
		// TODO(jay_guo) add metadata to be persisted with wal once we need it.
		// use case could be data dump and restore on a new node.
		w, err := wal.Create(walDir, nil)
		if err != nil {
			return nil, st, nil, errors.Errorf("failed to initialize WAL: %s", err)
		}

		if err = w.Close(); err != nil {
			return nil, st, nil, errors.Errorf("failed to close the WAL just created: %s", err)
		}
	} else {
		restart.Store(true)
		lg.Infof("Found WAL data at path '%s', replaying it", walDir)
	}

	walsnap := walpb.Snapshot{}
	if snapshot != nil {
		walsnap.Index, walsnap.Term = snapshot.Metadata.Index, snapshot.Metadata.Term
	}

	lg.Debugf("Loading WAL at Term %d and Index %d", walsnap.Term, walsnap.Index)

	if w, err = wal.Open(walDir, walsnap); err != nil {
		return nil, st, nil, errors.Errorf("failed to open WAL: %s", err)
	}

	if _, st, ents, err = w.ReadAll(); err != nil {
		return nil, st, nil, errors.Errorf("failed to read WAL: %s", err)
	}

	return w, st, ents, nil
}

// Store persists etcd/raft data
func (rs *RaftStorage) Store(entries []raftpb.Entry, hardstate raftpb.HardState, snapshot raftpb.Snapshot) error {
	if err := rs.wal.Save(hardstate, entries); err != nil {
		return fmt.Errorf("save hardstate failed: %w", err)
	}

	if !raft.IsEmptySnap(snapshot) {
		if err := rs.saveSnap(snapshot); err != nil {
			return fmt.Errorf("save snapshot failed: %w", err)
		}

		if err := rs.ram.ApplySnapshot(snapshot); err != nil && err != raft.ErrSnapOutOfDate {
			return fmt.Errorf("apply snapshot failed: %w", err)
		}
	}

	if err := rs.ram.Append(entries); err != nil {
		return fmt.Errorf("append entries to raft storage failed: %w", err)
	}

	return nil
}

func (rs *RaftStorage) saveSnap(snap raftpb.Snapshot) error {
	rs.lg.Infof("Persisting snapshot (term: %d, index: %d) to WAL and disk", snap.Metadata.Term, snap.Metadata.Index)

	// must save the snapshot index to the WAL before saving the
	// snapshot to maintain the invariant that we only Open the
	// wal at previously-saved snapshot indexes.
	walsnap := walpb.Snapshot{
		Index: snap.Metadata.Index,
		Term:  snap.Metadata.Term,
	}

	if err := rs.wal.SaveSnapshot(walsnap); err != nil {
		return errors.Errorf("failed to save snapshot to WAL: %s", err)
	}

	if err := rs.snap.SaveSnap(snap); err != nil {
		return errors.Errorf("failed to save snapshot to disk: %s", err)
	}

	rs.lg.Debugf("Releasing lock to wal files prior to %d", snap.Metadata.Index)
	if err := rs.wal.ReleaseLockTo(snap.Metadata.Index); err != nil {
		return err
	}

	return nil
}

// TakeSnapshot takes a snapshot at index i from MemoryStorage, and persists it to wal and disk.
func (rs *RaftStorage) TakeSnapshot(i uint64, cs raftpb.ConfState, data []byte) error {
	rs.lg.Debugf("Creating snapshot at index %d from MemoryStorage", i)
	snap, err := rs.ram.CreateSnapshot(i, &cs, data)
	if err != nil {
		return errors.Errorf("failed to create snapshot from MemoryStorage: %s", err)
	}

	if err = rs.saveSnap(snap); err != nil {
		return err
	}

	rs.snapshotIndex = append(rs.snapshotIndex, snap.Metadata.Index)

	// Keep some entries in memory for slow followers to catchup
	if i > rs.SnapshotCatchUpEntries {
		compacti := i - rs.SnapshotCatchUpEntries
		rs.lg.Debugf("Purging in-memory raft entries prior to %d", compacti)
		if err = rs.ram.Compact(compacti); err != nil && err != raft.ErrCompacted {
			return err
		}
	}

	rs.lg.Infof("Snapshot is taken at index %d", i)

	rs.gc()
	return nil
}

// gc collects etcd/raft garbage files, namely wal and snapshot files
func (rs *RaftStorage) gc() {
	if len(rs.snapshotIndex) < MaxSnapshotFiles {
		rs.lg.Debugf("Snapshots on disk (%d) < limit (%d), no need to purge wal/snapshot",
			len(rs.snapshotIndex), MaxSnapshotFiles)
		return
	}

	rs.snapshotIndex = rs.snapshotIndex[len(rs.snapshotIndex)-MaxSnapshotFiles:]

	rs.purgeWAL()
	rs.purgeSnap()
}

func (rs *RaftStorage) purgeWAL() {
	retain := rs.snapshotIndex[0]

	walFiles, err := fileutil.ReadDir(rs.walDir)
	if err != nil {
		rs.lg.Errorf("Failed to read WAL directory %s: %s", rs.walDir, err)
	}

	var files []string
	for _, f := range walFiles {
		if !strings.HasSuffix(f, ".wal") {
			continue
		}

		var seq, index uint64
		fmt.Sscanf(f, "%016x-%016x.wal", &seq, &index)
		if index >= retain {
			break
		}

		files = append(files, filepath.Join(rs.walDir, f))
	}

	if len(files) <= 1 {
		// we need to keep one wal segment with index smaller than snapshot.
		// see comment on wal.ReleaseLockTo for the more details.
		return
	}

	rs.purge(files[:len(files)-1])
}

func (rs *RaftStorage) purgeSnap() {
	snapFiles, err := fileutil.ReadDir(rs.snapDir)
	if err != nil {
		rs.lg.Errorf("Failed to read Snapshot directory %s: %s", rs.snapDir, err)
	}

	var files []string
	for _, f := range snapFiles {
		if !strings.HasSuffix(f, ".snap") {
			continue
		}

		files = append(files, filepath.Join(rs.snapDir, f))
	}

	l := len(files)
	if l <= MaxSnapshotFiles {
		return
	}

	rs.purge(files[:l-MaxSnapshotFiles]) // retain last MaxSnapshotFiles snapshot files
}

func (rs *RaftStorage) purge(files []string) {
	for _, file := range files {
		l, err := fileutil.TryLockFile(file, os.O_WRONLY, fileutil.PrivateFileMode)
		if err != nil {
			rs.lg.Debugf("Failed to lock %s, abort purging", file)
			break
		}

		if err = os.Remove(file); err != nil {
			rs.lg.Errorf("Failed to remove %s: %s", file, err)
		} else {
			rs.lg.Debugf("Purged file %s", file)
		}

		if err = l.Close(); err != nil {
			rs.lg.Errorf("Failed to close file lock %s: %s", l.Name(), err)
		}
	}
}

// Close closes storage
func (rs *RaftStorage) Close() error {
	if err := rs.wal.Close(); err != nil {
		return fmt.Errorf("close WAL failed: %w", err)
	}

	return nil
}
