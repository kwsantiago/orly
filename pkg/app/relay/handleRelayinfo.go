package relay

import (
	"encoding/json"
	"net/http"
	"orly.dev/pkg/interfaces/relay"
	"orly.dev/pkg/protocol/relayinfo"
	"orly.dev/pkg/utils/chk"
	"orly.dev/pkg/utils/log"
	"orly.dev/pkg/version"
	"sort"
)

// HandleRelayInfo generates and returns a relay information document in JSON
// format based on the server's configuration and supported NIPs.
//
// # Parameters
//
//   - w: HTTP response writer used to send the generated document.
//
//   - r: HTTP request object containing incoming client request data.
//
// # Expected Behaviour
//
// The function constructs a relay information document using either the
// Informer interface implementation or predefined server configuration. It
// returns this document as a JSON response to the client.
func (s *Server) HandleRelayInfo(w http.ResponseWriter, r *http.Request) {
	r.Header.Set("Content-Type", "application/json")
	log.I.Ln("handling relay information document")
	var info *relayinfo.T
	if informationer, ok := s.relay.(relay.Informer); ok {
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
				AuthRequired: s.C.AuthRequired,
			},
			Icon: "https://cdn.satellite.earth/ac9778868fbf23b63c47c769a74e163377e6ea94d3f0f31711931663d035c4f6.png",
		}
	}
	if err := json.NewEncoder(w).Encode(info); chk.E(err) {
	}
}
