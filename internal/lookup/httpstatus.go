package lookup

import (
	"strconv"
	"strings"
)

type StatusInfo struct {
	Code        int
	Name        string
	Description string
	Example     string
}

var httpStatuses = map[int]StatusInfo{
	200: {200, "OK", "Request succeeded", "GET /api/users → 200 OK"},
	201: {201, "Created", "Resource created successfully", "POST /api/users → 201 Created + Location header"},
	204: {204, "No Content", "Success with no response body", "DELETE /api/users/1 → 204 No Content"},

	301: {301, "Moved Permanently", "Resource moved to new URL", "GET /old-path → 301 + Location: /new-path"},
	302: {302, "Found", "Temporary redirect", "GET /login → 302 + Location: /dashboard"},
	304: {304, "Not Modified", "Resource unchanged, use cache", "GET /api/data + If-None-Match → 304"},

	400: {400, "Bad Request", "Malformed request syntax", "POST /api/users + invalid JSON → 400"},
	401: {401, "Unauthorized", "Authentication required", "GET /api/private (no token) → 401"},
	403: {403, "Forbidden", "Server refuses to authorize", "DELETE /api/admin (user role) → 403"},
	404: {404, "Not Found", "Resource does not exist", "GET /api/users/999 → 404 Not Found"},
	405: {405, "Method Not Allowed", "HTTP method not supported", "DELETE /api/readonly → 405"},
	408: {408, "Request Timeout", "Server timed out waiting", "POST /upload (slow client) → 408"},
	409: {409, "Conflict", "Request conflicts with current state", "PUT /api/users/1 (version mismatch) → 409"},
	413: {413, "Payload Too Large", "Request entity too large", "POST /upload (>10MB) → 413"},
	415: {415, "Unsupported Media Type", "Media format not supported", "POST /api + text/plain → 415"},
	422: {422, "Unprocessable Entity", "Semantic errors in request", "POST /api/users + {age: -5} → 422"},
	429: {429, "Too Many Requests", "Rate limit exceeded", "100 req/min exceeded → 429 + Retry-After"},

	500: {500, "Internal Server Error", "Unexpected server condition", "Unhandled exception → 500"},
	501: {501, "Not Implemented", "Server lacks functionality", "PATCH not implemented → 501"},
	502: {502, "Bad Gateway", "Invalid response from upstream", "Nginx ↔ dead backend → 502"},
	503: {503, "Service Unavailable", "Server temporarily overloaded", "Maintenance mode → 503 + Retry-After"},
	504: {504, "Gateway Timeout", "Upstream server timeout", "Nginx ↔ slow backend → 504"},
}

var allCodes = []int{
	200, 201, 204,
	301, 302, 304,
	400, 401, 403, 404, 405, 408, 409, 413, 415, 422, 429,
	500, 501, 502, 503, 504,
}

func GetStatus(code int) (StatusInfo, bool) {
	info, ok := httpStatuses[code]
	return info, ok
}

func Search(query string, limit int) []StatusInfo {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil
	}

	if limit <= 0 {
		limit = 10
	}

	var results []StatusInfo

	if isNumeric(query) {
		results = searchNumeric(query, limit)
	} else {
		results = searchText(query, limit)
	}

	return results
}

func isNumeric(s string) bool {
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return len(s) > 0
}

func searchNumeric(query string, limit int) []StatusInfo {
	var results []StatusInfo

	if len(query) == 3 {
		if code, err := strconv.Atoi(query); err == nil {
			if info, ok := httpStatuses[code]; ok {
				return []StatusInfo{info}
			}
		}
		return nil
	}

	for _, code := range allCodes {
		codeStr := strconv.Itoa(code)
		if strings.HasPrefix(codeStr, query) {
			results = append(results, httpStatuses[code])
			if len(results) >= limit {
				break
			}
		}
	}

	return results
}

func searchText(query string, limit int) []StatusInfo {
	query = strings.ToLower(query)
	var results []StatusInfo

	for _, code := range allCodes {
		info := httpStatuses[code]
		name := strings.ToLower(info.Name)
		desc := strings.ToLower(info.Description)

		if strings.Contains(name, query) || strings.Contains(desc, query) {
			results = append(results, info)
			if len(results) >= limit {
				break
			}
		}
	}

	return results
}

func ExtractHTTPCode(text string) int {
	words := strings.Fields(text)
	for i, word := range words {
		if len(word) == 3 && isNumeric(word) {
			if code, err := strconv.Atoi(word); err == nil {
				if _, ok := httpStatuses[code]; ok {
					return code
				}
			}
		}
		lower := strings.ToLower(word)
		if (lower == "http" || lower == "status" || lower == "error" || lower == "code") && i+1 < len(words) {
			next := words[i+1]
			next = strings.TrimRight(next, ".,;:!?")
			if len(next) == 3 && isNumeric(next) {
				if code, err := strconv.Atoi(next); err == nil {
					if _, ok := httpStatuses[code]; ok {
						return code
					}
				}
			}
		}
	}
	return 0
}

func FormatResult(info StatusInfo) string {
	return strconv.Itoa(info.Code) + " " + info.Name + " — " + info.Description
}
