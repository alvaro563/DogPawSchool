package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
)

// bodyLimitMaxBytes is the upper bound for any mutating request
// body. Set high enough to accommodate the largest legitimate
// payload in the API (a dog edit with all notes + photo URL fits
// well under 32 KB) and low enough to make OOM-by-payload attacks
// impractical.
const bodyLimitMaxBytes int64 = 1 << 20 // 1 MiB

// BodyLimitMiddleware caps the request body for mutating HTTP
// methods (POST, PATCH, PUT, DELETE). GETs are not capped here
// because they carry no body in the API surface — large GET
// responses are a different problem (handler-level limits, future).
//
// The cap is applied via http.MaxBytesReader. When the limit is
// exceeded, the underlying io.ReadAll returns an error, which Gin
// surfaces as a 413 Payload Too Large via the response. We also
// short-circuit explicitly so the error message is consistent with
// the rest of the API.
//
// Applied at the engine level so every endpoint inherits the cap
// without per-handler boilerplate. Skipped for /swagger/*any so
// the swagger-ui bundle (which is served via asset requests, not
// POSTs) is unaffected.
func BodyLimitMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !isMutatingMethod(c.Request.Method) {
			c.Next()
			return
		}
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, bodyLimitMaxBytes)
		// Reset Content-Length if present and too large; the
		// server uses Content-Length when available for faster
		// rejection, but MaxBytesReader is the authoritative cap.
		if c.Request.ContentLength > bodyLimitMaxBytes {
			c.AbortWithStatusJSON(http.StatusRequestEntityTooLarge, errorResponse{
				Error:   "payload_too_large",
				Details: "request body exceeds 1 MiB",
			})
			return
		}
		c.Next()
		// If MaxBytesReader tripped during the handler, Gin
		// surfaces the error via c.Errors. We translate the
		// sentinel into a clean 413 here instead of letting it
		// become a 500.
		for _, e := range c.Errors {
			var maxErr *http.MaxBytesError
			if errors.As(e.Err, &maxErr) {
				c.AbortWithStatusJSON(http.StatusRequestEntityTooLarge, errorResponse{
					Error:   "payload_too_large",
					Details: "request body exceeds 1 MiB",
				})
				return
			}
		}
	}
}

func isMutatingMethod(method string) bool {
	switch method {
	case http.MethodPost, http.MethodPatch, http.MethodPut, http.MethodDelete:
		return true
	default:
		return false
	}
}
