package handler_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"taskflow/internal/handler"
)

func TestRespondJSON(t *testing.T) {
	rec := httptest.NewRecorder()
	data := map[string]string{"hello": "world"}

	handler.ExportRespondJSON(rec, http.StatusOK, data)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("expected application/json, got %s", ct)
	}

	var resp map[string]string
	json.Unmarshal(rec.Body.Bytes(), &resp)
	if resp["hello"] != "world" {
		t.Errorf("unexpected body: %v", resp)
	}
}

func TestRespondError(t *testing.T) {
	rec := httptest.NewRecorder()
	handler.ExportRespondError(rec, http.StatusNotFound, "not found")

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rec.Code)
	}

	var resp map[string]string
	json.Unmarshal(rec.Body.Bytes(), &resp)
	if resp["error"] != "not found" {
		t.Errorf("expected 'not found', got %s", resp["error"])
	}
}

func TestRespondValidationError(t *testing.T) {
	rec := httptest.NewRecorder()
	fields := map[string]string{"email": "is required", "name": "is required"}

	handler.ExportRespondValidationError(rec, fields)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}

	var resp map[string]interface{}
	json.Unmarshal(rec.Body.Bytes(), &resp)
	if resp["error"] != "validation failed" {
		t.Errorf("expected 'validation failed', got %v", resp["error"])
	}
}

func TestNewPaginatedResponse(t *testing.T) {
	items := []string{"a", "b", "c"}
	result := handler.ExportNewPaginatedResponse(items, 25, 2, 10)

	pr, ok := result.(handler.PaginatedResponse)
	if !ok {
		t.Fatal("expected PaginatedResponse type")
	}
	if pr.Total != 25 {
		t.Errorf("expected total 25, got %d", pr.Total)
	}
	if pr.Page != 2 {
		t.Errorf("expected page 2, got %d", pr.Page)
	}
	if pr.TotalPages != 3 {
		t.Errorf("expected 3 total pages, got %d", pr.TotalPages)
	}
}
