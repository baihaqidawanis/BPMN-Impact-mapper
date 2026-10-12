// Package api is the HTTP transport adapter for the analysis application.
package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/baihaqidawan/bpmn-impact-mapper/internal/app"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/domain"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/mapping"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/report"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/reportstore"
)

const (
	defaultRequestBytes = int64(20 << 20)
	defaultBPMNBytes    = int64(8 << 20)
	maxFieldBytes       = int64(16 << 10)
)

type Handler struct {
	Analyzer       app.Analyzer
	Store          reportstore.Store
	TempRoot       string
	MaxRequestSize int64
	MaxBPMNSize    int64
	LogError       func(error)
}

func (handler Handler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	writer.Header().Set("X-Content-Type-Options", "nosniff")
	switch {
	case request.Method == http.MethodPost && request.URL.Path == "/api/v1/review-packs":
		handler.prepareReviewPack(writer, request)
	case request.Method == http.MethodPost && request.URL.Path == "/api/v1/reviews/reconcile":
		handler.reconcileReviews(writer, request)
	case request.Method == http.MethodPost && request.URL.Path == "/api/v1/evaluations":
		handler.evaluateReports(writer, request)
	case request.Method == http.MethodGet && request.URL.Path == "/api/v1/health":
		writeJSON(writer, http.StatusOK, map[string]string{"status": "ok"})
	case request.Method == http.MethodGet && request.URL.Path == "/api/v1/analyses":
		handler.listAnalyses(writer, request)
	case request.Method == http.MethodPost && request.URL.Path == "/api/v1/analyses":
		handler.createAnalysis(writer, request)
	case request.Method == http.MethodGet && strings.HasPrefix(request.URL.Path, "/api/v1/analyses/"):
		handler.getAnalysis(writer, request)
	case request.Method == http.MethodPost && request.URL.Path == "/analyze":
		handler.legacyAnalyze(writer, request)
	default:
		http.NotFound(writer, request)
	}
}

func (handler Handler) listAnalyses(writer http.ResponseWriter, request *http.Request) {
	if handler.Store == nil {
		handler.internalError(writer, errors.New("report store is not configured"))
		return
	}
	values, err := handler.Store.List(request.Context())
	if err != nil {
		handler.internalError(writer, err)
		return
	}
	writeJSON(writer, http.StatusOK, struct {
		Analyses []domain.ReportSummary `json:"analyses"`
	}{values})
}

func (handler Handler) createAnalysis(writer http.ResponseWriter, request *http.Request) {
	if handler.Store == nil {
		handler.internalError(writer, errors.New("report store is not configured"))
		return
	}
	maxRequest := handler.MaxRequestSize
	if maxRequest <= 0 {
		maxRequest = defaultRequestBytes
	}
	request.Body = http.MaxBytesReader(writer, request.Body, maxRequest)
	defer request.Body.Close()

	temporaryRoot := handler.TempRoot
	if temporaryRoot == "" {
		temporaryRoot = os.TempDir()
	}
	if err := os.MkdirAll(temporaryRoot, 0o750); err != nil {
		handler.internalError(writer, fmt.Errorf("create upload root: %w", err))
		return
	}
	temporaryDirectory, err := os.MkdirTemp(temporaryRoot, "analysis-*")
	if err != nil {
		handler.internalError(writer, fmt.Errorf("create upload workspace: %w", err))
		return
	}
	defer os.RemoveAll(temporaryDirectory)

	fields, files, err := handler.readMultipart(request, temporaryDirectory)
	if err != nil {
		writeAPIError(writer, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
		return
	}
	input, err := webRequest(fields, files)
	if err != nil {
		writeAPIError(writer, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
		return
	}
	value, err := handler.Analyzer.Analyze(request.Context(), input)
	if err != nil {
		handler.log(err)
		writeAPIError(writer, http.StatusUnprocessableEntity, "ANALYSIS_FAILED", "Analysis could not be completed. Verify the repository path, BPMN files, and selected method.")
		return
	}
	if err := handler.Store.Save(request.Context(), value); err != nil {
		handler.internalError(writer, err)
		return
	}
	writer.Header().Set("Location", "/api/v1/analyses/"+value.AnalysisID)
	data, err := report.JSON(value)
	if err != nil {
		handler.internalError(writer, err)
		return
	}
	writer.Header().Set("Content-Type", "application/json; charset=utf-8")
	writer.WriteHeader(http.StatusCreated)
	_, _ = writer.Write(data)
}

func (handler Handler) readMultipart(request *http.Request, temporaryDirectory string) (map[string]string, map[string]string, error) {
	reader, err := request.MultipartReader()
	if err != nil {
		return nil, nil, errors.New("content type must be multipart/form-data")
	}
	fields := map[string]string{}
	files := map[string]string{}
	partCount := 0
	for {
		part, err := reader.NextPart()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, nil, errors.New("invalid multipart body")
		}
		partCount++
		if partCount > 24 {
			_ = part.Close()
			return nil, nil, errors.New("too many multipart fields")
		}
		name := part.FormName()
		if name == "" {
			_ = part.Close()
			return nil, nil, errors.New("multipart field name is required")
		}
		if part.FileName() != "" {
			if name != "old_bpmn" && name != "new_bpmn" {
				_ = part.Close()
				return nil, nil, fmt.Errorf("unsupported file field %q", name)
			}
			if files[name] != "" {
				_ = part.Close()
				return nil, nil, fmt.Errorf("duplicate file field %q", name)
			}
			path := filepath.Join(temporaryDirectory, name+".bpmn")
			if err := handler.copyBPMN(path, part); err != nil {
				_ = part.Close()
				return nil, nil, err
			}
			files[name] = path
		} else {
			if !validTextField(name) {
				_ = part.Close()
				return nil, nil, fmt.Errorf("unsupported field %q", name)
			}
			if _, exists := fields[name]; exists {
				_ = part.Close()
				return nil, nil, fmt.Errorf("duplicate field %q", name)
			}
			data, err := io.ReadAll(io.LimitReader(part, maxFieldBytes+1))
			if err != nil {
				_ = part.Close()
				return nil, nil, fmt.Errorf("read field %q", name)
			}
			if int64(len(data)) > maxFieldBytes {
				_ = part.Close()
				return nil, nil, fmt.Errorf("field %q is too large", name)
			}
			fields[name] = strings.TrimSpace(string(data))
		}
		if err := part.Close(); err != nil {
			return nil, nil, errors.New("close multipart field")
		}
	}
	return fields, files, nil
}

func (handler Handler) copyBPMN(path string, source *multipart.Part) error {
	limit := handler.MaxBPMNSize
	if limit <= 0 {
		limit = defaultBPMNBytes
	}
	destination, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return errors.New("create temporary BPMN file")
	}
	written, copyErr := io.Copy(destination, io.LimitReader(source, limit+1))
	closeErr := destination.Close()
	if copyErr != nil || closeErr != nil {
		return errors.New("store uploaded BPMN file")
	}
	if written > limit {
		return fmt.Errorf("BPMN file exceeds %d bytes", limit)
	}
	return nil
}

func webRequest(fields, files map[string]string) (app.Request, error) {
	repositoryPath := fields["repo_path"]
	if repositoryPath == "" {
		return app.Request{}, errors.New("repo_path is required")
	}
	if files["old_bpmn"] == "" || files["new_bpmn"] == "" {
		return app.Request{}, errors.New("old_bpmn and new_bpmn files are required")
	}
	method := domain.AnalysisMethod(fields["method"])
	if method == "" {
		method = domain.MethodHybrid
	}
	if !method.Valid() {
		return app.Request{}, fmt.Errorf("unsupported method %q", method)
	}
	if method == domain.MethodConfiguration && fields["mapping_config_path"] == "" {
		return app.Request{}, errors.New("mapping_config_path is required for configuration method")
	}
	topK := 10
	if fields["top_k"] != "" {
		value, err := strconv.Atoi(fields["top_k"])
		if err != nil || value < 1 || value > 100 {
			return app.Request{}, errors.New("top_k must be between 1 and 100")
		}
		topK = value
	}
	return app.Request{
		RepositoryPath:    repositoryPath,
		OldBPMNPath:       files["old_bpmn"],
		NewBPMNPath:       files["new_bpmn"],
		RepositoryName:    fields["repository_name"],
		Revision:          fields["revision"],
		Method:            method,
		MappingConfigPath: fields["mapping_config_path"],
		Retrieval:         mapping.Options{TopK: topK},
	}, nil
}

func validTextField(name string) bool {
	switch name {
	case "repo_path", "repository_name", "revision", "method", "mapping_config_path", "top_k":
		return true
	default:
		return false
	}
}

func (handler Handler) getAnalysis(writer http.ResponseWriter, request *http.Request) {
	if handler.Store == nil {
		handler.internalError(writer, errors.New("report store is not configured"))
		return
	}
	remainder := strings.TrimPrefix(request.URL.Path, "/api/v1/analyses/")
	parts := strings.Split(remainder, "/")
	if len(parts) > 2 || parts[0] == "" {
		http.NotFound(writer, request)
		return
	}
	value, err := handler.Store.Get(request.Context(), parts[0])
	if errors.Is(err, reportstore.ErrNotFound) {
		writeAPIError(writer, http.StatusNotFound, "NOT_FOUND", "Analysis report was not found.")
		return
	}
	if err != nil {
		handler.internalError(writer, err)
		return
	}
	format := "report.json"
	if len(parts) == 2 {
		format = parts[1]
	}
	switch format {
	case "report.json":
		data, err := report.JSON(value)
		if err != nil {
			handler.internalError(writer, err)
			return
		}
		writer.Header().Set("Content-Type", "application/json; charset=utf-8")
		writer.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="impact-report-%s.json"`, value.AnalysisID))
		_, _ = writer.Write(data)
	case "report.md":
		writer.Header().Set("Content-Type", "text/markdown; charset=utf-8")
		writer.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="impact-report-%s.md"`, value.AnalysisID))
		_, _ = writer.Write(report.Markdown(value))
	default:
		http.NotFound(writer, request)
	}
}

func (handler Handler) legacyAnalyze(writer http.ResponseWriter, request *http.Request) {
	request.Body = http.MaxBytesReader(writer, request.Body, 64<<10)
	defer request.Body.Close()
	var input app.Request
	decoder := json.NewDecoder(request.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&input); err != nil {
		http.Error(writer, "invalid request", http.StatusBadRequest)
		return
	}
	value, err := handler.Analyzer.Analyze(request.Context(), input)
	if err != nil {
		http.Error(writer, err.Error(), http.StatusUnprocessableEntity)
		return
	}
	data, err := report.JSON(value)
	if err != nil {
		http.Error(writer, "encode report", http.StatusInternalServerError)
		return
	}
	writer.Header().Set("Content-Type", "application/json; charset=utf-8")
	writer.WriteHeader(http.StatusOK)
	_, _ = writer.Write(data)
}

func (handler Handler) internalError(writer http.ResponseWriter, err error) {
	handler.log(err)
	writeAPIError(writer, http.StatusInternalServerError, "INTERNAL_ERROR", "The server could not complete the request.")
}

func (handler Handler) log(err error) {
	if handler.LogError != nil {
		handler.LogError(err)
	}
}

func writeAPIError(writer http.ResponseWriter, status int, code, message string) {
	writeJSON(writer, status, map[string]any{"error": map[string]string{"code": code, "message": message}})
}

func writeJSON(writer http.ResponseWriter, status int, value any) {
	writer.Header().Set("Content-Type", "application/json; charset=utf-8")
	writer.WriteHeader(status)
	_ = json.NewEncoder(writer).Encode(value)
}
