package auth

import (
	"bytes"
	"orly.dev/pkg/encoders/event"
	"orly.dev/pkg/encoders/hex"
	"orly.dev/pkg/encoders/tag"
)

func CheckPrivilege(authedPubkey []byte, ev *event.E) (privileged bool) {
	if ev.Kind.IsPrivileged() {
		if len(authedPubkey) == 0 {
			// this is a shortcut because none of the following
			// tests would return true.
			return
		}
		// authed users when auth is required must be present in the
		// event if it is privileged.
		privileged = bytes.Equal(ev.Pubkey, authedPubkey)
		// if the authed pubkey matches the event author, it is
		// allowed.
		if !privileged {
			// check whether one of the p (mention) tags is
			// present designating the authed pubkey, as this means
			// the author wants the designated pubkey to be able to
			// access the event. this is the case for nip-4, nip-44
			// DMs, and gift-wraps. The query would usually have
			// been for precisely a p tag with their pubkey.
			eTags := ev.Tags.GetAll(tag.New("p"))
			var hexAuthedKey []byte
			hex.EncAppend(hexAuthedKey, authedPubkey)
			for _, e := range eTags.ToSliceOfTags() {
				if bytes.Equal(e.Value(), hexAuthedKey) {
					privileged = true
					break
				}
			}
		}
	}
	return
}
