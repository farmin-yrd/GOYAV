package web

func (d *DocumentMux) setup() {

	// root
	d.HandleFunc("GET /{$}", d.root)

	// /documents
	d.HandleFunc("GET /documents", methodNotAllowed)
	d.HandleFunc("POST /documents", d.postDocumentHandler)
	d.HandleFunc("GET /documents/{id}", d.getDocumentByIDHandler)

	// /ping
	d.HandleFunc("GET /ping/", d.ping)
}
