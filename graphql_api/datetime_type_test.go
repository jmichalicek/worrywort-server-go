package graphql_api

import (
	"testing"
	"time"
)

func TestDateTimeType(t *testing.T) {
	t.Run("ImplementsGraphQLType", func(t *testing.T) {
		dt := DateTime{}
		if dt.ImplementsGraphQLType("DateTime") != true {
			t.Errorf("Expected ImplementsGraphQlType(\"DateTime\") to return true but got false")
		}

		if dt.ImplementsGraphQLType("Foo") != false {
			t.Errorf("Expected ImplementsGraphQlType(\"Foo\") to return false but got true")
		}
	})
	t.Run("UnmarshalGraphQL()", func(t *testing.T) {
		n := time.Now()
		var testmatrix = []struct {
			name     string
			input    interface{}
			expected time.Time
		}{
			{"time.Time", n, n},
			{"string", n.Round(time.Second).Format(time.RFC3339), n.Round(time.Second)},
			{"int", int(n.Round(time.Second).Unix()), n.Round(time.Second)},
		}

		for _, tm := range testmatrix {
			t.Run(tm.name, func(t *testing.T) {
				dt := DateTime{}
				err := dt.UnmarshalGraphQL(tm.input)
				if err != nil {
					t.Fatalf("Unexpected error: %s", err)
				}

				if !dt.Time.Equal(tm.expected) {
					t.Errorf("\nExpected: %v\nGot: %v", n, dt.Time)
				}
			})
		}

		t.Run("Unsupported time format", func(t *testing.T) {
			dt := DateTime{}
			err := dt.UnmarshalGraphQL(int64(1))
			if err != ErrBadDateTimeInput {
				t.Errorf("Expected `ErrBadDateTimeInput`, got: %v", err)
			}
		})
	})
}
