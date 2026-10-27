package gateway

import (
	_ "embed"
	"net/http"
)

//go:embed openapi/white.swagger.json
var openapiSpec []byte

// docsPage renders Swagger UI over /openapi.json so the gateway root is a
// usable API explorer.
const docsPage = `<!doctype html>
<html lang="id">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<meta name="color-scheme" content="light">
<title>Mixist API</title>
<style>:root{color-scheme:light}body{margin:0;background:#fff}</style>
<link rel="stylesheet" href="https://cdnjs.cloudflare.com/ajax/libs/swagger-ui/5.17.14/swagger-ui.min.css">
</head>
<body>
<div id="ui"></div>
<script src="https://cdnjs.cloudflare.com/ajax/libs/swagger-ui/5.17.14/swagger-ui-bundle.min.js"></script>
<script>
SwaggerUIBundle({ url: "/openapi.json", dom_id: "#ui", persistAuthorization: true });
</script>
</body>
</html>`

func mountDocs(mux *http.ServeMux) {
	mux.HandleFunc("GET /openapi.json", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(openapiSpec)
	})
	page := func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(docsPage))
	}
	mux.HandleFunc("GET /{$}", page)
	mux.HandleFunc("GET /docs", page)
}
