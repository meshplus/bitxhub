package leveldb

import (
	"fmt"
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIter_Next(t *testing.T) {
	dir, err := ioutil.TempDir("", "TestNext")
	require.Nil(t, err)

	_, err = New(dir)
	require.Nil(t, err)
	_, err = New(dir)
	require.EqualValues(t, fmt.Sprintf("open %s failed: resource temporarily unavailable", dir), err.Error())
}

func TestLdb_Put(t *testing.T) {
	dir, err := ioutil.TempDir("", "TestPut")
	require.Nil(t, err)

	s, err := New(dir)
	require.Nil(t, err)

	s.Put([]byte("key"), []byte("value"))
	err = s.Close()
	require.Nil(t, err)
}

func TestLdb_Delete(t *testing.T) {
	dir, err := ioutil.TempDir("", "TestDelete")
	require.Nil(t, err)

	s, err := New(dir)
	require.Nil(t, err)

	s.Put([]byte("key"), []byte("value"))
	s.Delete([]byte("key"))
}

func TestLdb_Get(t *testing.T) {
	dir, err := ioutil.TempDir("", "TestGet")
	require.Nil(t, err)

	s, err := New(dir)
	require.Nil(t, err)

	s.Put([]byte("key"), []byte("value"))
	v1 := s.Get([]byte("key"))
	assert.Equal(t, v1, []byte("value"))
	s.Delete([]byte("key"))
	v2 := s.Get([]byte("key"))
	assert.True(t, v2 == nil)
}

func TestLdb_GetPanic(t *testing.T) {
	defer func() {
		if err := recover(); err != nil {
			assert.NotNil(t, err)
		}
	}()

	dir, err := ioutil.TempDir("", "TestGetPanic")
	require.Nil(t, err)

	s, err := New(dir)
	require.Nil(t, err)

	err = s.Close()
	require.Nil(t, err)

	s.Get([]byte("key"))
	assert.True(t, false)
}

func TestLdb_PutPanic(t *testing.T) {
	defer func() {
		if err := recover(); err != nil {
			assert.NotNil(t, err)
		}
	}()

	dir, err := ioutil.TempDir("", "TestPutPanic")
	require.Nil(t, err)

	s, err := New(dir)
	require.Nil(t, err)

	err = s.Close()
	require.Nil(t, err)

	s.Put([]byte("key"), []byte("key"))
	assert.True(t, false)
}

func TestLdb_DeletePanic(t *testing.T) {
	defer func() {
		if err := recover(); err != nil {
			assert.NotNil(t, err)
		}
	}()

	dir, err := ioutil.TempDir("", "TestDeletePanic")
	require.Nil(t, err)

	s, err := New(dir)
	require.Nil(t, err)

	err = s.Close()
	require.Nil(t, err)

	s.Delete([]byte("key"))
	assert.True(t, false)
}

func TestLdb_Has(t *testing.T) {
	dir, err := ioutil.TempDir("", "TestHas")
	require.Nil(t, err)

	s, err := New(dir)
	require.Nil(t, err)

	key := []byte("key")
	r1 := s.Has(key)
	assert.True(t, !r1)
	s.Put(key, []byte("value"))
	r2 := s.Has(key)
	assert.True(t, r2)
	s.Delete(key)
	r3 := s.Has(key)
	assert.True(t, !r3)
}

func TestLdb_NewBatch(t *testing.T) {
	dir, err := ioutil.TempDir("", "TestNewBatch")
	require.Nil(t, err)

	s, err := New(dir)
	require.Nil(t, err)

	batch := s.NewBatch()
	for i := 0; i < 11; i++ {
		key := fmt.Sprintf("key%d", i)
		batch.Put([]byte(key), []byte(key))
	}
	batch.Delete([]byte("key10"))
	batch.Commit()

	for i := 0; i < 10; i++ {
		key := fmt.Sprintf("key%d", i)
		value := s.Get([]byte(key))
		assert.EqualValues(t, key, value)
	}
}

func TestLdb_CommitPanic(t *testing.T) {
	defer func() {
		if err := recover(); err != nil {
			assert.NotNil(t, err)
		}
	}()

	dir, err := ioutil.TempDir("", "TestDeletePanic")
	require.Nil(t, err)

	s, err := New(dir)
	require.Nil(t, err)

	batch := s.NewBatch()
	for i := 0; i < 11; i++ {
		key := fmt.Sprintf("key%d", i)
		batch.Put([]byte(key), []byte(key))
	}
	err = s.Close()
	require.Nil(t, err)

	batch.Commit()
	assert.True(t, false)
}

func TestLdb_Iterator(t *testing.T) {
	dir, err := ioutil.TempDir("", "TestIterator")
	require.Nil(t, err)

	s, err := New(dir)
	require.Nil(t, err)

	batch := s.NewBatch()
	for i := 0; i < 10; i++ {
		key := fmt.Sprintf("key%d", i)
		batch.Put([]byte(key), []byte(key))
	}
	batch.Commit()

	iter := s.Iterator([]byte("key0"), []byte("key9"))
	i := 0
	for iter.Next() {
		assert.EqualValues(t, []byte(fmt.Sprintf("key%d", i)), iter.Value())
		assert.EqualValues(t, []byte(fmt.Sprintf("key%d", i)), iter.Key())
		i++
	}
}

func TestLdb_Prefix(t *testing.T) {
	dir, err := ioutil.TempDir("", "TestPrefix")
	require.Nil(t, err)

	s, err := New(dir)
	require.Nil(t, err)

	batch := s.NewBatch()
	for i := 0; i < 10; i++ {
		key := fmt.Sprintf("key%d", i)
		batch.Put([]byte(key), []byte(key))
	}
	batch.Commit()

	iter := s.Prefix([]byte("key"))

	i := 0
	for iter.Next() {
		assert.EqualValues(t, []byte(fmt.Sprintf("key%d", i)), iter.Value())
		assert.EqualValues(t, []byte(fmt.Sprintf("key%d", i)), iter.Key())
		i++
	}
}

func TestLdb_Seek(t *testing.T) {
	dir, err := ioutil.TempDir("", "TestSeek")
	require.Nil(t, err)

	s, err := New(dir)
	require.Nil(t, err)

	batch := s.NewBatch()
	for i := 0; i < 10; i++ {
		key := fmt.Sprintf("key%d", i)
		batch.Put([]byte(key), []byte(key))
	}
	batch.Commit()

	iter := s.Iterator([]byte("key0"), []byte("key9"))
	iter.Seek([]byte("key5"))
	assert.EqualValues(t, []byte("key5"), iter.Key())
	i := 6
	for iter.Next() {
		assert.EqualValues(t, []byte(fmt.Sprintf("key%d", i)), iter.Value())
		assert.EqualValues(t, []byte(fmt.Sprintf("key%d", i)), iter.Key())
		i++
	}
}
func TestLdb_Prev(t *testing.T) {
	dir, err := ioutil.TempDir("", "TestPrev")
	require.Nil(t, err)

	s, err := New(dir)
	require.Nil(t, err)

	batch := s.NewBatch()
	for i := 0; i < 10; i++ {
		key := fmt.Sprintf("key%d", i)
		batch.Put([]byte(key), []byte(key))
	}
	batch.Commit()

	iter := s.Iterator([]byte("key0"), []byte("key9"))
	iter.Seek([]byte("key8"))
	assert.EqualValues(t, []byte("key8"), iter.Key())
	i := 7
	for iter.Prev() {
		assert.EqualValues(t, []byte(fmt.Sprintf("key%d", i)), iter.Value())
		assert.EqualValues(t, []byte(fmt.Sprintf("key%d", i)), iter.Key())
		i--
	}
}
