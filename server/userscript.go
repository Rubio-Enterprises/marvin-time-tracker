package main

import (
	"log"
	"net/http"
	"strings"

	"github.com/strubio/marvin-time-tracker/userscript"
)

func userscriptHandler(externalURL string) http.HandlerFunc {
	content := userscript.ScriptContent
	if externalURL != "" {
		content = strings.ReplaceAll(content, "__RELAY_URL__", strings.TrimRight(externalURL, "/"))
		log.Printf("userscript: serving with updateURL rewriting (EXTERNAL_URL=%s)", externalURL)
	} else {
		log.Printf("userscript: serving without updateURL rewriting (EXTERNAL_URL not set)")
	}

	return func(w http.ResponseWriter, r *http.Request) {
		log.Printf("userscript: fetch from %s", r.RemoteAddr)
		w.Header().Set("Content-Type", "text/javascript")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Write([]byte(content))
	}
}
