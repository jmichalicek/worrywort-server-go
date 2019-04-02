package graphqlApi

import (
	"fmt"
)

func MakeOffsetCursor(offset int) string {
	// TODO: probably should just make a struct for this. I've had enough silly typos already.
	return fmt.Sprintf("{\"offset\": %d}", offset)
}
