package domain

import "time"

// User — владелец мониторов. PasswordHash наружу транспортом никогда не отдаётся.
type User struct {
	ID             string
	Email          string
	PasswordHash   string
	StatusPageSlug string
	CreatedAt      time.Time
}
