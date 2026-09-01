package auth

import "time"

type AuthCookie struct {
	Name   string
	TTL    time.Duration
	Secure bool
}
