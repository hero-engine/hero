package mocktracker

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"
)

const (
	defaultPerPage = 20
	maxPerPage     = 100
)

// pageParams parses per_page (default 20, capped at 100) and page
// (1-based, default 1) from a query.
func pageParams(q url.Values) (perPage, page int) {
	perPage = defaultPerPage
	if v := q.Get("per_page"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			perPage = n
		}
	}
	if perPage > maxPerPage {
		perPage = maxPerPage
	}
	page = 1
	if v := q.Get("page"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			page = n
		}
	}
	return perPage, page
}

// pageSlice returns the [start,end) window of total for the given
// 1-based page, plus whether a next page exists.
func pageSlice(total, perPage, page int) (start, end int, hasNext bool) {
	start = (page - 1) * perPage
	if start > total {
		start = total
	}
	end = start + perPage
	if end > total {
		end = total
	}
	return start, end, end < total
}

// setLinkHeader writes an RFC 5988 Link header with rel="next" (and
// rel="last") in the GitHub/GitLab style, preserving the request's
// other query params. Called only when a next page exists.
func setLinkHeader(w http.ResponseWriter, r *http.Request, page, perPage, total int) {
	build := func(p int) string {
		q := r.URL.Query()
		q.Set("page", strconv.Itoa(p))
		q.Set("per_page", strconv.Itoa(perPage))
		// Absolute URL so the client can follow it verbatim.
		return fmt.Sprintf("%s://%s%s?%s", schemeOf(r), r.Host, r.URL.Path, q.Encode())
	}
	last := (total + perPage - 1) / perPage
	link := fmt.Sprintf("<%s>; rel=\"next\", <%s>; rel=\"last\"", build(page+1), build(last))
	w.Header().Set("Link", link)
}

func schemeOf(r *http.Request) string {
	if r.TLS != nil {
		return "https"
	}
	return "http"
}
