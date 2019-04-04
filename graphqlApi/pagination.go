package graphqlApi

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
)

type CursorData struct {
	Offset *int `json:"offset,omitempty"`
}

func DecodeCursor(cursor string) (CursorData, error) {
	// TODO: Put this somewhere reusable!!
	var cursordata CursorData
	raw, err := base64.StdEncoding.DecodeString(cursor)
	if err == nil {
		json.Unmarshal(raw, &cursordata)
	}
	return cursordata, err
}

func MakeOffsetCursor(offset int) (string, error) {
	// TODO: probably should just make a struct for this. I've had enough silly typos already.
	// return fmt.Sprintf("{\"offset\": %d}", offset)
	cd := CursorData{Offset: &offset}
	j, err := json.Marshal(cd)
	return fmt.Sprintf("%s", j), err
}

// steal a bit from ruby...
// go does not like !, so end with P for Panic
func MakeOffsetCursorP(offset int) string {
	c, err := MakeOffsetCursor(offset)
	if err != nil {
		panic(err)
	}
	return c
}
