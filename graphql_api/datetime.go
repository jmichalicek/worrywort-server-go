package graphql_api

import (
	"errors"
	"time"
)

var ErrBadDateTimeInput = errors.New("Cannot Unmarshal DateTime")

type DateTime struct {
	time.Time
}

func (_ DateTime) ImplementsGraphQLType(name string) bool {
	return name == "DateTime"
}

func (t *DateTime) UnmarshalGraphQL(input interface{}) error {
	switch input := input.(type) {
	case time.Time:
		t.Time = input
		return nil
	case string:
		var err error
		t.Time, err = time.Parse(time.RFC3339, input)
		return err
	case int:
		t.Time = time.Unix(int64(input), 0)
		return nil
	default:
		return ErrBadDateTimeInput
	}
}
