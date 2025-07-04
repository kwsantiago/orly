package server

import (
	"encoding/json"
	"net/http"
	"not.realy.lol/chk"
	"not.realy.lol/helpers"
	"not.realy.lol/log"
	"not.realy.lol/relayinfo"
	"not.realy.lol/version"
	"sort"
)

func (s *S) HandleRelayInfo(w http.ResponseWriter, r *http.Request) {
	remote := helpers.GetRemoteFromReq(r)
	log.T.F("handling relay info request from %s", remote)
	r.Header.Set("Content-Type", "application/json")
	var info *relayinfo.T
	supportedNIPs := relayinfo.GetList(
		relayinfo.BasicProtocol,
		relayinfo.RelayInformationDocument,
		relayinfo.GenericTagQueries,
		relayinfo.EventTreatment,
		relayinfo.ParameterizedReplaceableEvents,
		// relayinfo.CommandResults,
		// relayinfo.NostrMarketplace,
		// relayinfo.EncryptedDirectMessage,
		// relayinfo.EventDeletion,
		// relayinfo.ExpirationTimestamp,
		// relayinfo.ProtectedEvents,
		// relayinfo.RelayListMetadata,
	)
	// if s.ServiceURL(r) != "" {
	// 	supportedNIPs = append(supportedNIPs, relayinfo.Authentication.N())
	// }
	sort.Sort(supportedNIPs)
	log.T.Ln("supported NIPs", supportedNIPs)
	info = &relayinfo.T{
		Name:        s.Cfg.AppName,
		Description: version.Description,
		Nips:        supportedNIPs,
		Software:    "https://not.realy.lol",
		Version:     version.V,
		Limitation:  relayinfo.Limits{
			// MaxLimit: s.MaxLimit,
			// AuthRequired: s.AuthRequired(),
			// RestrictedWrites: !s.PublicReadable() || s.AuthRequired() || len(s.owners) > 0,
		},
		Icon: "https://cdn.satellite.earth/ac9778868fbf23b63c47c769a74e163377e6ea94d3f0f31711931663d035c4f6.png",
	}
	if err := json.NewEncoder(w).Encode(info); chk.E(err) {

	}
}
