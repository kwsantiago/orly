package relay

type Lists struct {
	OwnersPubkeys   [][]byte
	OwnersFollowed  [][]byte
	FollowedFollows [][]byte
	OwnersMuted     [][]byte
}
