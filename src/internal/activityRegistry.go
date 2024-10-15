package internal

import "github.com/GustavBW/bsc-multiplayer-backend/src/util"

type ActivityRegistry struct {
	ByID util.ConcurrentTypedMap[uint32, *Activity[any]]
}

func InitActivityRegistry() error {

	return nil
}
