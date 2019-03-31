package graphqlApi

import (
	"fmt"
)

func MakeOffsetCursor(offset int) string {
	return fmt.Sprintf("{offset: %d}", offset)
}
