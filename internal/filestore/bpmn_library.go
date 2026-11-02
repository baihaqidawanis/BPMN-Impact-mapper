package filestore

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"

	"github.com/baihaqidawan/bpmn-impact-mapper/internal/bpmnlib"
)

// BPMNLibrary stores complete immutable entries under content-derived names.
type BPMNLibrary struct{ Root string }

func diagramID(id string) bool {
	b, e := hex.DecodeString(id)
	return e == nil && len(b) == 32 && id == fmt.Sprintf("%x", b)
}

func (s BPMNLibrary) Put(entry bpmnlib.Entry) error {
	if !diagramID(entry.ID) {
		return errors.New("invalid diagram ID")
	}
	if err := os.MkdirAll(s.Root, 0o750); err != nil {
		return err
	}
	data, err := json.Marshal(entry)
	if err != nil {
		return err
	}
	temp, err := os.CreateTemp(s.Root, ".diagram-*")
	if err != nil {
		return err
	}
	name := temp.Name()
	defer os.Remove(name)
	if _, err = temp.Write(data); err != nil {
		temp.Close()
		return err
	}
	if err = temp.Close(); err != nil {
		return err
	}
	target := filepath.Join(s.Root, entry.ID+".json")
	// Content IDs are immutable: an existing entry needs no replacement.
	if _, err = os.Lstat(target); err == nil {
		_, err = s.Get(entry.ID)
		return err
	} else if !os.IsNotExist(err) {
		return err
	}
	if err = os.Rename(name, target); err != nil {
		// Another request may have saved the same content ID concurrently.
		if existing, readErr := s.Get(entry.ID); readErr == nil && existing.XML == entry.XML && existing.Metadata == entry.Metadata {
			return nil
		}
		return err
	}
	return nil
}

func (s BPMNLibrary) Get(id string) (bpmnlib.Entry, error) {
	if !diagramID(id) {
		return bpmnlib.Entry{}, bpmnlib.ErrNotFound
	}
	path := filepath.Join(s.Root, id+".json")
	info, err := os.Lstat(path)
	if os.IsNotExist(err) {
		return bpmnlib.Entry{}, bpmnlib.ErrNotFound
	}
	if err != nil {
		return bpmnlib.Entry{}, err
	}
	if !info.Mode().IsRegular() || info.Size() > 64<<20 {
		return bpmnlib.Entry{}, errors.New("invalid library entry")
	}
	file, err := os.Open(path)
	if err != nil {
		return bpmnlib.Entry{}, err
	}
	defer file.Close()
	var entry bpmnlib.Entry
	if err = json.NewDecoder(io.LimitReader(file, 64<<20)).Decode(&entry); err != nil {
		return entry, err
	}
	if entry.ID != id {
		return entry, errors.New("library ID mismatch")
	}
	return entry, nil
}

func (s BPMNLibrary) List() ([]bpmnlib.Entry, error) {
	result := []bpmnlib.Entry{}
	files, err := os.ReadDir(s.Root)
	if os.IsNotExist(err) {
		return result, nil
	}
	if err != nil {
		return nil, err
	}
	for _, file := range files {
		name := file.Name()
		if filepath.Ext(name) != ".json" {
			continue
		}
		id := name[:len(name)-5]
		if !diagramID(id) {
			continue
		}
		entry, err := s.Get(id)
		if err != nil {
			return nil, err
		}
		entry.XML = ""
		result = append(result, entry)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID })
	return result, nil
}
