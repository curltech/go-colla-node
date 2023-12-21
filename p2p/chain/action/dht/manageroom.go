package dht

import (
	"errors"
	"github.com/curltech/go-colla-core/logger"
	"github.com/curltech/go-colla-core/util/message"
	"github.com/curltech/go-colla-node/livekit"
	"github.com/curltech/go-colla-node/p2p/chain/action"
	"github.com/curltech/go-colla-node/p2p/chain/handler"
	"github.com/curltech/go-colla-node/p2p/msg/entity"
	"github.com/curltech/go-colla-node/p2p/msgtype"
	lksdk "github.com/livekit/protocol/livekit"
	"time"
)

type manageRoomAction struct {
	action.BaseAction
}

var ManageRoomAction manageRoomAction

type LiveKitManageRoom struct {
	ManageType   string                   `json:"manageType,omitempty"`
	EmptyTimeout int64                    `json:"emptyTimeout,omitempty"`
	Host         string                   `json:"host,omitempty"`
	RoomName     string                   `json:"roomName,omitempty"`
	Identities   []string                 `json:"identities,omitempty"`
	Names        []string                 `json:"names,omitempty"`
	Tokens       []string                 `json:"tokens,omitempty"`
	Participants []*lksdk.ParticipantInfo `json:"participants,omitempty"`
	Rooms        []*lksdk.Room            `json:"rooms,omitempty"`
}

func (this *manageRoomAction) Receive(chainMessage *entity.ChainMessage) (*entity.ChainMessage, error) {
	logger.Sugar.Infof("Receive %v message", this.MsgType)
	var response *entity.ChainMessage = nil
	liveKitManageRoom := &LiveKitManageRoom{}
	if chainMessage.Payload != nil {
		conditionBean, ok := chainMessage.Payload.(map[string]interface{})
		if !ok {
			response = handler.Error(chainMessage.MessageType, errors.New("ErrorCondition"))
			return response, nil
		}
		data, err := message.Marshal(conditionBean)
		if err != nil {
			response = handler.Error(chainMessage.MessageType, errors.New("ErrorCondition"))
			return response, nil
		}

		err = message.Unmarshal(data, liveKitManageRoom)
		if err != nil {
			response = handler.Error(chainMessage.MessageType, errors.New("ErrorCondition"))
			return response, nil
		}
	}
	roomServiceClient := livekit.GetRoomServiceClient()
	liveKitManageRoom.Host = livekit.LivekitParams.Host
	if liveKitManageRoom.ManageType == "create" {
		if liveKitManageRoom.RoomName == "" {
			response = handler.Error(chainMessage.MessageType, errors.New("ErrorRoomName"))
			return response, nil
		}
		if liveKitManageRoom.EmptyTimeout <= 0 {
			response = handler.Error(chainMessage.MessageType, errors.New("ErrorEmptyTimeout"))
			return response, nil
		}
		room, _ := roomServiceClient.CreateRoom(liveKitManageRoom.RoomName, uint32(liveKitManageRoom.EmptyTimeout), 0, "")
		if room != nil {
			rooms := make([]*lksdk.Room, 0)
			rooms = append(rooms, room)
			liveKitManageRoom.Rooms = rooms
			tokens, _ := roomServiceClient.CreateTokens(liveKitManageRoom.RoomName, liveKitManageRoom.Identities, liveKitManageRoom.Names, time.Duration(liveKitManageRoom.EmptyTimeout*1000*1000*1000), "")
			if tokens != nil {
				liveKitManageRoom.Tokens = tokens
			}
		}
	}
	if liveKitManageRoom.ManageType == "delete" {
		if liveKitManageRoom.RoomName == "" {
			response = handler.Error(chainMessage.MessageType, errors.New("ErrorRoomName"))
			return response, nil
		}
		deleteResponse, _ := roomServiceClient.DeleteRoom(liveKitManageRoom.RoomName)
		if deleteResponse == nil {
			liveKitManageRoom.RoomName = ""
		}
	}
	if liveKitManageRoom.ManageType == "list" {
		roomNames := make([]string, 0)
		rooms, _ := roomServiceClient.ListRooms(roomNames)
		if rooms != nil {
			liveKitManageRoom.Rooms = rooms
		}
	}
	if liveKitManageRoom.ManageType == "listParticipants" {
		if liveKitManageRoom.RoomName == "" {
			response = handler.Error(chainMessage.MessageType, errors.New("ErrorRoomName"))
			return response, nil
		}
		participants, _ := roomServiceClient.ListParticipants(liveKitManageRoom.RoomName)
		if participants != nil {
			liveKitManageRoom.Participants = participants
		}
	}
	response = handler.Response(chainMessage.MessageType, liveKitManageRoom)
	response.PayloadType = handler.PayloadType_Map

	return response, nil
}

func init() {
	ManageRoomAction = manageRoomAction{}
	ManageRoomAction.MsgType = msgtype.ManageRoom
	handler.RegistChainMessageHandler(msgtype.ManageRoom, ManageRoomAction.Send, ManageRoomAction.Receive, ManageRoomAction.Response)
}
