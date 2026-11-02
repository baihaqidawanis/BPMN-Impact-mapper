package api

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/baihaqidawan/bpmn-impact-mapper/internal/bpmnlib"
)

func (h Handler) library(w http.ResponseWriter, r *http.Request) {
	if h.Library == nil {
		writeAPIError(w, http.StatusServiceUnavailable, "LIBRARY_UNAVAILABLE", "BPMN library is not configured.")
		return
	}
	const base = "/api/v1/bpmn-library"
	if strings.HasPrefix(r.URL.Path, base+"/") && strings.HasSuffix(r.URL.Path, "/links") {
		h.libraryLinks(w, r, strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, base+"/"), "/links"))
		return
	}
	if r.URL.Path == base && r.Method == http.MethodGet {
		entries, err := h.Library.List()
		if err != nil {
			h.internalError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"entries": entries})
		return
	}
	if r.URL.Path == base && r.Method == http.MethodPost {
		r.Body = http.MaxBytesReader(w, r.Body, 12<<20)
		defer r.Body.Close()
		var input struct {
			bpmnlib.Metadata
			XML string `json:"xml"`
		}
		decoder := json.NewDecoder(r.Body)
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&input); err != nil {
			writeAPIError(w, 400, "INVALID_DIAGRAM", "Invalid or oversized library upload.")
			return
		}
		if err := decoder.Decode(&struct{}{}); err != io.EOF {
			writeAPIError(w, 400, "INVALID_DIAGRAM", "Upload must contain one JSON object.")
			return
		}
		entry, err := bpmnlib.Prepare(input.Metadata, input.XML)
		if err != nil {
			writeAPIError(w, 400, "INVALID_DIAGRAM", err.Error())
			return
		}
		if err = h.Library.Put(entry); err != nil {
			h.internalError(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, entry)
		return
	}
	if strings.HasPrefix(r.URL.Path, base+"/") && r.Method == http.MethodGet {
		entry, err := h.Library.Get(strings.TrimPrefix(r.URL.Path, base+"/"))
		if errors.Is(err, bpmnlib.ErrNotFound) {
			writeAPIError(w, 404, "NOT_FOUND", "Diagram not found.")
			return
		}
		if err != nil {
			h.internalError(w, err)
			return
		}
		entry.Activities, err = bpmnlib.Activities(entry)
		if err != nil {
			h.internalError(w, err)
			return
		}
		writeJSON(w, 200, entry)
		return
	}
	http.NotFound(w, r)
}

func (h Handler) libraryLinks(w http.ResponseWriter, r *http.Request, parentID string) {
	store, ok := h.Library.(bpmnlib.LinkStore)
	if !ok {
		writeAPIError(w, 503, "LIBRARY_UNAVAILABLE", "Hierarchy storage is not configured.")
		return
	}
	parent, err := h.Library.Get(parentID)
	if errors.Is(err, bpmnlib.ErrNotFound) {
		writeAPIError(w, 404, "NOT_FOUND", "Parent diagram not found.")
		return
	}
	if err != nil {
		h.internalError(w, err)
		return
	}
	if r.Method == http.MethodGet {
		links, err := store.Links(parentID)
		if err != nil {
			h.internalError(w, err)
			return
		}
		writeJSON(w, 200, map[string]any{"links": links})
		return
	}
	if r.Method != http.MethodPut {
		http.NotFound(w, r)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 4096)
	defer r.Body.Close()
	var input struct {
		ElementID string `json:"element_id"`
		TargetID  string `json:"target_id"`
	}
	d := json.NewDecoder(r.Body)
	d.DisallowUnknownFields()
	if err = d.Decode(&input); err != nil {
		writeAPIError(w, 400, "INVALID_LINK", "Invalid activity link.")
		return
	}
	if err = d.Decode(&struct{}{}); err != io.EOF {
		writeAPIError(w, 400, "INVALID_LINK", "Expected one JSON object.")
		return
	}
	var target *bpmnlib.Entry
	if input.TargetID != "" {
		entry, e := h.Library.Get(input.TargetID)
		if errors.Is(e, bpmnlib.ErrNotFound) {
			writeAPIError(w, 404, "NOT_FOUND", "Detail diagram not found.")
			return
		}
		if e != nil {
			h.internalError(w, e)
			return
		}
		target = &entry
	}
	if err = bpmnlib.ValidateLink(parent, input.ElementID, target); err != nil {
		writeAPIError(w, 400, "INVALID_LINK", err.Error())
		return
	}
	if target == nil {
		err = store.DeleteLink(parentID, input.ElementID)
	} else {
		err = store.SaveLink(bpmnlib.Link{ParentID: parentID, ElementID: input.ElementID, TargetID: input.TargetID})
	}
	if err != nil {
		h.internalError(w, err)
		return
	}
	links, err := store.Links(parentID)
	if err != nil {
		h.internalError(w, err)
		return
	}
	writeJSON(w, 200, map[string]any{"links": links})
}
