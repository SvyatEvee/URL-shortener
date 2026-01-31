package response

import (
	"fmt"
	"github.com/go-playground/validator/v10"
	"strings"
)

type Response struct {
	Status string `json:"status"`
	Error  string `json:"message,omitempty"`
}

const (
	StatusOK    = "OK"
	StatusError = "Error"
)

func OK() Response {
	return Response{
		Status: StatusOK,
	}
}

func Error(msg string) Response {
	return Response{
		Status: StatusError,
		Error:  msg,
	}
}

func ValidationError(errs validator.ValidationErrors) string {
	var errMsgs []string

	for _, err := range errs {
		switch err.ActualTag() {
		case "required":
			errMsgs = append(errMsgs, fmt.Sprintf("field %s is a required field", err.Field()))
		case "url":
			errMsgs = append(errMsgs, fmt.Sprintf("field %s is not a valid URL", err.Field()))
		default:
			errMsgs = append(errMsgs, fmt.Sprintf("field %s is not valid", err.Field()))
		}
	}

	return strings.Join(errMsgs, ", ")
}

func ValidateEnvVar(errs validator.ValidationErrors) string {
	var errMsgs []string

	for _, err := range errs {
		switch err.ActualTag() {
		case "required":
			errMsgs = append(errMsgs, fmt.Sprintf("%s is a required environment variable", err.Field()))
		case "number":
			errMsgs = append(errMsgs, fmt.Sprintf("%s must be a number", err.Field()))
		case "hostname":
			errMsgs = append(errMsgs, fmt.Sprintf("%s is not valid hostname", err.Field()))
		case "ip":
			errMsgs = append(errMsgs, fmt.Sprintf("%s is not valid ip", err.Field()))
		default:
			errMsgs = append(errMsgs, fmt.Sprintf(" %s is not valid", err.Field()))
		}
	}

	return strings.Join(errMsgs, ", ")
}
