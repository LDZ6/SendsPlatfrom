package config

import "time"

var (
	AppId     = ""
	AppSerect = ""
)

var (
	StartTime int64
	EndTime   int64
	Reason    string
)

var (
	AccessToken          string
	ExpiresIn            int
	AccessTokenCreatTime time.Time
)
