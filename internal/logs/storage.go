package logs

import (
	"bufio"
	"encoding/json"
	"io"
	"janvlog/internal/libs/xerrors"
	"os"
	"path"
	"path/filepath"
	"sync"
	"time"
)

type Storage interface {
	io.Closer
	Add(items ...Item)
	Items() []Item
	File() string
}

var _ Storage = (*storage)(nil)

func NewStorage(fname string) (*storage, error) {
	if path.Ext(fname) == "" {
		fname += ".jsonl"
	}

	err := os.MkdirAll(filepath.Dir(fname), 0777)
	if err != nil {
		return nil, xerrors.Wrap(err, "os.MkdirAll")
	}

	file, err := os.Create(fname)
	if err != nil {
		return nil, xerrors.Wrap(err, "os.Create")
	}

	return &storage{
		fname: fname,
		file:  file,
		items: make([]Item, 0),
	}, nil
}

func ItemsFromStorage(fname string) ([]Item, error) {
	file, err := os.OpenFile(fname, os.O_RDWR|os.O_CREATE, 0777)
	if err != nil {
		return nil, xerrors.Wrap(err, "os.OpenFile")
	}

	items := make([]Item, 0)
	scan := bufio.NewScanner(file)

	for scan.Scan() {
		var item Item

		err := json.Unmarshal(scan.Bytes(), &item)
		if err != nil {
			return nil, xerrors.Wrap(err, "json.Unmarshal")
		}

		items = append(items, item)
	}

	return items, nil
}

type storage struct {
	fname string
	file  *os.File
	items []Item
	mu    sync.Mutex
}

func (s *storage) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.items = nil

	return s.file.Close() //nolint:wrapcheck
}

func (s *storage) File() string {
	return s.fname
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
		_ = json.NewEncoder(s.file).Encode(item)
	}
}

func (s *storage) Items() []Item {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.items
}
