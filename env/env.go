package env

import (
	"github.com/thoriqadillah/gema"
)

var (
	APP_ENV string
	PORT    string
	HOST    string
)

func init() {
	APP_ENV = gema.Env("APP_ENV").String("development")
	PORT = gema.Env("APP_PORT").String(":8001")
	HOST = gema.Env("APP_HOST").String("localhost")
}
