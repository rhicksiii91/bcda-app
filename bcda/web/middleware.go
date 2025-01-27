package web

import (
	"net/http"

	"github.com/CMSgov/bcda-app/bcda/responseutils"
	"github.com/CMSgov/bcda-app/bcda/servicemux"
	fhircodes "github.com/google/fhir/go/proto/google/fhir/proto/stu3/codes_go_proto"
)

func ValidateBulkRequestHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := r.Header

		acceptHeader := h.Get("Accept")
		preferHeader := h.Get("Prefer")

		if acceptHeader == "" {
			oo := responseutils.CreateOpOutcome(fhircodes.IssueSeverityCode_ERROR, fhircodes.IssueTypeCode_STRUCTURE, responseutils.FormatErr, "Accept header is required")
			responseutils.WriteError(oo, w, http.StatusBadRequest)
			return
		} else if acceptHeader != "application/fhir+json" {
			oo := responseutils.CreateOpOutcome(fhircodes.IssueSeverityCode_ERROR, fhircodes.IssueTypeCode_STRUCTURE, responseutils.FormatErr, "application/fhir+json is the only supported response format")
			responseutils.WriteError(oo, w, http.StatusBadRequest)
			return
		}

		if preferHeader == "" {
			oo := responseutils.CreateOpOutcome(fhircodes.IssueSeverityCode_ERROR, fhircodes.IssueTypeCode_STRUCTURE, responseutils.FormatErr, "Prefer header is required")
			responseutils.WriteError(oo, w, http.StatusBadRequest)
			return
		} else if preferHeader != "respond-async" {
			oo := responseutils.CreateOpOutcome(fhircodes.IssueSeverityCode_ERROR, fhircodes.IssueTypeCode_STRUCTURE, responseutils.FormatErr, "Only asynchronous responses are supported")
			responseutils.WriteError(oo, w, http.StatusBadRequest)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func ConnectionClose(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Connection", "close")
		next.ServeHTTP(w, r)
	})
}

func SecurityHeader(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if servicemux.IsHTTPS(r) {
			w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains; preload")
			w.Header().Set("Cache-Control", "no-cache; no-store; must-revalidate; max-age=0")
			w.Header().Set("Pragma", "no-cache")
			w.Header().Set("X-Content-Type-Options", "nosniff")
		}
		next.ServeHTTP(w, r)
	})
}
