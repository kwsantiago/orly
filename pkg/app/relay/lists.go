package relay

import (
	"sync"
)

// Lists manages lists of pubkeys, followed users, follows, and muted users with
// concurrency safety via a mutex.
//
// This list is designed primarily for owner-follow-list in mind, but with an
// explicit allowlist/blocklist set up, ownersFollowed corresponds to the
// allowed users, and ownersMuted corresponds to the blocked users, and all
// filtering logic will work the same way.
//
// Currently, there is no explicit purpose for the followedFollows list being
// separate from the ownersFollowed list, but there could be reasons for this
// distinction, such as rate limiting applying to the former and not the latter.
type Lists struct {
	sync.Mutex
	ownersPubkeys   [][]byte
	ownersFollowed  [][]byte
	followedFollows [][]byte
	ownersMuted     [][]byte
}

func (l *Lists) LenOwnersPubkeys() (ll int) {
	l.Lock()
	defer l.Unlock()
	ll = len(l.ownersPubkeys)
	return
}

func (l *Lists) OwnersPubkeys() (pks [][]byte) {
	l.Lock()
	defer l.Unlock()
	pks = append(pks, l.ownersPubkeys...)
	return
}

func (l *Lists) SetOwnersPubkeys(pks [][]byte) {
	l.Lock()
	defer l.Unlock()
	l.ownersPubkeys = pks
	return
}

func (l *Lists) LenOwnersFollowed() (ll int) {
	l.Lock()
	defer l.Unlock()
	ll = len(l.ownersFollowed)
	return
}

func (l *Lists) OwnersFollowed() (pks [][]byte) {
	l.Lock()
	defer l.Unlock()
	pks = append(pks, l.ownersFollowed...)
	return
}

func (l *Lists) SetOwnersFollowed(pks [][]byte) {
	l.Lock()
	defer l.Unlock()
	l.ownersFollowed = pks
	return
}

func (l *Lists) LenFollowedFollows() (ll int) {
	l.Lock()
	defer l.Unlock()
	ll = len(l.followedFollows)
	return
}

func (l *Lists) FollowedFollows() (pks [][]byte) {
	l.Lock()
	defer l.Unlock()
	pks = append(pks, l.followedFollows...)
	return
}

func (l *Lists) SetFollowedFollows(pks [][]byte) {
	l.Lock()
	defer l.Unlock()
	l.followedFollows = pks
	return
}

func (l *Lists) LenOwnersMuted() (ll int) {
	l.Lock()
	defer l.Unlock()
	ll = len(l.ownersMuted)
	return
}

func (l *Lists) OwnersMuted() (pks [][]byte) {
	l.Lock()
	defer l.Unlock()
	pks = append(pks, l.ownersMuted...)
	return
}

func (l *Lists) SetOwnersMuted(pks [][]byte) {
	l.Lock()
	defer l.Unlock()
	l.ownersMuted = pks
	return
}
