package nex_matchmake_extension

import (
	"encoding/hex"
	"fmt"
	"math"
	"strconv"

	nex "github.com/PretendoNetwork/nex-go"
	match_making "github.com/PretendoNetwork/nex-protocols-go/match-making"
	matchmake_extension "github.com/PretendoNetwork/nex-protocols-go/matchmake-extension"
	"github.com/PretendoNetwork/nex-protocols-go/notifications"
	"github.com/PretendoNetwork/splatoon-secure/database"
	"github.com/PretendoNetwork/splatoon-secure/globals"
)

var testGid uint32

func AutoMatchmakeWithParam_Postpone(err error, client *nex.Client, callID uint32, autoMatchmakeParam *match_making.AutoMatchmakeParam) {

	// * From Jon: This was added here because this function had the wrong signature
	// * I have no idea if this works at all, I just got it to build
	matchmakeSession := autoMatchmakeParam.SourceMatchmakeSession
	sourceGid := matchmakeSession.Gathering.ID

	gameConfig := matchmakeSession.Attributes[2]
	fmt.Println(strconv.FormatUint(uint64(gameConfig), 2))
	fmt.Println("===== MATCHMAKE SESSION SEARCH =====")
	fmt.Println("REGION: " + strconv.Itoa((int)(matchmakeSession.Attributes[3])))
	//fmt.Println("REGION: " + regionList[matchmakeSession.Attributes[3]])
	fmt.Println("GAME MODE: " + strconv.Itoa((int)(matchmakeSession.GameMode)))
	//fmt.Println("GAME MODE: " + gameModes[matchmakeSession.GameMode])
	//fmt.Println("CC: " + ccList[gameConfig%0b111])
	gameConfig = gameConfig >> 12
	//fmt.Println("DLC MODE: " + dlcModes[matchmakeSession.Attributes[5]&0xF])
	//fmt.Println("ITEM MODE: " + itemModes[gameConfig%0b1111])
	gameConfig = gameConfig >> 8
	//fmt.Println("VEHICLE MODE: " + vehicleModes[gameConfig%0b11])
	gameConfig = gameConfig >> 4
	//fmt.Println("CONTROLLER MODE: " + controllerModes[gameConfig%0b11])
	//fmt.Println("HAVE GUEST PLAYER: " + strconv.FormatBool(false))

	var gid uint32

	fmt.Println(sourceGid)

	if (int)(matchmakeSession.GameMode) == 12 {
		var team uint32
		if matchmakeSession.Attributes[3] == 0 {
			team = 1
		} else {
			team = 0
		}
		gid = database.FindRoom(matchmakeSession.GameMode, true, team, matchmakeSession.Attributes[2], uint32(1), matchmakeSession.Attributes[5]&0xF)
	} else {
		gid = database.FindRoom(matchmakeSession.GameMode, true, matchmakeSession.Attributes[3], matchmakeSession.Attributes[2], uint32(1), matchmakeSession.Attributes[5]&0xF)
	}
	if gid == math.MaxUint32 {
		gid = database.NewRoom(client.PID(), matchmakeSession.GameMode, true, matchmakeSession.Attributes[3], matchmakeSession.Attributes[2], uint32(1), matchmakeSession.Attributes[5]&0xF)
	}

	if matchmakeSession.GameMode == 12 {
		rmcMessage := nex.NewRMCRequest()
		rmcMessage.SetProtocolID(notifications.ProtocolID)
		rmcMessage.SetCallID(0xffff0000 + callID)
		rmcMessage.SetMethodID(notifications.MethodProcessNotificationEvent)

		oEvent := notifications.NewNotificationEvent()
		oEvent.PIDSource = client.PID()
		oEvent.Type = 122000 // Switch gathering
		oEvent.Param1 = gid
		oEvent.Param3 = 1

		for _, pid := range database.GetRoomPlayers(sourceGid) {
			if pid == 0 {
				continue
			}
			targetClient := globals.NEXServer.FindClientFromPID(uint32(pid))
			if targetClient != nil {
				oEvent.Param2 = pid

				stream := nex.NewStreamOut(globals.NEXServer)
				oEventBytes := oEvent.Bytes(stream)
				rmcMessage.SetParameters(oEventBytes)
				rmcMessageBytes := rmcMessage.Bytes()
				messagePacket, _ := nex.NewPacketV1(targetClient, nil)
				messagePacket.SetVersion(1)
				messagePacket.SetSource(0xA1)
				messagePacket.SetDestination(0xAF)
				messagePacket.SetType(nex.DataPacket)
				messagePacket.SetPayload(rmcMessageBytes)

				messagePacket.AddFlag(nex.FlagNeedsAck)
				messagePacket.AddFlag(nex.FlagReliable)

				globals.NEXServer.Send(messagePacket)
			} else {
				fmt.Println("not found")
			}
		}
	}

	fmt.Println("GATHERING ID: " + strconv.Itoa((int)(gid)))

	database.AddPlayerToRoom(gid, client.PID(), uint32(1))

	hostpid, gamemode, region, gconfig, dlcmode := database.GetRoomInfo(gid)
	participationCount := len(database.GetRoomPlayers(sourceGid))

	// TODO - Don't hardcode the session key
	sessionKey, _ := hex.DecodeString("161466a08c8df18b118ed5a67650a47435f081d09804a7c1902b145e18eff47c")

	// TODO - Use the real creation date of the room
	startedTime := nex.NewDateTime(0)
	startedTime = nex.NewDateTime(startedTime.Now())

	matchmakeSession.Gathering.ID = gid
	matchmakeSession.Gathering.OwnerPID = hostpid
	matchmakeSession.Gathering.HostPID = hostpid
	matchmakeSession.SessionKey = sessionKey
	matchmakeSession.GameMode = gamemode
	matchmakeSession.Attributes[3] = region
	matchmakeSession.Attributes[2] = gconfig
	matchmakeSession.Attributes[5] = dlcmode
	matchmakeSession.ParticipationCount = uint32(participationCount)
	matchmakeSession.StartedTime = startedTime

	rmcResponseStream := nex.NewStreamOut(globals.NEXServer)

	rmcResponseStream.WriteStructure(matchmakeSession.Gathering)
	rmcResponseStream.WriteStructure(matchmakeSession)

	rmcResponseBody := rmcResponseStream.Bytes()

	rmcResponse := nex.NewRMCResponse(matchmake_extension.ProtocolID, callID)
	rmcResponse.SetSuccess(matchmake_extension.MethodAutoMatchmakeWithParam_Postpone, rmcResponseBody)

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
	oEvent.PIDSource = hostpid
	oEvent.Type = 3001 // New participant
	oEvent.Param1 = gid
	oEvent.Param2 = client.PID()
	oEvent.Param3 = uint32(participationCount)
	/* if matchmakeSession.GameMode == 12 {
		oEvent.Param3 = 4
	} else {
		oEvent.Param3 = 1
		matchmakeSession.GameMode = 2
	} */
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
