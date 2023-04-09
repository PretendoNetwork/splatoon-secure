package nex

import (
	"fmt"

	nex "github.com/PretendoNetwork/nex-go"
	nexmatchmaking "github.com/PretendoNetwork/nex-protocols-common-go/matchmaking"
	nexnattraversal "github.com/PretendoNetwork/nex-protocols-common-go/nat-traversal"
	nexsecure "github.com/PretendoNetwork/nex-protocols-common-go/secure-connection"
	match_making_ext "github.com/PretendoNetwork/nex-protocols-go/match-making-ext"
	matchmake_extension "github.com/PretendoNetwork/nex-protocols-go/matchmake-extension"
	"github.com/PretendoNetwork/nex-protocols-go/ranking"
	"github.com/PretendoNetwork/splatoon-secure/database"
	"github.com/PretendoNetwork/splatoon-secure/globals"
	nex_match_making_ext "github.com/PretendoNetwork/splatoon-secure/nex/match-making-ext"
	nex_matchmake_extension "github.com/PretendoNetwork/splatoon-secure/nex/matchmake-extension"
	nex_ranking "github.com/PretendoNetwork/splatoon-secure/nex/ranking"
)

func StartNEXServer() {
	globals.MatchmakingState = append(globals.MatchmakingState, nil)

	globals.NEXServer = nex.NewServer()
	globals.NEXServer.SetPRUDPVersion(1)
	globals.NEXServer.SetPRUDPProtocolMinorVersion(2)
	globals.NEXServer.SetDefaultNEXVersion(&nex.NEXVersion{
		Major: 3,
		Minor: 8,
		Patch: 3,
	})
	globals.NEXServer.SetKerberosPassword(globals.Config.AccessKey)
	globals.NEXServer.SetAccessKey("6f599f81")

	globals.NEXServer.On("Data", func(packet *nex.PacketV1) {
		request := packet.RMCRequest()

		fmt.Println("==Splatoon - Secure==")
		fmt.Printf("Protocol ID: %#v\n", request.ProtocolID())
		fmt.Printf("Method ID: %#v\n", request.MethodID())
		fmt.Println("===============")
	})

	globals.NEXServer.On("Kick", func(packet *nex.PacketV1) {
		pid := packet.Sender().PID()
		database.RemovePlayer(pid)

		fmt.Println("Leaving")
	})

	natTraversalProtocol := nexnattraversal.InitNatTraversalProtocol(globals.NEXServer)
	nexnattraversal.GetConnectionUrls(database.GetPlayerURLs)
	nexnattraversal.ReplaceConnectionUrl(database.UpdatePlayerSessionURL)
	_ = natTraversalProtocol

	secureConnectionProtocol := nexsecure.NewCommonSecureConnectionProtocol(globals.NEXServer)
	secureConnectionProtocol.AddConnection(database.AddPlayerSession)
	secureConnectionProtocol.UpdateConnection(database.UpdatePlayerSessionAll)
	secureConnectionProtocol.DoesConnectionExist(database.DoesSessionExist)
	secureConnectionProtocol.ReplaceConnectionUrl(database.UpdatePlayerSessionURL)

	matchmakeExtensionProtocol := matchmake_extension.NewMatchmakeExtensionProtocol(globals.NEXServer)
	matchmakeExtensionProtocol.CloseParticipation(nex_matchmake_extension.CloseParticipation)
	matchmakeExtensionProtocol.GetPlayingSession(nex_matchmake_extension.GetPlayingSession)
	matchmakeExtensionProtocol.UpdateProgressScore(nex_matchmake_extension.UpdateProgressScore)
	matchmakeExtensionProtocol.CreateMatchmakeSessionWithParam(nex_matchmake_extension.CreateMatchmakeSessionWithParam)
	matchmakeExtensionProtocol.JoinMatchmakeSessionWithParam(nex_matchmake_extension.JoinMatchmakeSessionWithParam)
	matchmakeExtensionProtocol.AutoMatchmakeWithParam_Postpone(nex_matchmake_extension.AutoMatchmakeWithParam_Postpone)

	matchMakingProtocol := nexmatchmaking.InitMatchmakingProtocol(globals.NEXServer)
	nexmatchmaking.GetConnectionUrls(database.GetPlayerURLs)
	nexmatchmaking.UpdateRoomHost(database.UpdateRoomHost)
	nexmatchmaking.DestroyRoom(database.DestroyRoom)
	nexmatchmaking.GetRoomInfo(database.GetRoomInfo)
	nexmatchmaking.GetRoomPlayers(database.GetRoomPlayers)
	_ = matchMakingProtocol

	matchMakingExtProtocol := match_making_ext.NewMatchMakingExtProtocol(globals.NEXServer)
	matchMakingExtProtocol.EndParticipation(nex_match_making_ext.EndParticipation)

	rankingProtocol := ranking.NewRankingProtocol(globals.NEXServer)
	rankingProtocol.UploadCommonData(nex_ranking.UploadCommonData)

	globals.NEXServer.Listen(":" + globals.Config.ServerPort)
}
