package nex_matchmake_extension

import (
	"encoding/hex"
	"fmt"
	"strconv"

	nex "github.com/PretendoNetwork/nex-go"
	match_making "github.com/PretendoNetwork/nex-protocols-go/match-making"
	matchmake_extension "github.com/PretendoNetwork/nex-protocols-go/matchmake-extension"
	"github.com/PretendoNetwork/nex-protocols-go/notifications"
	"github.com/PretendoNetwork/splatoon-secure/database"
	"github.com/PretendoNetwork/splatoon-secure/globals"
)

func JoinMatchmakeSessionWithParam(err error, client *nex.Client, callID uint32, joinMatchmakeSessionParam *match_making.JoinMatchmakeSessionParam) {

	// * From Jon: This was added here because this function had the wrong signature
	// * I have no idea if this works at all, I just got it to build
	gid := joinMatchmakeSessionParam.GID

	fmt.Println("===== MATCHMAKE SESSION JOIN =====")
	fmt.Println("GATHERING ID: " + strconv.Itoa((int)(gid)))

	database.AddPlayerToRoom(gid, client.PID(), uint32(1))

	hostpid, gamemode, region, gconfig, dlcmode := database.GetRoomInfo(gid)
	if hostpid == 0xffffffff {
		rmcResponse := nex.NewRMCResponse(matchmake_extension.ProtocolID, callID)
		rmcResponse.SetError(nex.Errors.RendezVous.InvalidGID)

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

	stationUrlsStrings := database.GetPlayerURLs(client.ConnectionID())
	stationUrls := make([]nex.StationURL, len(stationUrlsStrings))
	pid := strconv.FormatUint(uint64(client.PID()), 10)
	rvcid := strconv.FormatUint(uint64(client.ConnectionID()), 10)

	for i := 0; i < len(stationUrlsStrings); i++ {
		stationUrls[i] = *nex.NewStationURL(stationUrlsStrings[i])
		if stationUrls[i].Type() == "3" {
			natm_s := strconv.FormatUint(uint64(1), 10)
			natf_s := strconv.FormatUint(uint64(2), 10)
			stationUrls[i].SetNatm(natm_s)
			stationUrls[i].SetNatf(natf_s)
		}
		stationUrls[i].SetPID(pid)
		stationUrls[i].SetRVCID(rvcid)
		database.UpdatePlayerSessionURL(client.ConnectionID(), stationUrlsStrings[i], stationUrls[i].EncodeToString())
	}

	// TODO - Don't hardcode these things
	applicationData, _ := hex.DecodeString("088100000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000EA801C8B0000000000010100410000000010011C")
	sessionKey, _ := hex.DecodeString("161466a08c8df18b118ed5a67650a47435f081d09804a7c1902b145e18eff47c")
	matchmakeParamBytes, _ := hex.DecodeString("02000000040040535200030105004047495200010300000000000000")

	startedTime := nex.NewDateTime(0)
	startedTime = nex.NewDateTime(startedTime.Now())

	matchmakeParamStream := nex.NewStreamIn(matchmakeParamBytes, client.Server())

	matchmakeParam := match_making.NewMatchmakeParam()
	_ = matchmakeParam.ExtractFromStream(matchmakeParamStream)

	participationCount := len(database.GetRoomPlayers(gid))

	// TODO - We don't know what are all of the parameters of the session.
	// Here we try our best to recreate the structure

	matchmakeSession := match_making.NewMatchmakeSession()
	matchmakeSession.Gathering.ID = gid
	matchmakeSession.Gathering.OwnerPID = hostpid
	matchmakeSession.Gathering.HostPID = hostpid
	matchmakeSession.Gathering.MinimumParticipants = 0
	matchmakeSession.Gathering.MaximumParticipants = 8
	matchmakeSession.Gathering.ParticipationPolicy = 95
	matchmakeSession.Gathering.Flags = 2560
	matchmakeSession.GameMode = gamemode
	matchmakeSession.Attributes[3] = region
	matchmakeSession.Attributes[2] = gconfig
	matchmakeSession.Attributes[5] = dlcmode
	matchmakeSession.Attributes[6] = 0 // padding
	matchmakeSession.OpenParticipation = true
	matchmakeSession.MatchmakeSystemType = 1
	matchmakeSession.ApplicationData = applicationData
	matchmakeSession.ParticipationCount = uint32(participationCount)
	matchmakeSession.ProgressScore = 100
	matchmakeSession.SessionKey = sessionKey
	matchmakeSession.MatchmakeParam = matchmakeParam
	matchmakeSession.StartedTime = startedTime

	rmcResponseStream := nex.NewStreamOut(globals.NEXServer)

	rmcResponseStream.WriteStructure(matchmakeSession.Gathering)
	rmcResponseStream.WriteStructure(matchmakeSession)

	rmcResponseBody := rmcResponseStream.Bytes()

	rmcResponse := nex.NewRMCResponse(matchmake_extension.ProtocolID, callID)
	rmcResponse.SetSuccess(matchmake_extension.MethodJoinMatchmakeSessionWithParam, rmcResponseBody)

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

	rmcMessage := nex.NewRMCRequest()
	rmcMessage.SetProtocolID(notifications.ProtocolID)
	rmcMessage.SetCallID(0xffff0000 + callID)
	rmcMessage.SetMethodID(notifications.MethodProcessNotificationEvent)

	oEvent := notifications.NewNotificationEvent()
	oEvent.PIDSource = client.PID()
	oEvent.Type = 3001 // New participant
	oEvent.Param1 = gid
	oEvent.Param2 = client.PID()
	oEvent.Param3 = 1

	stream := nex.NewStreamOut(globals.NEXServer)
	oEventBytes := oEvent.Bytes(stream)
	rmcMessage.SetParameters(oEventBytes)
	rmcMessageBytes := rmcMessage.Bytes()

	targetClient := globals.NEXServer.FindClientFromPID(uint32(hostpid))

	messagePacket, _ := nex.NewPacketV1(targetClient, nil)
	messagePacket.SetVersion(1)
	messagePacket.SetSource(0xA1)
	messagePacket.SetDestination(0xAF)
	messagePacket.SetType(nex.DataPacket)
	messagePacket.SetPayload(rmcMessageBytes)

	messagePacket.AddFlag(nex.FlagNeedsAck)
	messagePacket.AddFlag(nex.FlagReliable)

	globals.NEXServer.Send(messagePacket)

	messagePacket, _ = nex.NewPacketV1(client, nil)
	messagePacket.SetVersion(1)
	messagePacket.SetSource(0xA1)
	messagePacket.SetDestination(0xAF)
	messagePacket.SetType(nex.DataPacket)
	messagePacket.SetPayload(rmcMessageBytes)

	messagePacket.AddFlag(nex.FlagNeedsAck)
	messagePacket.AddFlag(nex.FlagReliable)

	globals.NEXServer.Send(messagePacket)
}
