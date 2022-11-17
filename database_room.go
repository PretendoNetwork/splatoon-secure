package main

import (
	"context"
	"math"
	"math/rand"
	"fmt"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	nexproto "github.com/PretendoNetwork/nex-protocols-go"

	"strconv"
	"encoding/hex"
)

type SearchCriteriaData struct {
	Attributes            []uint32
	GameMode              uint32
	MinParticipants       uint16
	MaxParticipants       uint16
	MatchmakeSystemType   uint32
}

func newRoom(hostPID uint32, hostRVCID uint32, matchmakeSession *nexproto.MatchmakeSession) uint32 {
	var gatheringId uint32
	var result bson.M

	for true {
		gatheringId = rand.Uint32() % 500000
		err := roomsCollection.FindOne(context.TODO(), bson.D{{"gid", gatheringId}}, options.FindOne()).Decode(&result)
		if err != nil {
			if err == mongo.ErrNoDocuments {
				break
			} else {
				panic(err)
			}
		} else {
			continue
		}
	}

	var searchCriteriaData SearchCriteriaData
	searchCriteriaData.Attributes = matchmakeSession.Attributes
	searchCriteriaData.GameMode = matchmakeSession.GameMode
	//searchCriteriaData.MinParticipants = matchmakeSession.Gathering.MinimumParticipants
	//searchCriteriaData.MaxParticipants = matchmakeSession.Gathering.MaximumParticipants
	searchCriteriaData.MatchmakeSystemType = matchmakeSession.MatchmakeSystemType

	matchmakeSession.MatchmakeParam = nexproto.NewMatchmakeParam()
	matchmakeSession.Attributes[1] = 0
	matchmakeSession.Attributes[4] = 0
	if((int)(matchmakeSession.GameMode) == 12){
		tmp := matchmakeSession.Attributes[3]
		matchmakeSession = nexproto.NewMatchmakeSession()
		matchmakeSession.GameMode = 12
		matchmakeSession.Attributes = make([]uint32, 6)
		matchmakeSession.MatchmakeParam = nexproto.NewMatchmakeParam()
		matchmakeSession.Attributes[3] = tmp
	}
	
	searchMatchmakeSession, _ := bson.Marshal(matchmakeSession)
	searchCriteriaDataBson, _ := bson.Marshal(searchCriteriaData)
	fmt.Println(hex.EncodeToString(searchCriteriaDataBson[:]))

	matchmakeSession.Gathering.ID = gatheringId
	matchmakeSession.Gathering.OwnerPID = hostPID
	matchmakeSession.Gathering.HostPID = hostPID
	matchmakeSession.SessionKey = []byte("00000000000000000000000000000000")

	players := make([][]int64, 12)
	for i := 0; i < 12; i++ {
		players[i] = make([]int64, 2)
	}
	gatheringDoc := bson.D{
		{"gid", gatheringId},
		{"hostPid", hostPID},
		{"hostRvcid", hostRVCID},
		{"players", players},
		{"searchMatchmakeSession", searchMatchmakeSession},
		{"searchCriteriaData", searchCriteriaDataBson},
		{"matchmakeSession", matchmakeSession}}
	_, err := roomsCollection.InsertOne(context.TODO(), gatheringDoc)
	if err != nil {
		panic(err)
	}

	return gatheringId
}

func addPlayerToRoom(gid uint32, pid uint32, rvcid uint32, addplayercount uint32) {
	var result bson.M

	err := roomsCollection.FindOne(context.TODO(), bson.D{{"gid", gid}}, options.FindOne()).Decode(&result)
	if err != nil {
		//panic(err)
		return
	}

	oldPlayerList := result["players"].(bson.A)
	newPlayerList := make([][]int64, 12)
	for i := 0; i < 12; i++ {
		if oldPlayerList[i].(bson.A)[0].(int64) == int64(pid) || oldPlayerList[i].(bson.A)[0].(int64) == -1*int64(pid) {
			newPlayerList[i] = make([]int64, 2)
		} else {
			newPlayerList[i] = make([]int64, 2)
			newPlayerList[i][0] = oldPlayerList[i].(bson.A)[0].(int64)
			newPlayerList[i][1] = oldPlayerList[i].(bson.A)[1].(int64)
		}
	}
	unassignedPlayers := addplayercount
	needToAddGuest := (unassignedPlayers > 1)
	for i := 0; i < 12; i++ {
		if newPlayerList[i][0] == 0 && newPlayerList[i][1] == 0 && unassignedPlayers > 0 {
			if unassignedPlayers == 1 && needToAddGuest {
				newPlayerList[i][0] = -1 * int64(pid)
				newPlayerList[i][1] = -1 * int64(rvcid)
				needToAddGuest = false
			} else {
				newPlayerList[i] = make([]int64, 2)
				newPlayerList[i][0] = int64(pid)
				newPlayerList[i][1] = int64(rvcid)
			}
			unassignedPlayers--
		}
	}

	matchmakeSession := nexproto.NewMatchmakeSession()

	bsonBytes, _ := bson.Marshal(result["matchmakeSession"].(bson.M))
	bson.Unmarshal(bsonBytes, &matchmakeSession)

	matchmakeSession.ParticipationCount += addplayercount
	
	_, err = roomsCollection.UpdateOne(context.TODO(), bson.D{{"gid", gid}}, bson.D{{"$set", bson.D{{"players", newPlayerList}, {"matchmakeSession", matchmakeSession}}}})
	if err != nil {
		//panic(err)
		return
	}
}

func removePlayerFromRoom(gid uint32, pid uint32) {
	var result bson.M

	err := roomsCollection.FindOne(context.TODO(), bson.D{{"gid", gid}}, options.FindOne()).Decode(&result)
	if err != nil {
		//panic(err)
		return
	}

	oldPlayerList := result["players"].(bson.A)
	newPlayerList := make([][]int64, 12)
	newplayercount := result["player_count"].(int64)
	for i := 0; i < 12; i++ {
		newPlayerList[i][0] = oldPlayerList[i].(bson.A)[0].(int64)
		newPlayerList[i][1] = oldPlayerList[i].(bson.A)[1].(int64)
		if newPlayerList[i][0] == int64(pid) || newPlayerList[i][0] == -1*int64(pid) {
			newPlayerList[i] = make([]int64, 2)
			newplayercount--
		}
	}

	_, err = roomsCollection.UpdateOne(context.TODO(), bson.D{{"gid", gid}}, bson.D{{"$set", bson.D{{"players", newPlayerList}, {"player_count", newplayercount}}}})
	if err != nil {
		//panic(err)
		return
	}

	if newplayercount <= 0 {
		destroyRoom(gid)
	}
}

func removePlayer(pid uint32) {
	var result bson.M
	arr := []uint32{pid}

	err := roomsCollection.FindOne(context.TODO(), bson.M{"players": bson.M{"$in": arr}}, options.FindOne()).Decode(&result)
	if err != nil {
		return
	}

	oldPlayerList := result["players"].(bson.A)
	newPlayerList := make([]int64, 12)

	matchmakeSession := nexproto.NewMatchmakeSession()

	bsonBytes, _ := bson.Marshal(result["matchmakeSession"].(bson.M))
	bson.Unmarshal(bsonBytes, &matchmakeSession)

	for i := 0; i < 12; i++ {
		newPlayerList[i] = oldPlayerList[i].(int64)
		if newPlayerList[i] == int64(pid) || newPlayerList[i] == -1*int64(pid) {
			newPlayerList[i] = 0
			matchmakeSession.ParticipationCount--
		}
	}

	_, err = roomsCollection.UpdateOne(context.TODO(), bson.D{{"gid", result["gid"]}}, bson.D{{"$set", bson.D{{"players", newPlayerList}, {"matchmakeSession", matchmakeSession}}}})
	if err != nil {
		return
		//panic(err)
	}

	if matchmakeSession.ParticipationCount <= 0 {
		destroyRoom((uint32)(result["gid"].(int64)))
	}
}

func destroyRoom(gid uint32) {
	_, err := roomsCollection.DeleteOne(context.TODO(), bson.D{{"gid", gid}})
	if err != nil {
		//panic(err)
		return
	}
}

func findRoomViaSearchCriteria(matchmakeSessionSearchCriteria *nexproto.MatchmakeSessionSearchCriteria) uint32 {
	var result bson.M
	var searchCriteriaData SearchCriteriaData

	for _, element := range matchmakeSessionSearchCriteria.Attribs {
		tmp, _ := strconv.Atoi(element)
		searchCriteriaData.Attributes = append(searchCriteriaData.Attributes, uint32(tmp))
	}
	tmp, _ := strconv.Atoi(matchmakeSessionSearchCriteria.GameMode)
	searchCriteriaData.GameMode = uint32(tmp)
	tmp, _ = strconv.Atoi(matchmakeSessionSearchCriteria.MinParticipants)
	//searchCriteriaData.MinParticipants = uint16(tmp)
	tmp, _ = strconv.Atoi(matchmakeSessionSearchCriteria.MaxParticipants)
	//searchCriteriaData.MaxParticipants = uint16(tmp)
	tmp, _ = strconv.Atoi(matchmakeSessionSearchCriteria.GameMode)
	searchCriteriaData.GameMode = uint32(tmp)
	tmp, _ = strconv.Atoi(matchmakeSessionSearchCriteria.MatchmakeSystemType)
	searchCriteriaData.MatchmakeSystemType = uint32(tmp)
	//maxplayersinroom := 12 - vacantcount
	searchCriteriaDataBson, _ := bson.Marshal(searchCriteriaData)
	//bson.Unmarshal(bsonBytes, &searchMatchmakeSessionBson)
	fmt.Println(hex.EncodeToString(searchCriteriaDataBson[:]))
	filter := bson.D{{"searchCriteriaData", searchCriteriaDataBson}}

	err := roomsCollection.FindOne(context.TODO(), filter, options.FindOne()).Decode(&result)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return math.MaxUint32
		} else {
			panic(err)
		}
	} else {
		return uint32(result["gid"].(int64))
	}
}

func findRoomViaMatchmakeSession(searchMatchmakeSession *nexproto.MatchmakeSession) uint32 {
	var result bson.M
	//maxplayersinroom := 12 - vacantcount

	searchMatchmakeSession.MatchmakeParam = nexproto.NewMatchmakeParam()
	searchMatchmakeSession.Attributes[1] = 0
	searchMatchmakeSession.Attributes[4] = 0
	if((int)(searchMatchmakeSession.GameMode) == 12){
		tmp := searchMatchmakeSession.Attributes[3]
		searchMatchmakeSession = nexproto.NewMatchmakeSession()
		searchMatchmakeSession.Attributes = make([]uint32, 6)
		searchMatchmakeSession.GameMode = 12
		searchMatchmakeSession.MatchmakeParam = nexproto.NewMatchmakeParam()
		searchMatchmakeSession.Attributes[3] = tmp
	}

	searchMatchmakeSessionBson, _ := bson.Marshal(searchMatchmakeSession)
	//bson.Unmarshal(bsonBytes, &searchMatchmakeSessionBson)
	fmt.Println(searchMatchmakeSessionBson)
	filter := bson.D{{"searchMatchmakeSession", searchMatchmakeSessionBson}}

	err := roomsCollection.FindOne(context.TODO(), filter, options.FindOne()).Decode(&result)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return math.MaxUint32
		} else {
			panic(err)
		}
	} else {
		return uint32(result["gid"].(int64))
	}
}

func getRoom(gid uint32) (uint32, uint32, *nexproto.MatchmakeSession) {
	var result bson.M

	err := roomsCollection.FindOne(context.TODO(), bson.D{{"gid", gid}}, options.FindOne()).Decode(&result)
	if err != nil {
		return 0xffffffff, 0xffffffff, nexproto.NewMatchmakeSession()
		//panic(err)
	}

	matchmakeSession := nexproto.NewMatchmakeSession()

	bsonBytes, _ := bson.Marshal(result["matchmakeSession"].(bson.M))
	bson.Unmarshal(bsonBytes, &matchmakeSession)

	return uint32(result["hostPid"].(int64)), uint32(result["hostRvcid"].(int64)), matchmakeSession
}

func getRoomPlayers(gid uint32) ([][]uint32) {
	var result bson.M

	err := roomsCollection.FindOne(context.TODO(), bson.D{{"gid", gid}}, options.FindOne()).Decode(&result)
	if err != nil {
		return make([][]uint32, 0)
		panic(err)
	}

	dbPlayerList := result["players"].(bson.A)
	pidList := make([][]uint32, 0)

	for i := 0; i < 12; i++ {
		if((uint32)(dbPlayerList[i].(bson.A)[0].(int64)) != 0){
			player := dbPlayerList[i].(bson.A)
			pidList = append(pidList, make([]uint32, 2))
			pidList[i][0] = uint32(player[0].(int64))
			pidList[i][1] = uint32(player[1].(int64))
		}
	}

	return pidList
}

func updateRoomHost(gid uint32, newownerpid uint32, newownerrvcid uint32) {
	_, err := roomsCollection.UpdateOne(context.TODO(), bson.D{{"gid", gid}}, bson.D{{"$set", bson.D{{"hostPid", int64(newownerpid)}, {"hostRvcid", int64(newownerrvcid)}}}})
	if err != nil {
		//panic(err)
		return
	}
}
