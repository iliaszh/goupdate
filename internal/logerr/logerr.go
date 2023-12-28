package logerr

import (
	"fmt"
)

type Error struct {
	Message     string
	InternalErr error
}

func (e Error) Error() string {
	return fmt.Sprintf("%s: %s", e.Message, e.InternalErr)
}
