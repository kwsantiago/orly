package ratel

import (
	"orly.dev/chk"
	"orly.dev/log"
	"orly.dev/ratel/prefixes"
)

func (r *T) Wipe() (err error) {
	log.W.F("nuking database at %s", r.dataDir)
	log.I.S(prefixes.AllPrefixes)
	if err = r.DB.DropPrefix(prefixes.AllPrefixes...); chk.E(err) {
		return
	}
	if err = r.DB.RunValueLogGC(0.8); chk.E(err) {
		return
	}
	return
}
