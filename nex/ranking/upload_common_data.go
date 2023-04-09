package nex_ranking

import (
	nex "github.com/PretendoNetwork/nex-go"
	"github.com/PretendoNetwork/nex-protocols-go/ranking"
	"github.com/PretendoNetwork/splatoon-secure/globals"
)

func UploadCommonData(err error, client *nex.Client, callID uint32, commonData []byte, uniqueID uint64) {
	rmcResponse := nex.NewRMCResponse(ranking.ProtocolID, callID)
	rmcResponse.SetSuccess(ranking.MethodUploadCommonData, nil)

	rmcResponseBytes := rmcResponse.Bytes()

	responsePacket, _ := nex.NewPacketV1(client, nil)

	responsePacket.SetVersion(1)
	responsePacket.SetSource(0xA1)
	responsePacket.SetDestination(0xAF)
	responsePacket.SetType(nex.DataPacket)
	responsePacket.SetPayload(rmcResponseBytes)

	responsePacket.AddFlag(nex.FlagNeedsAck)
	responsePacket.AddFlag(nex.FlagReliable)

	globals.NEXServer.Send(responsePacket)
}
