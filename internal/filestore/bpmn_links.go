package filestore

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"sync"

	"github.com/baihaqidawan/bpmn-impact-mapper/internal/bpmnlib"
)

var libraryLinkWrites sync.Mutex

func (s BPMNLibrary) Links(parent string) ([]bpmnlib.Link, error) {
	if !diagramID(parent) {
		return nil, bpmnlib.ErrNotFound
	}
	path := filepath.Join(s.Root, "links", parent+".json")
	info, err := os.Lstat(path)
	if os.IsNotExist(err) {
		return []bpmnlib.Link{}, nil
	}
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() || info.Size() > 1<<20 {
		return nil, errors.New("invalid diagram links file")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var links []bpmnlib.Link
	if err = json.Unmarshal(data, &links); err != nil {
		return nil, err
	}
	for _, link := range links {
		if link.ParentID != parent || !diagramID(link.TargetID) {
			return nil, errors.New("invalid persisted link")
		}
	}
	return links, nil
}

func (s BPMNLibrary) SaveLink(link bpmnlib.Link) error { return s.changeLink(link, false) }
func (s BPMNLibrary) DeleteLink(parent, element string) error {
	return s.changeLink(bpmnlib.Link{ParentID: parent, ElementID: element}, true)
}
func (s BPMNLibrary) changeLink(link bpmnlib.Link, remove bool) error {
	if !diagramID(link.ParentID) || (!remove && !diagramID(link.TargetID)) || link.ElementID == "" || len(link.ElementID) > 1024 {
		return errors.New("invalid diagram link")
	}
	libraryLinkWrites.Lock()
	defer libraryLinkWrites.Unlock()
	links, err := s.Links(link.ParentID)
	if err != nil {
		return err
	}
	updated := []bpmnlib.Link{}
	for _, existing := range links {
		if existing.ElementID != link.ElementID {
			updated = append(updated, existing)
		}
	}
	if !remove {
		updated = append(updated, link)
	}
	sort.Slice(updated, func(i, j int) bool { return updated[i].ElementID < updated[j].ElementID })
	root := filepath.Join(s.Root, "links")
	if err = os.MkdirAll(root, 0750); err != nil {
		return err
	}
	data, err := json.Marshal(updated)
	if err != nil {
		return err
	}
	if len(data) > 1<<20 {
		return errors.New("too many diagram links")
	}
	f, err := os.CreateTemp(root, ".links-*")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	if _, err = f.Write(data); err != nil {
		f.Close()
		return err
	}
	if err = f.Close(); err != nil {
		return err
	}
	return os.Rename(f.Name(), filepath.Join(root, link.ParentID+".json"))
}
