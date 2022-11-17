package main

import (
	nex "github.com/PretendoNetwork/nex-go"
	nexproto "github.com/PretendoNetwork/nex-protocols-go"
)

func getSessionURLs(err error, client *nex.Client, callID uint32, gatheringId uint32) {
	var stationUrlStrings []string

	hostpid, hostRVCID, _ := getRoom(gatheringId)

	if(hostpid == 0xffffffff){
		// Build response packet
		rmcResponse := nex.NewRMCResponse(nexproto.MatchMakingProtocolID, callID)
		rmcResponse.SetSuccess(nexproto.MatchMakingMethodGetSessionURLs, nil)
	
		rmcResponseBytes := rmcResponse.Bytes()
	
		responsePacket, _ := nex.NewPacketV1(client, nil)
	
		responsePacket.SetVersion(1)
		responsePacket.SetSource(0xA1)
		responsePacket.SetDestination(0xAF)
		responsePacket.SetType(nex.DataPacket)
		responsePacket.SetPayload(rmcResponseBytes)
	
		responsePacket.AddFlag(nex.FlagNeedsAck)
		responsePacket.AddFlag(nex.FlagReliable)
	
		nexServer.Send(responsePacket)
	}

	stationUrlStrings = getPlayerUrls(hostRVCID)

	rmcResponseStream := nex.NewStreamOut(nexServer)
	rmcResponseStream.WriteListString(stationUrlStrings)

	rmcResponseBody := rmcResponseStream.Bytes()

	// Build response packet
	rmcResponse := nex.NewRMCResponse(nexproto.MatchMakingProtocolID, callID)
	rmcResponse.SetSuccess(nexproto.MatchMakingMethodGetSessionURLs, rmcResponseBody)

	rmcResponseBytes := rmcResponse.Bytes()

	responsePacket, _ := nex.NewPacketV1(client, nil)

	responsePacket.SetVersion(1)
	responsePacket.SetSource(0xA1)
	responsePacket.SetDestination(0xAF)
	responsePacket.SetType(nex.DataPacket)
	responsePacket.SetPayload(rmcResponseBytes)

	responsePacket.AddFlag(nex.FlagNeedsAck)
	responsePacket.AddFlag(nex.FlagReliable)

	nexServer.Send(responsePacket)
}
