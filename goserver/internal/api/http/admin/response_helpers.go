package admin

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	platformrepo "goserver/internal/platform/repository"
	servicecommon "goserver/internal/service/common"
)

const statusClientClosedRequest = 499

type requestError struct {
	status int
	code   string
	err    error
}

func (err requestError) Error() string {
	if err.err == nil {
		return ""
	}
	return err.err.Error()
}

func (err requestError) Unwrap() error {
	return err.err
}

func badRequestf(format string, args ...any) error {
	return requestError{
		status: http.StatusBadRequest,
		code:   "invalid_request",
		err:    fmt.Errorf(format, args...),
	}
}

func notImplementedf(format string, args ...any) error {
	return requestError{
		status: http.StatusNotImplemented,
		code:   "not_implemented",
		err:    fmt.Errorf("%w: %s", servicecommon.ErrUnsupportedServiceOperation, fmt.Sprintf(format, args...)),
	}
}

func writeJSON(writer http.ResponseWriter, status int, payload any) {
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(status)
	_ = json.NewEncoder(writer).Encode(payload)
}

func writeError(writer http.ResponseWriter, err error) {
	if err == nil {
		return
	}

	status := http.StatusInternalServerError
	code := "internal_error"
	var reqErr requestError
	switch {
	case errors.As(err, &reqErr):
		status = reqErr.status
		code = reqErr.code
	case errors.Is(err, platformrepo.ErrNotFound):
		status = http.StatusNotFound
		code = "not_found"
	case errors.Is(err, platformrepo.ErrAlreadyExists):
		status = http.StatusConflict
		code = "already_exists"
	case errors.Is(err, platformrepo.ErrConflict),
		errors.Is(err, platformrepo.ErrImmutableState),
		errors.Is(err, platformrepo.ErrInvalidTransition),
		errors.Is(err, platformrepo.ErrPreconditionFailed),
		errors.Is(err, servicecommon.ErrWorkflowNotReadyToContinue):
		status = http.StatusConflict
		code = "conflict"
	case errors.Is(err, servicecommon.ErrInvalidServiceRequest):
		status = http.StatusBadRequest
		code = "invalid_request"
	case errors.Is(err, servicecommon.ErrUnsupportedServiceOperation):
		status = http.StatusNotImplemented
		code = "not_implemented"
	case errors.Is(err, context.Canceled):
		status = statusClientClosedRequest
		code = "request_cancelled"
	case errors.Is(err, context.DeadlineExceeded):
		status = http.StatusGatewayTimeout
		code = "deadline_exceeded"
	}

	writeJSON(writer, status, ErrorResponseDTO{
		Error: err.Error(),
		Code:  code,
	})
}

func writeActionError(writer http.ResponseWriter, action string, request AdminActionRequestDTO, err error) {
	if isNoopError(err) {
		writeJSON(writer, http.StatusOK, AdminActionResponseDTO{
			Action:  action,
			Status:  "noop",
			Success: true,
			DryRun:  request.DryRun,
			Message: err.Error(),
		})
		return
	}
	writeError(writer, err)
}

func isNoopError(err error) bool {
	return errors.Is(err, servicecommon.ErrNothingToSubmit) ||
		errors.Is(err, servicecommon.ErrNothingToPoll) ||
		errors.Is(err, servicecommon.ErrNothingToReconcile) ||
		errors.Is(err, servicecommon.ErrNothingToValidate) ||
		errors.Is(err, servicecommon.ErrNothingToMaterialize) ||
		errors.Is(err, servicecommon.ErrNothingToFinalize)
}
