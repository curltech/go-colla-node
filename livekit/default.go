package livekit

import (
	"context"
	"fmt"
	"github.com/livekit/protocol/auth"
	"github.com/livekit/protocol/livekit"
	lksdk "github.com/livekit/server-sdk-go"
	"github.com/pion/webrtc/v3"
	"io"
	"time"
)

type RoomServiceClient struct {
	roomClient *lksdk.RoomServiceClient
}

// NewRoomServiceClient 创建房间服务的客户端，这个客户端连接到livekit sfu服务器
// 有权限创建房间，因此需要相应的APIKey和APISecret
func NewRoomServiceClient(host string, connectInfo lksdk.ConnectInfo) *RoomServiceClient {
	roomClient := lksdk.NewRoomServiceClient(host, connectInfo.APIKey, connectInfo.APISecret)

	return &RoomServiceClient{roomClient}
}

// CreateRoom 创建新的房间，
func (this *RoomServiceClient) CreateRoom(roomId string, maxParticipants uint32) (*livekit.Room, error) {
	return this.roomClient.CreateRoom(context.Background(), &livekit.CreateRoomRequest{
		Name:            roomId,
		MaxParticipants: maxParticipants,
		EmptyTimeout:    240 * 60, // 240 minutes
	})
}

// CreateToken 创建新的token
func (this *RoomServiceClient) CreateToken(roomId, identity string) (string, error) {
	at := this.roomClient.CreateToken()
	grant := &auth.VideoGrant{
		RoomJoin: true,
		Room:     roomId,
	}
	at.AddGrant(grant).
		SetIdentity(identity).
		SetValidFor(time.Hour)

	return at.ToJWT()
}

// ListRooms 列出房间
func (this *RoomServiceClient) ListRooms() (*livekit.ListRoomsResponse, error) {
	return this.roomClient.ListRooms(context.Background(), &livekit.ListRoomsRequest{})
}

// DeleteRoom 删除房间，所以参与人离开
func (this *RoomServiceClient) DeleteRoom(roomId string) (*livekit.DeleteRoomResponse, error) {
	return this.roomClient.DeleteRoom(context.Background(), &livekit.DeleteRoomRequest{
		Room: roomId,
	})
}

// ListParticipants 列出房间的参与人
func (this *RoomServiceClient) ListParticipants(roomId string) (*livekit.ListParticipantsResponse, error) {
	return this.roomClient.ListParticipants(context.Background(), &livekit.ListParticipantsRequest{
		Room: roomId,
	})
}

// GetParticipant 列出房间的参与人的详细信息
func (this *RoomServiceClient) GetParticipant(roomId string, identity string) (*livekit.ParticipantInfo, error) {
	return this.roomClient.GetParticipant(context.Background(), &livekit.RoomParticipantIdentity{
		Room:     roomId,
		Identity: identity,
	})
}

// RemoveParticipant 参与人从房间离开
func (this *RoomServiceClient) RemoveParticipant(roomId string, identity string) (*livekit.RemoveParticipantResponse, error) {
	return this.roomClient.RemoveParticipant(context.Background(), &livekit.RoomParticipantIdentity{
		Room:     roomId,
		Identity: identity,
	})
}

// MutePublishedTrack 关闭打开轨道的声音
func (this *RoomServiceClient) MutePublishedTrack(roomId string, identity string, trackSid string, muted bool) (*livekit.MuteRoomTrackResponse, error) {
	return this.roomClient.MutePublishedTrack(context.Background(), &livekit.MuteRoomTrackRequest{
		Room:     roomId,
		Identity: identity,
		TrackSid: trackSid,
		Muted:    muted,
	})
}

// NewLocalFileTrack 视频文件变成轨道
func NewLocalFileTrack(file string) (*lksdk.LocalSampleTrack, error) {
	return lksdk.NewLocalFileTrack(file,
		// control FPS to ensure synchronization
		lksdk.ReaderTrackWithFrameDuration(33*time.Millisecond),
		lksdk.ReaderTrackWithOnWriteComplete(func() { fmt.Println("track finished") }),
	)
}

// NewLocalReaderTrack 视频流变成轨道
func NewLocalReaderTrack(in io.ReadCloser, mime string) (*lksdk.LocalSampleTrack, error) {
	return lksdk.NewLocalReaderTrack(in, mime,
		lksdk.ReaderTrackWithFrameDuration(33*time.Millisecond),
		lksdk.ReaderTrackWithOnWriteComplete(func() { fmt.Println("track finished") }),
	)
}

// PublishTrack 把文件或者流轨道发布到房间
func PublishTrack(room *lksdk.Room, name string, track *lksdk.LocalSampleTrack, videoWidth int, videoHeight int) (*lksdk.LocalTrackPublication, error) {
	return room.LocalParticipant.PublishTrack(track, &lksdk.TrackPublicationOptions{
		Name:        name,
		VideoWidth:  videoWidth,
		VideoHeight: videoHeight,
	})
}

// ConnectToRoom 连接房间，设置回调函数，可以跟踪房间
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

// Disconnect 断开房间的连接
func Disconnect(room *lksdk.Room) {
	room.Disconnect()
}

func onTrackSubscribed(track *webrtc.TrackRemote, publication *lksdk.RemoteTrackPublication, rp *lksdk.RemoteParticipant) {
	fileName := fmt.Sprintf("%s-%s", rp.Identity(), track.ID())
	fmt.Println("write track to file ", fileName)
}
