package repository

import (
	"fmt"
	"time"
)

func GetIdToRecord(code string) string {
	now := time.Now()
	idToReturn := code + fmt.Sprintf("%d", now.Unix())
	return idToReturn
}
