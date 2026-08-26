package httpapi

import (
	"errors"
	"net/http"
	"strconv"
)

// parseID reads an int64 path parameter by name.
func parseID(r *http.Request, name string) (int64, error) {
	v := r.PathValue(name)
	if v == "" {
		return 0, errors.New("missing path parameter " + name)
	}
	id, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		return 0, errors.New("invalid " + name + ": " + v)
	}
	if id <= 0 {
		return 0, errors.New("non-positive " + name)
	}
	return id, nil
}
