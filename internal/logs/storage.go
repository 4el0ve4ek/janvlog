package logs

import (
	"bufio"
	"encoding/json"
	"io"
	"os"
	"path"
	"path/filepath"
	"sync"
	"time"
)

type Storage interface {
	io.Closer
	Add(...Item)
	Items() []Item
	File() string
	Clear()
}

var _ Storage = (*storage)(nil)

func NewStorage(fname string) (*storage, error) {
	if path.Ext(fname) == "" {
		fname += ".jsonl"
	}

	err := os.MkdirAll(filepath.Dir(fname), 0777)
	if err != nil {
		return nil, err
	}

	f, err := os.OpenFile(fname, os.O_RDWR|os.O_CREATE, 0777)
	if err != nil {
		return nil, err
	}

	s := &storage{
		fname: fname,
		file:  f,
		items: make([]Item, 0),
	}

	scan := bufio.NewScanner(f)
	for scan.Scan() {
		var item Item

		err := json.Unmarshal(scan.Bytes(), &item)
		if err != nil {
			return nil, err
		}

		s.items = append(s.items, item)
	}

	return s, nil
}

type storage struct {
	fname  string
	file   *os.File
	items  []Item
	mu     sync.Mutex
	closed bool
}

func (s *storage) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.items = nil
	return s.file.Close()
}

func (s *storage) File() string {
	return s.fname
}

func (s *storage) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.items = nil
	s.file.Truncate(0)
	s.file.Seek(0, 0)
}

func (s *storage) Add(items ...Item) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i := range items {
		if items[i].Time.IsZero() {
			items[i].Time = time.Now()
		}
	}

	s.items = append(s.items, items...)
	for _, item := range items {
		json.NewEncoder(s.file).Encode(item)
	}
}

func (s *storage) Items() []Item {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.items
}
