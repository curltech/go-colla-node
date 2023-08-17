package client

import (
	"context"
	"fmt"
	"github.com/livekit/protocol/livekit"
	lksdk "github.com/livekit/server-sdk-go"
	"github.com/pion/webrtc/v3"
	"io"
	"time"
)

//创建房间服务的客户端
func NewRoomServiceClient(host string, connectInfo lksdk.ConnectInfo) *lksdk.RoomServiceClient {
	return lksdk.NewRoomServiceClient(host, connectInfo.APIKey, connectInfo.APISecret)
}

//创建新的房间，
func CreateRoom(roomClient *lksdk.RoomServiceClient, roomId string) (*livekit.Room, error) {
	return roomClient.CreateRoom(context.Background(), &livekit.CreateRoomRequest{
		Name: roomId,
	})
}

//列出房间
func ListRooms(roomClient *lksdk.RoomServiceClient) (*livekit.ListRoomsResponse, error) {
	return roomClient.ListRooms(context.Background(), &livekit.ListRoomsRequest{})
}

//删除房间，所以参与人离开
func DeleteRoom(roomClient *lksdk.RoomServiceClient, roomId string) (*livekit.DeleteRoomResponse, error) {
	return roomClient.DeleteRoom(context.Background(), &livekit.DeleteRoomRequest{
		Room: roomId,
	})
}

//列出房间的参与人
func ListParticipants(roomClient *lksdk.RoomServiceClient, roomId string) (*livekit.ListParticipantsResponse, error) {
	return roomClient.ListParticipants(context.Background(), &livekit.ListParticipantsRequest{
		Room: roomId,
	})
}

//参与人从房间离开
func RemoveParticipant(roomClient *lksdk.RoomServiceClient, roomId string, identity string) (*livekit.RemoveParticipantResponse, error) {
	return roomClient.RemoveParticipant(context.Background(), &livekit.RoomParticipantIdentity{
		Room:     roomId,
		Identity: identity,
	})
}

//关闭打开轨道的声音
func MutePublishedTrack(roomClient *lksdk.RoomServiceClient, roomId string, identity string, trackSid string, muted bool) (*livekit.MuteRoomTrackResponse, error) {
	return roomClient.MutePublishedTrack(context.Background(), &livekit.MuteRoomTrackRequest{
		Room:     roomId,
		Identity: identity,
		TrackSid: trackSid,
		Muted:    muted,
	})
}

//视频文件变成轨道
func NewLocalFileTrack(file string) (*lksdk.LocalSampleTrack, error) {
	return lksdk.NewLocalFileTrack(file,
		// control FPS to ensure synchronization
		lksdk.ReaderTrackWithFrameDuration(33*time.Millisecond),
		lksdk.ReaderTrackWithOnWriteComplete(func() { fmt.Println("track finished") }),
	)
}

//视频流变成轨道
func NewLocalReaderTrack(in io.ReadCloser, mime string) (*lksdk.LocalSampleTrack, error) {
	return lksdk.NewLocalReaderTrack(in, mime,
		lksdk.ReaderTrackWithFrameDuration(33*time.Millisecond),
		lksdk.ReaderTrackWithOnWriteComplete(func() { fmt.Println("track finished") }),
	)
}

//把轨道发布到房间
func PublishTrack(room *lksdk.Room, name string, track *lksdk.LocalSampleTrack, videoWidth int, videoHeight int) (*lksdk.LocalTrackPublication, error) {
	return room.LocalParticipant.PublishTrack(track, &lksdk.TrackPublicationOptions{
		Name:        name,
		VideoWidth:  videoWidth,
		VideoHeight: videoHeight,
	})
}

//连接房间，设置回调函数，可以跟踪房间
func ConnectToRoom(host string, connectInfo *lksdk.ConnectInfo) (*lksdk.Room, error) {
	if host == "" || connectInfo.APIKey == "" || connectInfo.APISecret == "" || connectInfo.RoomName == "" || connectInfo.ParticipantIdentity == "" {
		fmt.Println("invalid arguments.")
		return nil, nil
	}
	room, err := lksdk.ConnectToRoom(host, *connectInfo, &lksdk.RoomCallback{
		ParticipantCallback: lksdk.ParticipantCallback{
			OnTrackSubscribed: onTrackSubscribed,
		},
	})

	return room, err
}

//断开房间的连接
func Disconnect(room *lksdk.Room) {
	room.Disconnect()
}

func onTrackSubscribed(track *webrtc.TrackRemote, publication *lksdk.RemoteTrackPublication, rp *lksdk.RemoteParticipant) {
	fileName := fmt.Sprintf("%s-%s", rp.Identity(), track.ID())
	fmt.Println("write track to file ", fileName)
}
