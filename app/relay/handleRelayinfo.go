package relay

import (
	"encoding/json"
	"net/http"
	"orly.dev/interfaces/relay"
	"orly.dev/utils/chk"
	"orly.dev/utils/log"
	"orly.dev/version"
	"sort"

	"orly.dev/protocol/relayinfo"
)

func (s *Server) handleRelayInfo(w http.ResponseWriter, r *http.Request) {
	r.Header.Set("Content-Type", "application/json")
	log.I.Ln("handling relay information document")
	var info *relayinfo.T
	if informationer, ok := s.relay.(relay.Informationer); ok {
		info = informationer.GetNIP11InformationDocument()
	} else {
		supportedNIPs := relayinfo.GetList(
			relayinfo.BasicProtocol,
			relayinfo.Authentication,
			// relayinfo.EncryptedDirectMessage,
			relayinfo.EventDeletion,
			relayinfo.RelayInformationDocument,
			relayinfo.GenericTagQueries,
			// relayinfo.NostrMarketplace,
			relayinfo.EventTreatment,
			// relayinfo.CommandResults,
			relayinfo.ParameterizedReplaceableEvents,
			// relayinfo.ExpirationTimestamp,
			// relayinfo.ProtectedEvents,
			// relayinfo.RelayListMetadata,
		)
		sort.Sort(supportedNIPs)
		log.T.Ln("supported NIPs", supportedNIPs)
		info = &relayinfo.T{
			Name:        s.relay.Name(),
			Description: version.Description,
			Nips:        supportedNIPs, Software: version.URL,
			Version: version.V,
			Limitation: relayinfo.Limits{
				AuthRequired: s.authRequired,
			},
			Icon: "https://cdn.satellite.earth/ac9778868fbf23b63c47c769a74e163377e6ea94d3f0f31711931663d035c4f6.png",
		}
	}
	if err := json.NewEncoder(w).Encode(info); chk.E(err) {
	}
}
