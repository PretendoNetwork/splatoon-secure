package main

import (
	nex "github.com/PretendoNetwork/nex-go"
	nexproto "github.com/PretendoNetwork/nex-protocols-go"
)

func replaceURL(err error, client *nex.Client, callID uint32, oldStation *nex.StationURL, newStation *nex.StationURL) {
	updatePlayerSessionUrl(client.ConnectionID(), oldStation.EncodeToString(), newStation.EncodeToString())

	rmcResponse := nex.NewRMCResponse(nexproto.SecureProtocolID, callID)
	rmcResponse.SetSuccess(nexproto.SecureMethodReplaceURL, nil)

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
