package graphqlApi

import (
	"context"
)

type userErrorResolver struct {
	f   []string
	err string
}

// I think graphql-go newer versions have functionality to just use the fields on the struct
func (u *userErrorResolver) Field(ctx context.Context) []string { return u.f }
func (u *userErrorResolver) Error() string                      { return u.err }
