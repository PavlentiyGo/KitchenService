package request

import (
	"encoding/json"
	"fmt"
	core_errors "foodDelivery/internal/core/errors"
	"net/http"

	"github.com/google/uuid"
)

type Validator interface {
	Struct(s any) error
}

func ParseUUIDPathParam(r *http.Request, name string) (uuid.UUID, error) {
	value := r.PathValue(name)
	id, err := uuid.Parse(value)
	if err != nil {
		return uuid.Nil, fmt.Errorf("invalid %s path parameter %q: %w", name, value, core_errors.ErrInvalidRequest)
	}

	return id, nil
}

func DecodeAndValidate(r *http.Request, dest any, validator Validator) error {
	err := json.NewDecoder(r.Body).Decode(dest)
	if err != nil {
		return fmt.Errorf("failed to decode request: %s: %w", err.Error(), core_errors.ErrInvalidRequest)
	}

	err = validator.Struct(dest)
	if err != nil {
		return fmt.Errorf("failed to validate request: %s: %w", err.Error(), core_errors.ErrInvalidRequest)
	}

	return nil
}
