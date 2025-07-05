package database

import (
	"orly.dev/context"
	"orly.dev/filter"
	"orly.dev/interfaces/store"
)

func (d *D) QueryForIds(c context.T, f *filter.T) (
	evs []store.IdTsPk, err error,
) {

	return

}
