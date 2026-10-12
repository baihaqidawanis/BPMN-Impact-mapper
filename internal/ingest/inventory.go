package ingest

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"unicode/utf8"
)

const (
	defaultMaxFileSize  = int64(1 << 20)
	defaultMaxTotalSize = int64(64 << 20)
	defaultMaxFiles     = 10000
)

type Options struct {
	MaxFileSize  int64    `json:"max_file_size,omitempty"`
	MaxTotalSize int64    `json:"max_total_size,omitempty"`
	MaxFiles     int      `json:"max_files,omitempty"`
	Ignore       []string `json:"ignore,omitempty"`
}

type File struct {
	RelativePath string
	AbsolutePath string
	Content      []byte
}

type Inventory struct {
	Root     string
	Files    []File
	Warnings []string
	Bytes    int64
}

var excludedDirectories = map[string]bool{
	".git": true, "node_modules": true, "vendor": true, "dist": true,
	"build": true, "cache": true, ".cache": true, "coverage": true,
}

func Build(ctx context.Context, root string, options Options) (Inventory, error) {
	if err := ctx.Err(); err != nil {
		return Inventory{}, err
	}
	if options.MaxFileSize <= 0 {
		options.MaxFileSize = defaultMaxFileSize
	}
	if options.MaxTotalSize <= 0 {
		options.MaxTotalSize = defaultMaxTotalSize
	}
	if options.MaxFiles <= 0 {
		options.MaxFiles = defaultMaxFiles
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		return Inventory{}, fmt.Errorf("absolute repository path: %w", err)
	}
	info, err := os.Lstat(abs)
	if err != nil {
		return Inventory{}, fmt.Errorf("repository path: %w", err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return Inventory{}, errors.New("repository root must not be a symlink")
	}
	if !info.IsDir() {
		return Inventory{}, errors.New("repository path is not a directory")
	}
	canonical, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return Inventory{}, fmt.Errorf("canonical repository path: %w", err)
	}
	result := Inventory{Root: filepath.Clean(canonical), Files: []File{}, Warnings: []string{}}
	err = filepath.WalkDir(result.Root, func(current string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		rel, err := filepath.Rel(result.Root, current)
		if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			return errors.New("inventory path escaped repository root")
		}
		rel = filepath.ToSlash(rel)
		if rel == "." {
			return nil
		}
		name := strings.ToLower(entry.Name())
		if entry.Type()&os.ModeSymlink != 0 {
			result.Warnings = append(result.Warnings, "skipped symlink: "+rel)
			if entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if entry.IsDir() {
			if excludedDirectories[name] || leakageDirectory(rel) || matchesAny(rel, options.Ignore) {
				return filepath.SkipDir
			}
			return nil
		}
		if secretFile(name) || processModelFile(name) || leakageFile(rel) || generatedFile(name) || matchesAny(rel, options.Ignore) {
			return nil
		}
		fileInfo, err := entry.Info()
		if err != nil {
			return err
		}
		if fileInfo.Size() > options.MaxFileSize {
			result.Warnings = append(result.Warnings, fmt.Sprintf("skipped oversized file: %s", rel))
			return nil
		}
		if len(result.Files) >= options.MaxFiles {
			return fmt.Errorf("repository file limit exceeded: %d", options.MaxFiles)
		}
		if result.Bytes+fileInfo.Size() > options.MaxTotalSize {
			return fmt.Errorf("repository byte limit exceeded: %d", options.MaxTotalSize)
		}
		content, err := os.ReadFile(current)
		if err != nil {
			return err
		}
		if isBinary(content) {
			return nil
		}
		result.Bytes += int64(len(content))
		result.Files = append(result.Files, File{RelativePath: rel, AbsolutePath: current, Content: content})
		return nil
	})
	if err != nil {
		return Inventory{}, err
	}
	sort.Slice(result.Files, func(i, j int) bool { return result.Files[i].RelativePath < result.Files[j].RelativePath })
	sort.Strings(result.Warnings)
	return result, nil
}

func matchesAny(relative string, patterns []string) bool {
	relative = strings.ToLower(filepath.ToSlash(relative))
	for _, pattern := range patterns {
		pattern = strings.ToLower(filepath.ToSlash(strings.TrimSpace(pattern)))
		if pattern == "" {
			continue
		}
		if matched, _ := path.Match(pattern, relative); matched {
			return true
		}
		if strings.HasSuffix(pattern, "/**") && strings.HasPrefix(relative, strings.TrimSuffix(pattern, "**")) {
			return true
		}
	}
	return false
}

func secretFile(name string) bool {
	if name == ".env" || strings.HasPrefix(name, ".env.") {
		return true
	}
	ext := strings.ToLower(filepath.Ext(name))
	if ext == ".pem" || ext == ".key" || ext == ".p12" || ext == ".pfx" || ext == ".jks" {
		return true
	}
	return strings.Contains(name, "credential") || strings.Contains(name, "credentials") ||
		strings.Contains(name, "private_key") || strings.Contains(name, "private-key")
}

func processModelFile(name string) bool {
	return strings.EqualFold(filepath.Ext(name), ".bpmn")
}

func leakageFile(relative string) bool {
	lower := strings.ToLower(filepath.ToSlash(relative))
	base := path.Base(lower)
	if containsPathSegment(lower, "checkpoints") || containsPathSegment(lower, "ground-truth") || containsPathSegment(lower, "ground_truth") {
		return true
	}
	// Documentation plan trees are method-development artifacts, not prediction
	// evidence. Keep similarly named source packages (for example internal/plans)
	// available to the index.
	if strings.HasPrefix(lower, "docs/") && strings.Contains("/"+lower+"/", "/plans/") {
		return true
	}
	blocked := []string{"process-mapping", "traceability-coverage", "release-candidate", "review-findings", "impact-report", "ground-truth", "ground_truth", "implementation-plan", "audit-report"}
	for _, marker := range blocked {
		if strings.Contains(base, marker) {
			return true
		}
	}
	return false
}

func leakageDirectory(relative string) bool {
	lower := strings.ToLower(filepath.ToSlash(relative))
	return containsPathSegment(lower, "checkpoints") || containsPathSegment(lower, "ground-truth") || containsPathSegment(lower, "ground_truth")
}

func containsPathSegment(relative, segment string) bool {
	return strings.Contains("/"+strings.Trim(relative, "/")+"/", "/"+segment+"/")
}

func generatedFile(name string) bool {
	return strings.HasSuffix(name, ".min.js") || strings.HasSuffix(name, ".min.css") ||
		strings.HasSuffix(name, ".map") || strings.HasSuffix(name, ".generated.go") ||
		strings.HasSuffix(name, ".g.cs")
}

func isBinary(content []byte) bool {
	if len(content) == 0 {
		return false
	}
	sample := content
	if len(sample) > 8192 {
		sample = sample[:8192]
	}
	if bytes.IndexByte(sample, 0) >= 0 || !utf8.Valid(sample) {
		return true
	}
	controls := 0
	for _, value := range sample {
		if value < 9 || (value > 13 && value < 32) {
			controls++
		}
	}
	return float64(controls)/float64(len(sample)) > 0.10
}
