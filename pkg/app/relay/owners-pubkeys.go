package relay

func (s *Server) OwnersPubkeys() (pks [][]byte) {
	pks = s.ownersPubkeys
	return
}
