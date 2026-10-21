// Package ids generates primary keys. Rows imported from the Prisma era keep
// their cuid strings; new rows get time-ordered UUIDv7 so inserts stay
// index-friendly. Both live in text columns.
package ids

import "github.com/google/uuid"

func New() string {
	id, err := uuid.NewV7()
	if err != nil {
		return uuid.NewString()
	}
	return id.String()
}
