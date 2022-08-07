package main

import (
	"fmt"
	"time"

	nex "github.com/PretendoNetwork/nex-go"
	nexproto "github.com/PretendoNetwork/nex-protocols-go"
	nexnattraversal "github.com/PretendoNetwork/nex-protocols-common-go/nat-traversal"
	nexsecure "github.com/PretendoNetwork/nex-protocols-common-go/secure-connection"
	nexmatchmaking "github.com/PretendoNetwork/nex-protocols-common-go/matchmaking"
)

type MatchmakingData struct {
	matchmakeSession *nexproto.MatchmakeSession
	clients          []*nex.Client
}

var nexServer *nex.Server
var secureServer *nexproto.SecureProtocol
var MatchmakingState []*MatchmakingData
var config *ServerConfig

var regionList = []string{"Worldwide", "Japan", "United States", "Europe", "Korea", "China", "Taiwan"}
var gameModes = []string{"Turf War", "Unk1", "Unk2", "Private Battle", "Unk4"}
var ccList = []string{"Unk", "200cc", "50cc", "100cc", "150cc", "Mirror", "BattleCC"}
var itemModes = []string{"Unk1", "Unk2", "Unk3", "Unk4", "Unk5", "Normal", "Unk7", "All Items", "Shells Only", "Bananas Only", "Mushrooms Only", "Bob-ombs Only", "No Items", "No Items or Coins", "Frantic"}
var vehicleModes = []string{"All Vehicles", "Karts Only", "Bikes Only"}
var controllerModes = []string{"Unk", "Tilt Only", "All Controls"}
var dlcModes = []string{"No DLC", "DLC Pack 1 Only", "DLC Pack 2 Only", "Both DLC Packs"}

func main() {
	MatchmakingState = append(MatchmakingState, nil)

	nexServer = nex.NewServer()
	nexServer.SetPrudpVersion(1)
	nexServer.SetNexVersion(40000)
	nexServer.SetKerberosKeySize(32)
	nexServer.SetAccessKey(config.AccessKey)
	nexServer.SetPingTimeout(5)

	nexServer.On("Data", func(packet *nex.PacketV1) {
		request := packet.RMCRequest()

		fmt.Println("==MK8 - Secure==")
		fmt.Printf("Protocol ID: %#v\n", request.ProtocolID())
		fmt.Printf("Method ID: %#v\n", request.MethodID())
		fmt.Printf("Method ID: %#v\n", nexServer.NexVersion())
		fmt.Println("=================")
	})

	nexServer.On("Kick", func(packet *nex.PacketV1) {
		pid := packet.Sender().PID()
		removePlayer(pid)

		fmt.Println("Leaving")
	})
	matchmakeExtensionProtocolServer := nexproto.NewMatchmakeExtensionProtocol(nexServer)
	matchMakingExtProtocolServer := nexproto.NewMatchMakingExtProtocol(nexServer)
	rankingProtocolServer := nexproto.NewRankingProtocol(nexServer)

	// have datastore available if called, but respond as unimplemented
	dataStorePrococolServer := nexproto.NewDataStoreProtocol(nexServer)
	_ = dataStorePrococolServer

	// Handle PRUDP CONNECT packet (not an RMC method)
	//nexServer.On("Connect", connect)

	natTraversalProtocolServer := nexnattraversal.InitNatTraversalProtocol(nexServer)
	nexnattraversal.GetConnectionUrls(getPlayerUrls)
	nexnattraversal.ReplaceConnectionUrl(updatePlayerSessionUrl)
	_ = natTraversalProtocolServer
	
	secureProto := nexsecure.NewCommonSecureConnectionProtocol(nexServer)
	secureProto.AddConnection(addPlayerSession)
	secureProto.UpdateConnection(updatePlayerSessionAll)
	secureProto.DoesConnectionExist(doesSessionExist)
	secureProto.ReplaceConnectionUrl(updatePlayerSessionUrl)
	secureServer = secureProto.SecureProtocol

	matchmakeExtensionProtocolServer.CloseParticipation(closeParticipation)
	matchmakeExtensionProtocolServer.AutoMatchmakeWithParam_Postpone(autoMatchmakeWithParam_Postpone)
	matchmakeExtensionProtocolServer.GetPlayingSession(getPlayingSession)
	matchmakeExtensionProtocolServer.UpdateProgressScore(updateProgressScore)
	matchmakeExtensionProtocolServer.CreateMatchmakeSessionWithParam(createMatchmakeSessionWithParam)
	matchmakeExtensionProtocolServer.JoinMatchmakeSessionWithParam(joinMatchmakeSessionWithParam)
	
	matchMakingProtocolServer := nexmatchmaking.InitMatchmakingProtocol(nexServer)
	nexmatchmaking.GetConnectionUrls(getPlayerUrls)
	nexmatchmaking.UpdateRoomHost(updateRoomHost)
	nexmatchmaking.DestroyRoom(destroyRoom)
	nexmatchmaking.GetRoomInfo(getRoomInfo)
	nexmatchmaking.GetRoomPlayers(getRoomPlayers)
	_ = matchMakingProtocolServer

	matchMakingExtProtocolServer.EndParticipation(endParticipation)

	rankingProtocolServer.UploadCommonData(uploadCommonData)

	nexServer.Listen(":" + config.ServerPort)
}

// Modified version of https://gist.github.com/ryanfitz/4191392

// will eventually be used to occasionally check for disconnected clients
// so as to clean their session info out of the database
func doEvery(d time.Duration, f func()) {
	for x := range time.Tick(d) {
		x = x
		f()
	}
}
