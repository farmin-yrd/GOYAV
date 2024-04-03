package web

import (
	"goyav/internal/core/port"
	"net/http"
)

// Default upload size limit in bytes : 1 Mib
const DefaultMaxUploadSize int64 = 1 << 20

// DocumentMux extends http.ServeMux with a document management service and an upload size limit.
type DocumentMux struct {
	*http.ServeMux
	service       port.DocumentService
	maxUploadSize uint64 // Maximum upload size for documents, in bytes.
}

func NewDocumentMux(s port.DocumentService, n uint64) *DocumentMux {
	d := &DocumentMux{
		ServeMux:      http.NewServeMux(),
		maxUploadSize: n,
		service:       s,
	}
	d.setup()
	return d
}
