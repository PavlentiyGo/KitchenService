package http_middleware

import (
	"context"

	"github.com/google/uuid"
)

type userIdKey struct{}

func GetUserIdFromCtx(ctx context.Context) (uuid.UUID, error) {
	userId := ctx.Value(userIdKey{}).(string)
	userIdUUID, err := uuid.Parse(userId)
	if err != nil {
		return uuid.Nil, err
	}
	return userIdUUID, nil
}

type companyIdKey struct{}

func GetCompanyIdFromCtx(ctx context.Context) (uuid.UUID, error) {
	companyId := ctx.Value(companyIdKey{}).(string)
	companyIdUUID, err := uuid.Parse(companyId)
	if err != nil {
		return uuid.Nil, err
	}
	return companyIdUUID, nil
}
