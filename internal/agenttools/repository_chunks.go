package agenttools

import (
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/planning"
	"sort"
	"unicode/utf8"
)

// String slices share the immutable snapshot backing memory. The complete
// eligible file is covered, including empty files and UTF-8 boundary overlap.
func (s *Snapshot) RepositoryChunks() []planning.RepositoryChunk {
	names := make([]string, 0, len(s.files))
	for name := range s.files {
		names = append(names, name)
	}
	sort.Strings(names)
	metadata := map[string][2]string{}
	for _, a := range s.artifacts {
		if _, ok := metadata[a.File]; !ok {
			metadata[a.File] = [2]string{a.Module, a.Layer}
		}
	}
	out := []planning.RepositoryChunk{}
	for _, file := range names {
		text := s.files[file]
		start := 0
		for {
			end := min(start+4096, len(text))
			for end < len(text) && end > start && !utf8.RuneStart(text[end]) {
				end--
			}
			c := planning.RepositoryChunk{File: file, Module: metadata[file][0], Layer: metadata[file][1], StartByte: start, EndByte: end, FileBytes: len(text), LastInFile: end == len(text), Content: text[start:end]}
			c.ID = digest([]any{"source-chunk-v1-4096-128", file, start, end, c.Content})[:32]
			out = append(out, c)
			if c.LastInFile {
				break
			}
			start = max(start+1, end-128)
			for start < end && !utf8.RuneStart(text[start]) {
				start++
			}
		}
	}
	return out
}
