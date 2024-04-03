package web

import (
	"errors"
	"fmt"
	"goyav/internal/core/domain"
	"goyav/internal/core/port"
	"log"
	"log/slog"
	"net/http"
)

func (d *DocumentMux) root(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/ping", http.StatusPermanentRedirect)
}

func (d *DocumentMux) getDocumentByIDHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w, r)
		return
	}
	om := &ObjectMessage{}
	id := r.PathValue("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "please provide a document ID", om)
		return
	}
	doc, err := d.service.GetDocument(r.Context(), id)
	if err != nil {
		switch {
		case errors.Is(err, port.ErrServiceGetDocumentFailed):
			om.ID = id
			writeError(w, http.StatusNotFound, "document not found", om)
			return
		case errors.Is(err, port.ErrServiceInvalidID):
			om.ID = id
			writeError(w, http.StatusBadRequest, "the provided ID is invalid", om)
			return
		default:
			log.Printf("getDocumentHandler: %v", err.Error())
			writeError(w, http.StatusInternalServerError, "an error occured", om)
			return
		}
	}
	om.Message = "document found"
	om.Document = domain.NewDocumentDTO(doc)
	writeJson(w, http.StatusOK, om)
}

func (d *DocumentMux) postDocumentHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w, r)
		return
	}

	var (
		om               = &ObjectMessage{}
		reqSizeLim int64 = int64(d.maxUploadSize) + (1 << 10)
	)

	r.Body = http.MaxBytesReader(w, r.Body, reqSizeLim)
	defer r.Body.Close()
	if err := r.ParseMultipartForm(reqSizeLim); err != nil {
		slog.Debug(fmt.Sprintf("handler.postDocumentHandler: %v", om.Message), "error", err.Error())
		writeError(w, http.StatusRequestEntityTooLarge, fmt.Sprintf("uploaded data exceeds the maximum allowed size : %v Bytes.", d.maxUploadSize), om)
		return
	}

	tag := r.FormValue("tag")

	file, header, err := r.FormFile("file")
	if err != nil {
		slog.Error("handler.postDocumentHandler: "+om.Message, "msg", err.Error())
		writeError(w, http.StatusBadRequest, "failed to upload file", om)
		return
	}
	defer file.Close()

	if header.Size == 0 {
		writeError(w, http.StatusBadRequest, "the file to upload is empty", om)
		return
	}

	if tag == "" {
		tag = header.Filename
	}
	ID, err := d.service.Upload(r.Context(), file, header.Size, tag)
	switch {
	case err == nil:
		om.ID = ID
		om.Message = "document uploaded successfully."
		writeJson(w, http.StatusCreated, om)
		return
	case errors.Is(err, port.ErrDocumentAlreadyExists):
		om.ID = ID
		om.Message = "document already exists."
		writeJson(w, http.StatusOK, om)
		return
	default:
		writeError(w, http.StatusInternalServerError, "an error occured while uploading", om)
		slog.Error("handler.postDocumentHandler: "+om.Message, "msg", err.Error())
		return
	}
}

func (d *DocumentMux) ping(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w, r)
		return
	}
	om := &ObjectMessage{
		Information: d.service.Information(),
		Version:     d.service.Version(),
	}
	if err := d.service.Ping(); err != nil {
		writeError(w, http.StatusServiceUnavailable, "service unavailable", om)
		return
	}
	om.Message = "PONG : everything is good"
	writeJson(w, http.StatusOK, om)
}
