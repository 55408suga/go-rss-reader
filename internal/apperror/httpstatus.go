package apperror

import "net/http"

// HTTPStatus maps an application error code to an HTTP status code.
func (c Code) HTTPStatus() int {
	switch c {
	case CodeNotFound:
		return http.StatusNotFound
	case CodeInvalidArgument:
		return http.StatusBadRequest
	case CodeConflict:
		return http.StatusConflict
	case CodeExternalUnavailable:
		return http.StatusBadGateway
	default:
		return http.StatusInternalServerError
	}
}

// CodeFromStatus maps an HTTP status code to the closest application error code.
func CodeFromStatus(status int) Code {
	switch status {
	case http.StatusBadRequest, http.StatusUnprocessableEntity:
		return CodeInvalidArgument
	case http.StatusNotFound:
		return CodeNotFound
	case http.StatusConflict:
		return CodeConflict
	case http.StatusBadGateway, http.StatusServiceUnavailable, http.StatusGatewayTimeout:
		return CodeExternalUnavailable
	default:
		if status >= http.StatusInternalServerError {
			return CodeInternal
		}
		return CodeInvalidArgument
	}
}
