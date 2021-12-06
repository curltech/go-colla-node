package dht

import (
	"context"
	"errors"
	"fmt"
	"gitee.com/cristiane/go-push-sdk/push/huawei_channel"
	"gitee.com/cristiane/go-push-sdk/push/setting"
	"github.com/curltech/go-colla-core/logger"
	"github.com/curltech/go-colla-node/consensus/std"
	"github.com/curltech/go-colla-node/libp2p/global"
	"github.com/curltech/go-colla-node/libp2p/ns"
	handler1 "github.com/curltech/go-colla-node/libp2p/pipe/handler"
	"github.com/curltech/go-colla-node/p2p/chain/action"
	"github.com/curltech/go-colla-node/p2p/chain/handler"
	"github.com/curltech/go-colla-node/p2p/chain/handler/sender"
	"github.com/curltech/go-colla-node/p2p/dht/entity"
	"github.com/curltech/go-colla-node/p2p/dht/service"
	"github.com/curltech/go-colla-node/p2p/msg"
	"github.com/curltech/go-colla-node/p2p/msgtype"
	"github.com/google/uuid"
	"strings"
)

type p2pChatAction struct {
	action.BaseAction
}

var P2pChatAction p2pChatAction

/**
接收消息进行处理，返回为空则没有返回消息，否则，有返回消息
*/
func (this *p2pChatAction) Receive(chainMessage *msg.ChainMessage) (*msg.ChainMessage, error) {
	logger.Sugar.Infof("Receive %v message", this.MsgType)
	var response *msg.ChainMessage = nil

	targetPeerId := handler1.GetPeerId(chainMessage.TargetPeerId)
	key := ns.GetPeerClientKey(targetPeerId)
	peerClients, err := service.GetPeerClientService().GetLocals(key, "")
	if err != nil || len(peerClients) == 0 {
		peerClients, err = service.GetPeerClientService().GetValues(targetPeerId, "")
	}
	if err != nil {
		response = handler.Error(chainMessage.MessageType, err)
		return response, nil
	}
	if len(peerClients) == 0 {
		response = handler.Error(chainMessage.MessageType, errors.New("NUllPeerClients"))
		return response, nil
	}
	sent := false
	for _, peerClient := range peerClients {
		if peerClient.ActiveStatus == entity.ActiveStatus_Up {
			// 如果PeerClient的连接节点是自己，下一步就是最终目标，将目标会话放入消息中
			sent = true
			if global.IsMyself(peerClient.ConnectPeerId) {
				chainMessage.TargetConnectSessionId = peerClient.ConnectSessionId
				chainMessage.TargetConnectPeerId = peerClient.ConnectPeerId
				chainMessage.ConnectPeerId = chainMessage.TargetPeerId
			} else { // 否则下一步就是连接节点
				chainMessage.TargetConnectSessionId = peerClient.ConnectSessionId
				chainMessage.TargetConnectPeerId = peerClient.ConnectPeerId
				chainMessage.ConnectPeerId = peerClient.ConnectPeerId
			}
			go sender.SendCM(chainMessage)
		}
	}
	if sent == false {
		handler.Decrypt(chainMessage)
		response, _ = std.GetStdConsensus().ReceiveConsensus(chainMessage)
		// push notification
		for _, peerClient := range peerClients {
			prefixArr := strings.Split(peerClient.ClientType, "(")
			switch prefixArr[0] {
			case "PC":
				// do nothing
			case "Apple":
				pushApple(peerClient)
			case "HUAWEI":
				pushHuawei(peerClient)
			case "Xiaomi":

			case "OPPO":

			case "VIVO":

			default:
				// GCM
				// URORA
			}
		}

		return response, nil
	}

	return nil, nil
}

func pushApple(peerClient *entity.PeerClient) {
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
				Content:    getContent(peerClient.Language, peerClient.Name),
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

func pushHuawei(peerClient *entity.PeerClient) {
	huaweiClient, err := global.Global.PushRegisterClient.GetHUAWEIClient()
	if err != nil {
		logger.Sugar.Errorf("huawei GetHUAWEIClient error: %+v\n", err)
	} else {
		ctx := context.Background()
		respPush, _ := pushHuaweiSub(peerClient, huaweiClient)
		if respPush == nil || respPush.(*huawei_channel.PushMessageResponse).Code == "" || respPush.(*huawei_channel.PushMessageResponse).Code == "80200003" {
			accessTokenResp, err := huaweiClient.GetAccessToken(ctx)
			if err != nil {
				logger.Sugar.Errorf("huawei get access_token error: %+v\n", err)
			} else {
				global.Global.HuaweiAccessToken = accessTokenResp.(*huawei_channel.AccessTokenResp).AccessToken
				pushHuaweiSub(peerClient, huaweiClient)
			}
		}
	}
}

func pushHuaweiSub(peerClient *entity.PeerClient, huaweiClient setting.PushClientInterface) (interface{}, error) {
	var error error = nil
	var deviceTokens = []string{
		peerClient.DeviceToken,
	}
	msg := &setting.PushMessageRequest{
		DeviceTokens: deviceTokens,
		AccessToken:  global.Global.HuaweiAccessToken,
		Message: &setting.Message{
			BusinessId: uuid.New().String(),
			Title:      getTitle(peerClient.Language),
			SubTitle:   "",
			Content:    getContent(peerClient.Language, peerClient.Name),
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
	respPush, err := huaweiClient.PushNotice(ctx, msg)
	if err != nil {
		logger.Sugar.Errorf("huawei push error: %+v\n", err)
		error = err
	}
	logger.Sugar.Infof("huawei push response: %+v\n", respPush)
	return respPush, error
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
