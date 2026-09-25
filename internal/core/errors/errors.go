package core_errors

import "errors"

var ErrInvalidRequest = errors.New("invalid request")
var ErrNotFound = errors.New("not found")
var ErrCompanyInactive = errors.New("company is inactive")
var ErrProductUnavailable = errors.New("product is unavailable")
var ErrInsufficientStock = errors.New("insufficient product stock")

var ErrNotAuthorized = errors.New("no user id in request")
var ErrInvalidAuthorization = errors.New("invalid user id in request")
var ErrConflict = errors.New("conflict")
var ErrForbidden = errors.New("you are not allowed to do it")
