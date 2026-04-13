package handler

import "net/http"

// Export unexported functions for testing.
// This file is only compiled during `go test`.

var ExportRespondJSON = respondJSON
var ExportRespondError = respondError
var ExportRespondValidationError = respondValidationError

func ExportNewPaginatedResponse(data interface{}, total, page, limit int) interface{} {
	return newPaginatedResponse(data, total, page, limit)
}

func ExportHandleServiceError(w http.ResponseWriter, err error) {
	handleServiceError(w, err)
}
