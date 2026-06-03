package utils

import (
	"time"
)

var jst *time.Location

func init() {
	jst = time.FixedZone("Asia/Tokyo", 9*60*60)
}

func GetTimeJST() time.Time {
	return time.Now().In(jst)
}