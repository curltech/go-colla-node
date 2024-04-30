package livekit

import (
	"context"
	"fmt"
	"github.com/curltech/go-colla-core/config"
	"github.com/livekit/protocol/auth"
	"github.com/livekit/protocol/livekit"
	lksdk "github.com/livekit/server-sdk-go"
	"github.com/pion/webrtc/v3"
	"io"
	"strings"
	"time"
)

type livekitParams struct {
	Host      string
	ApiKey    string
	ApiSecret string
}

var LivekitParams = livekitParams{}

// RoomServiceClient 连接livekit服务器的客户端，可以创建房间，对服务器的房间进行管理
// 可以在控制层发布成restful服务
type RoomServiceClient struct {
	roomClient *lksdk.RoomServiceClient
}

var roomServiceClient = RoomServiceClient{}

// GetRoomServiceClient 创建房间服务的客户端，这个客户端连接到livekit sfu服务器
// 有权限创建房间，因此需要相应的APIKey和APISecret
func GetRoomServiceClient() *RoomServiceClient {
	roomClient := roomServiceClient.roomClient
	if roomClient == nil {
		LivekitParams.Host, _ = config.GetString("livekit.host")
		host := LivekitParams.Host
		if !strings.HasPrefix(host, "http://") {
			host = "http://" + host
		}
		LivekitParams.ApiKey, _ = config.GetString("livekit.apiKey")
		LivekitParams.ApiSecret, _ = config.GetString("livekit.apiSecret")
		roomClient = lksdk.NewRoomServiceClient(host, LivekitParams.ApiKey, LivekitParams.ApiSecret)
		roomServiceClient.roomClient = roomClient
	}
	return &roomServiceClient
}

// CreateRoom 创建新的房间，
func (svc *RoomServiceClient) CreateRoom(roomName string, emptyTimeout uint32, maxParticipants uint32, nodeId string) (*livekit.Room, error) {
	return svc.roomClient.CreateRoom(context.Background(), &livekit.CreateRoomRequest{
		Name:            roomName,
		MaxParticipants: maxParticipants,
		EmptyTimeout:    emptyTimeout, //240 * 60, // 240 minutes
		NodeId:          nodeId,
	})
}

func (svc *RoomServiceClient) CreateTokens(roomName string, identities []string, names []string, duration time.Duration, md string) ([]string, error) {
	tokens := make([]string, 0)
	i := 0
	for _, identity := range identities {
		name := names[i]
		token, err := svc.CreateToken(roomName, identity,
			name, duration, md)
		if err == nil {
			tokens = append(tokens, token)
		}
		i++
	}
	token, err := svc.CreateToken(roomName, "",
		"", duration, md)
	if err == nil {
		tokens = append(tokens, token)
	}

	return tokens, nil
}

// CreateToken 创建新的token，这个token在客户端连接房间的时候要使用
// 所以每个参与者都会有一个token
func (svc *RoomServiceClient) CreateToken(roomName, identity string, name string, duration time.Duration, md string) (string, error) {
	at := svc.roomClient.CreateToken()
	grant := &auth.VideoGrant{
		RoomJoin: true,
		Room:     roomName,
	}
	at.AddGrant(grant).
		SetValidFor(duration).SetMetadata(md)
	if len(name) > 0 {
		at.SetName(name)
	}
	if len(identity) > 0 {
		at.SetIdentity(identity)
	}

	return at.ToJWT()
}

// DeleteRoom 删除房间，所以参与人离开
func (svc *RoomServiceClient) DeleteRoom(roomName string) (*livekit.DeleteRoomResponse, error) {
	return svc.roomClient.DeleteRoom(context.Background(), &livekit.DeleteRoomRequest{
		Room: roomName,
	})
}

// ListRooms 列出房间
func (svc *RoomServiceClient) ListRooms(roomNames []string) ([]*livekit.Room, error) {
	roomsResponse, err := svc.roomClient.ListRooms(context.Background(), &livekit.ListRoomsRequest{
		Names: roomNames,
	})
	if err == nil {
		return roomsResponse.Rooms, nil
	}
	return nil, err
}

// GetParticipant 列出房间的参与人的详细信息
func (svc *RoomServiceClient) GetParticipant(roomName string, identity string) (*livekit.ParticipantInfo, error) {
	return svc.roomClient.GetParticipant(context.Background(), &livekit.RoomParticipantIdentity{
		Room:     roomName,
		Identity: identity,
	})
}

// ListParticipants 列出房间的参与人
func (svc *RoomServiceClient) ListParticipants(roomName string) ([]*livekit.ParticipantInfo, error) {
	participantsResponse, err := svc.roomClient.ListParticipants(context.Background(), &livekit.ListParticipantsRequest{
		Room: roomName,
	})
	if err == nil {
		return participantsResponse.GetParticipants(), nil
	}
	return nil, err
}

// RemoveParticipant 参与人从房间离开
func (svc *RoomServiceClient) RemoveParticipant(roomName string, identity string) (*livekit.RemoveParticipantResponse, error) {
	return svc.roomClient.RemoveParticipant(context.Background(), &livekit.RoomParticipantIdentity{
		Room:     roomName,
		Identity: identity,
	})
}

func (svc *RoomServiceClient) UpdateParticipant(roomName string, identity string,
	metadata string, permission *livekit.ParticipantPermission) (*livekit.ParticipantInfo, error) {
	return svc.roomClient.UpdateParticipant(context.Background(), &livekit.UpdateParticipantRequest{
		Room:       roomName,
		Identity:   identity,
		Metadata:   metadata,
		Permission: permission,
	})
}

// MutePublishedTrack 关闭打开轨道的声音
func (svc *RoomServiceClient) MutePublishedTrack(roomName string, identity string, trackSid string, muted bool) (*livekit.MuteRoomTrackResponse, error) {
	return svc.roomClient.MutePublishedTrack(context.Background(), &livekit.MuteRoomTrackRequest{
		Room:     roomName,
		Identity: identity,
		TrackSid: trackSid,
		Muted:    muted,
	})
}

func (svc *RoomServiceClient) UpdateRoomMetadata(roomName string, metadata string) (*livekit.Room, error) {
	return svc.roomClient.UpdateRoomMetadata(context.Background(), &livekit.UpdateRoomMetadataRequest{
		Room:     roomName,
		Metadata: metadata,
	})
}

func (svc *RoomServiceClient) UpdateSubscriptions(roomName string, identity string, trackSids []string,
	subscribe bool) (*livekit.UpdateSubscriptionsResponse, error) {
	return svc.roomClient.UpdateSubscriptions(context.Background(), &livekit.UpdateSubscriptionsRequest{
		Room:      roomName,
		Identity:  identity,
		TrackSids: trackSids,
		Subscribe: subscribe,
	})
}

// 连接房间的回调函数，返回远程流，轨道和参与者
func onTrackSubscribed(track *webrtc.TrackRemote, publication *lksdk.RemoteTrackPublication, rp *lksdk.RemoteParticipant) {
	fileName := fmt.Sprintf("%s-%s", rp.Identity(), track.ID())
	fmt.Println("write track to file ", fileName)
}

// Connect 连接房间，设置回调函数
func (svc *RoomServiceClient) Connect(host string, apiKey string, apiSecret string, roomName string, identity string) (*lksdk.Room, error) {
	roomCallback := &lksdk.RoomCallback{
		ParticipantCallback: lksdk.ParticipantCallback{
			OnTrackSubscribed: onTrackSubscribed,
		},
	}
	room, err := lksdk.ConnectToRoom(host, lksdk.ConnectInfo{
		APIKey:              apiKey,
		APISecret:           apiSecret,
		RoomName:            roomName,
		ParticipantIdentity: identity,
	}, roomCallback)
	if err != nil {
		panic(err)
	}

	return room, err
}

// PublishTrack 把文件或者流轨道发布到房间
func (svc *RoomServiceClient) PublishTrack(room *lksdk.Room, name string, track *lksdk.LocalSampleTrack, videoWidth int, videoHeight int) (*lksdk.LocalTrackPublication, error) {
	return room.LocalParticipant.PublishTrack(track, &lksdk.TrackPublicationOptions{
		Name:        name,
		VideoWidth:  videoWidth,
		VideoHeight: videoHeight,
	})
}

// Disconnect 断开房间的连接
func (svc *RoomServiceClient) Disconnect(room *lksdk.Room) {
	room.Disconnect()
}

// NewLocalFileTrack 视频文件变成轨道
//
//	ffmpeg -i <input.mp4> \
//	 -c:v libvpx -keyint_min 120 -qmax 50 -maxrate 2M -b:v 1M <output.ivf> \
//	 -c:a libopus -page_duration 20000 -vn <output.ogg>
//
//	ffmpeg -i <input.mp4> \
//	 -c:v libx264 -bsf:v h264_mp4toannexb -b:v 2M -profile baseline -pix_fmt yuv420p \
//	   -x264-params keyint=120 -max_delay 0 -bf 0 <output.h264> \
//	 -c:a libopus -page_duration 20000 -vn <output.ogg>
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
