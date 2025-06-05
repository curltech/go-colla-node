package dht

import (
	"context"
	"fmt"
	"github.com/curltech/go-colla-core/logger"
	"github.com/curltech/go-colla-node/consensus/std"
	"github.com/curltech/go-colla-node/libp2p/global"
	"github.com/curltech/go-colla-node/libp2p/ns"
	"github.com/curltech/go-colla-node/libp2p/util"
	"github.com/curltech/go-colla-node/p2p/chain/action"
	"github.com/curltech/go-colla-node/p2p/chain/handler"
	"github.com/curltech/go-colla-node/p2p/chain/handler/sender"
	"github.com/curltech/go-colla-node/p2p/dht/entity"
	"github.com/curltech/go-colla-node/p2p/dht/service"
	entity2 "github.com/curltech/go-colla-node/p2p/msg/entity"
	"github.com/curltech/go-colla-node/p2p/msgtype"
	"github.com/curltech/go-push-sdk/push/huawei_channel"
	"github.com/curltech/go-push-sdk/push/oppo_channel"
	"github.com/curltech/go-push-sdk/push/setting"
	"github.com/curltech/go-push-sdk/push/vivo_channel"
	"github.com/google/uuid"
	"strings"
)

type p2pChatAction struct {
	action.BaseAction
}

var P2pChatAction p2pChatAction

/*
*
接收消息进行处理，返回为空则没有返回消息，否则，有返回消息
*/
func (act *p2pChatAction) Receive(chainMessage *entity2.ChainMessage) (*entity2.ChainMessage, error) {
	// 查找最终目标会话
	peerClient, _, _ := sender.Lookup(chainMessage.TargetPeerId, chainMessage.TargetClientId)
	logger.Sugar.Infof("Receive %v message", act.MsgType)
	var response *entity2.ChainMessage = nil

	if peerClient != nil {
		handler.Decrypt(chainMessage)
		response, _ = std.GetStdConsensus().ReceiveConsensus(chainMessage)
		// push notification
		var srcPeerClientName string = ""
		srcPeerId := util.GetPeerId(chainMessage.SrcPeerId)
		srcKey := ns.GetPeerClientKey(srcPeerId)
		srcPeerClients, err := service.GetPeerClientService().GetLocals(srcKey, "")
		if err == nil && srcPeerClients != nil && len(srcPeerClients) > 0 {
			srcPeerClientName = srcPeerClients[0].Name
			content := getContent(peerClient.Language, srcPeerClientName)
			prefixArr := strings.Split(peerClient.ClientType, "(")
			switch prefixArr[0] {
			case "PC":
				// do nothing
			case "Apple":
				pushApple(peerClient, content)
			case "HUAWEI":
				pushHuawei(peerClient, content)
			case "Xiaomi":
				pushXiaomi(peerClient, content)
			case "OPPO":
				pushOppo(peerClient, content)
			case "VIVO":
				pushVivo(peerClient, content)
			case "Meizu":
				pushMeizu(peerClient, content)
			default:
				// GCM
				// URORA
			}
		}

		return response, nil
	}

	return nil, nil
}

func pushApple(peerClient *entity.PeerClient, content string) {
	iosTokenClient, err := global.Global.PushRegisterClient.GetIosTokenClient()
	if err != nil {
		logger.Sugar.Errorf("ios GetIosTokenClient error: %+v\n", err)
	} else {
		var deviceTokens = []string{
			peerClient.DeviceToken,
		}
		msg := &setting.PushMessageRequest{
			DeviceTokens: deviceTokens,
			Message: &setting.Message{
				BusinessId: uuid.New().String(),
				Title:      getTitle(peerClient.Language),
				SubTitle:   "",
				Content:    content,
				Extra: map[string]string{
					"type":        "TodoRemind",
					"link_type":   "TaskList",
					"link_params": "[]",
				},
				CallBack:      "",
				CallbackParam: "",
			},
		}
		ctx := context.Background()
		respPush, err := iosTokenClient.PushNotice(ctx, msg)
		if err != nil {
			logger.Sugar.Errorf("ios push error: %+v\n", err)
		}
		logger.Sugar.Infof("ios push response: %+v\n", respPush)
	}
}

func pushHuawei(peerClient *entity.PeerClient, content string) {
	huaweiClient, err := global.Global.PushRegisterClient.GetHUAWEIClient()
	if err != nil {
		logger.Sugar.Errorf("huawei GetHUAWEIClient error: %+v\n", err)
	} else {
		ctx := context.Background()
		respPush, _ := pushHuaweiSub(peerClient, huaweiClient, content)
		if respPush == nil || respPush.(*huawei_channel.PushMessageResponse).Code != "80000000" {
			accessTokenResp, err := huaweiClient.GetAccessToken(ctx)
			if err != nil {
				logger.Sugar.Errorf("huawei get access_token error: %+v\n", err)
			} else {
				global.Global.HuaweiAccessToken = accessTokenResp.(*huawei_channel.AccessTokenResp).AccessToken
				pushHuaweiSub(peerClient, huaweiClient, content)
			}
		}
	}
}

func pushHuaweiSub(peerClient *entity.PeerClient, huaweiClient setting.PushClientInterface, content string) (interface{}, error) {
	var error error = nil
	var deviceTokens = []string{
		peerClient.DeviceToken,
	}
	msg := &setting.PushMessageRequest{
		DeviceTokens: deviceTokens,
		AccessToken:  global.Global.HuaweiAccessToken,
		Message: &setting.Message{
			BusinessId:    uuid.New().String(),
			Title:         getTitle(peerClient.Language),
			SubTitle:      "",
			Content:       content,
			CallBack:      "",
			CallbackParam: "",
		},
	}
	ctx := context.Background()
	respPush, err := huaweiClient.PushNotice(ctx, msg)
	if err != nil {
		logger.Sugar.Errorf("huawei push error: %+v\n", err)
		error = err
	}
	logger.Sugar.Infof("huawei push response: %+v\n", respPush)
	return respPush, error
}

func pushXiaomi(peerClient *entity.PeerClient, content string) {
	xiaomiClient, err := global.Global.PushRegisterClient.GetXIAOMIClient()
	if err != nil {
		logger.Sugar.Errorf("xiaomi GetXIAOMIClient error: %+v\n", err)
	} else {
		var deviceTokens = []string{
			peerClient.DeviceToken,
		}
		msg := &setting.PushMessageRequest{
			DeviceTokens: deviceTokens,
			AccessToken:  "",
			Message: &setting.Message{
				BusinessId:    uuid.New().String(),
				Title:         getTitle(peerClient.Language),
				SubTitle:      "",
				Content:       content,
				CallBack:      "",
				CallbackParam: "",
			},
		}
		ctx := context.Background()
		respPush, err := xiaomiClient.PushNotice(ctx, msg)
		if err != nil {
			logger.Sugar.Errorf("xiaomi push error: %+v\n", err)
		}
		logger.Sugar.Infof("xiaomi push response: %+v\n", respPush)
	}
}

func pushOppo(peerClient *entity.PeerClient, content string) {
	oppoClient, err := global.Global.PushRegisterClient.GetOPPOClient()
	if err != nil {
		logger.Sugar.Errorf("oppo GetOPPOClient error: %+v\n", err)
	} else {
		ctx := context.Background()
		respPush, _ := pushOppoSub(peerClient, oppoClient, content)
		if respPush == nil || respPush.(*oppo_channel.PushMessageResponse).Code != 0 {
			authTokenResp, err := oppoClient.GetAccessToken(ctx)
			if err != nil {
				logger.Sugar.Errorf("oppo get auth_token error: %+v\n", err)
			} else {
				global.Global.OppoAccessToken = authTokenResp.(*oppo_channel.AuthTokenResp).Data.AuthToken
				pushOppoSub(peerClient, oppoClient, content)
			}
		}
	}
}

func pushOppoSub(peerClient *entity.PeerClient, oppoClient setting.PushClientInterface, content string) (interface{}, error) {
	var error error = nil
	var deviceTokens = []string{
		peerClient.DeviceToken,
	}
	msg := &setting.PushMessageRequest{
		DeviceTokens: deviceTokens,
		AccessToken:  global.Global.OppoAccessToken,
		Message: &setting.Message{
			BusinessId:    uuid.New().String(),
			Title:         getTitle(peerClient.Language),
			SubTitle:      "",
			Content:       content,
			CallBack:      "",
			CallbackParam: "",
		},
	}
	ctx := context.Background()
	respPush, err := oppoClient.PushNotice(ctx, msg)
	if err != nil {
		logger.Sugar.Errorf("oppo push error: %+v\n", err)
		error = err
	}
	logger.Sugar.Infof("oppo push response: %+v\n", respPush)
	return respPush, error
}

func pushVivo(peerClient *entity.PeerClient, content string) {
	vivoClient, err := global.Global.PushRegisterClient.GetVIVOClient()
	if err != nil {
		logger.Sugar.Errorf("vivo GetVIVOClient error: %+v\n", err)
	} else {
		ctx := context.Background()
		respPush, _ := pushVivoSub(peerClient, vivoClient, content)
		if respPush == nil || respPush.(*vivo_channel.PushMessageResponse).Result != 0 {
			authTokenResp, err := vivoClient.GetAccessToken(ctx)
			if err != nil {
				logger.Sugar.Errorf("vivo get auth_token error: %+v\n", err)
			} else {
				global.Global.VivoAccessToken = authTokenResp.(*vivo_channel.AuthTokenResp).AuthToken
				pushVivoSub(peerClient, vivoClient, content)
			}
		}
	}
}

func pushVivoSub(peerClient *entity.PeerClient, vivoClient setting.PushClientInterface, content string) (interface{}, error) {
	var error error = nil
	var deviceTokens = []string{
		peerClient.DeviceToken,
	}
	msg := &setting.PushMessageRequest{
		DeviceTokens: deviceTokens,
		AccessToken:  global.Global.VivoAccessToken,
		Message: &setting.Message{
			BusinessId:    uuid.New().String(),
			Title:         getTitle(peerClient.Language),
			SubTitle:      "",
			Content:       content,
			CallBack:      "",
			CallbackParam: "",
		},
	}
	ctx := context.Background()
	respPush, err := vivoClient.PushNotice(ctx, msg)
	if err != nil {
		logger.Sugar.Errorf("vivo push error: %+v\n", err)
		error = err
	}
	logger.Sugar.Infof("vivo push response: %+v\n", respPush)
	return respPush, error
}

func pushMeizu(peerClient *entity.PeerClient, content string) {
	meizuClient, err := global.Global.PushRegisterClient.GetMEIZUClient()
	if err != nil {
		logger.Sugar.Errorf("meizu GetMEIZUClient error: %+v\n", err)
	} else {
		var deviceTokens = []string{
			peerClient.DeviceToken,
		}
		msg := &setting.PushMessageRequest{
			DeviceTokens: deviceTokens,
			AccessToken:  "",
			Message: &setting.Message{
				BusinessId:    uuid.New().String(),
				Title:         getTitle(peerClient.Language),
				SubTitle:      "",
				Content:       content,
				CallBack:      "",
				CallbackParam: "",
			},
		}
		ctx := context.Background()
		respPush, err := meizuClient.PushNotice(ctx, msg)
		if err != nil {
			logger.Sugar.Errorf("meizu push error: %+v\n", err)
		}
		logger.Sugar.Infof("meizu push response: %+v\n", respPush)
	}
}

func getTitle(language string) string {
	switch language {
	case "en-us":
		return "Message Reminder"
	case "ja-jp":
		return "メッセージ通知"
	case "ko-kr":
		return "메시지 알림"
	case "zh-hans":
		return "消息提醒"
	case "zh-tw":
		return "消息提醒"
	default:
		return "Message Reminder"
	}
}

func getContent(language string, name string) string {
	switch language {
	case "en-us":
		return fmt.Sprintf("You have 1 message from %v", name)
	case "ja-jp":
		return fmt.Sprintf("%v からのメッセージが1つあります", name)
	case "ko-kr":
		return fmt.Sprintf("1개의 메시지가 있습니다 %v", name)
	case "zh-hans":
		return fmt.Sprintf("您有1条来自 %v 的消息", name)
	case "zh-tw":
		return fmt.Sprintf("您有1條來自 %v 的消息", name)
	default:
		return fmt.Sprintf("You have 1 message from %v", name)
	}
}

func init() {
	P2pChatAction = p2pChatAction{}
	P2pChatAction.MsgType = msgtype.P2PCHAT
	handler.RegistChainMessageHandler(msgtype.P2PCHAT, P2pChatAction.Send, P2pChatAction.Receive, P2pChatAction.Response)
}
