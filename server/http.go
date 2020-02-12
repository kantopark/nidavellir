package server

import (
	"fmt"
	"net/http"
)

func ok(w http.ResponseWriter) {
	w.WriteHeader(http.StatusOK)
	if _, err := fmt.Fprint(w, "Okay"); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
