package peer

import (
	"errors"
	"github.com/curltech/go-colla-core/logger"
	"github.com/curltech/go-colla-node/webrtc"
	webrtc1 "github.com/pion/webrtc/v3"
	"sync"
)

const (
	RouterType_Mesh = "mesh"
	RouterType_Sfu  = "sfu"
	RouterType_Mcu  = "mcu"

	RouterIdentity_Publisher  = "publisher"
	RouterIdentity_Subscriber = "subscriber"
	RouterIdentity_PubSub     = "pubsub"
)

/**
sfu是多个peer的集合，一个room可以有多个sfu，但是在一个服务器peer上只能有一个，所以roomId和sfuId相同
*/
type Sfu struct {
	// sfu编号
	id string
	// sfu的所有节点
	webrtcPeers map[string]IWebrtcPeer
	// sfu的所有发布者和对应的订阅者
	publishers map[string]map[string]IWebrtcPeer
	// sfu的订阅者清单
	subscribers map[string]IWebrtcPeer
	lock        sync.Mutex
}

/**
创建新的SFU
*/
func Create(sfuId string) *Sfu {
	sfu := &Sfu{webrtcPeers: make(map[string]IWebrtcPeer)}
	sfu.id = sfuId
	sfu.publishers = make(map[string]map[string]IWebrtcPeer)
	sfu.subscribers = make(map[string]IWebrtcPeer)

	return sfu
}

/**
加入SFU
*/
func (this *Sfu) join(webrtcPeer IWebrtcPeer) error {
	this.lock.Lock()
	defer this.lock.Unlock()
	id := webrtcPeer.Id()
	_, ok := this.webrtcPeers[id]
	if !ok {
		this.webrtcPeers[id] = webrtcPeer
		if webrtcPeer.Router().Identity == RouterIdentity_Publisher || webrtcPeer.Router().Identity == RouterIdentity_PubSub {
			err := this.publish(webrtcPeer)
			if err != nil {
				return err
			}
		}
		if webrtcPeer.Router().Identity == RouterIdentity_Subscriber || webrtcPeer.Router().Identity == RouterIdentity_PubSub {
			err := this.subscribe(webrtcPeer)
			if err != nil {
				return err
			}
		}
		this.onTrack(webrtcPeer)
	} else {
		logger.Sugar.Errorf("id:%v is exist", id)

		return errors.New("Exist")
	}

	return nil
}

func (this *Sfu) onTrack(webrtcPeer IWebrtcPeer) {
	webrtcPeer.RegistEvent(webrtc.EVENT_TRACK, func(event *PoolEvent) (interface{}, error) {
		track, ok := event.Data.(*webrtc1.TrackRemote)
		if ok {
			//err := this.publishTrack(webrtcPeer, track)
			//if err != nil {
			//	return nil, err
			//}
			c := track.Codec().RTPCodecCapability
			trackLocal, err := webrtcPeer.CreateTrack(&c, track.ID(), track.StreamID())
			if err != nil {
				logger.Sugar.Errorf(err.Error())
				return nil, err
			}
			var other IWebrtcPeer = webrtcPeer
			for _, peer := range this.webrtcPeers {
				if peer.Id() != webrtcPeer.Id() {
					other = peer
					break
				}
			}
			_, err = other.AddTrack(trackLocal)
			if err != nil {
				logger.Sugar.Errorf(err.Error())
				return nil, err
			}
			for {
				// 1.读取远程数据
				packet, err := webrtcPeer.ReadRTP(track)
				if err != nil {
					return nil, err
				}
				if other == nil {
					return nil, nil
				}
				senders := other.GetSenders("")
				if senders != nil && len(senders) > 0 {
					for _, sender := range senders {
						t := sender.Track()
						if t != nil && trackLocal.StreamID() == t.StreamID() && trackLocal.ID() == t.ID() {
							go other.WriteRTP(sender, packet)
						}
					}
				}
				// go webrtcPeer.WriteRTP(sender, packet)
				//trackId := track.ID()
				//streamId := track.StreamID()
				//subscribers, ok := this.publishers[webrtcPeer.Id()]
				//if ok {
				//	for _, subscriber := range subscribers {
				//		senders := subscriber.GetSenders("")
				//		if senders != nil && len(senders) > 0 {
				//			for _, sender := range senders {
				//				if streamId == sender.Track().StreamID() && trackId == sender.Track().ID() {
				//					subscriber.WriteRTP(sender, packet)
				//				}
				//			}
				//		}
				//	}
				//}
			}
		}

		return nil, nil
	})
}

/**
离开SFU
*/
func (this *Sfu) leave(webrtcPeer IWebrtcPeer) error {
	this.lock.Lock()
	defer this.lock.Unlock()
	id := webrtcPeer.Id()
	_, ok := this.webrtcPeers[id]
	if ok {
		for _, publisher := range this.publishers {
			delete(publisher, id)
		}
		delete(this.publishers, id)
		delete(this.subscribers, id)
		delete(this.webrtcPeers, id)
	} else {
		logger.Sugar.Errorf("id:%v is not exist", id)

		return errors.New("NotExist")
	}

	return nil
}

/**
发布的意思：
产生自己的发布者列表，把已经存在的订阅者加入到自己的订阅者列表中
*/
func (this *Sfu) publish(webrtcPeer IWebrtcPeer) error {
	id := webrtcPeer.Id()
	_, ok := this.webrtcPeers[id]
	// 确保新发布的peer存在
	if ok {
		_, ok = this.publishers[id]
		// 确保还没有发布过
		if !ok {
			subscribers := make(map[string]IWebrtcPeer, 0)
			// 遍历存在的订阅列表，加入到自己的订阅者列表中
			for subscriberId, subscriber := range this.subscribers {
				if subscriberId == id {
					continue
				}
				subscribers[subscriberId] = subscriber
			}
			this.publishers[id] = subscribers
		} else {
			logger.Sugar.Errorf("id:%v is exist", id)

			return errors.New("Exist")
		}
	} else {
		logger.Sugar.Errorf("id:%v is exist", id)

		return errors.New("Exist")
	}

	return nil
}

/**
新的远程轨道转换成本地轨道，然后加到所有的订阅者上
*/
func (this *Sfu) publishTrack(webrtcPeer IWebrtcPeer, trackRemote *webrtc1.TrackRemote) error {
	this.lock.Lock()
	defer this.lock.Unlock()
	id := webrtcPeer.Id()
	_, ok := this.webrtcPeers[id]
	// 确保新发布的peer存在
	if ok {
		//subscribers, ok := this.publishers[id]
		// 确保当前节点是发布者
		if ok {
			c := trackRemote.Codec().RTPCodecCapability
			trackLocal, err := webrtcPeer.CreateTrack(&c, trackRemote.ID(), trackRemote.StreamID())
			if err != nil {
				logger.Sugar.Errorf(err.Error())
				return err
			}
			_, err = webrtcPeer.AddTrack(trackLocal)
			if err != nil {
				logger.Sugar.Errorf(err.Error())
				return err
			}
			// 遍历订阅列表，对所有的订阅节点加上新的发布节点
			//for subscriberId, subscriber := range subscribers {
			//	if subscriberId == id {
			//		continue
			//	}
			//	_, ok := this.webrtcPeers[subscriberId]
			//	// 对当前节点的所有的订阅节点加上发布节点的本地轨道
			//	if ok {
			//		_, err = subscriber.AddTrack(trackLocal)
			//		if err != nil {
			//			logger.Sugar.Errorf(err.Error())
			//			return err
			//		}
			//	}
			//}
		} else {
			logger.Sugar.Errorf("id:%v is exist", id)

			return errors.New("Exist")
		}
	} else {
		logger.Sugar.Errorf("id:%v is exist", id)

		return errors.New("Exist")
	}

	return nil
}

/**
订阅的意思：
1.将自己加入到全局订阅者列表中，加入自己到所有的发布者的订阅者列表
2.将发布者的所有远程轨道生成本地轨道，加入到自己的轨道中
*/
func (this *Sfu) subscribe(webrtcPeer IWebrtcPeer) error {
	id := webrtcPeer.Id()
	_, ok := this.webrtcPeers[id]
	// 确保新的订阅节点存在
	if ok {
		_, ok := this.subscribers[id]
		// 确保是新的订阅节点
		if !ok {
			// 加入全局订阅者列表
			this.subscribers[id] = webrtcPeer
			trackLocals := make([]webrtc1.TrackLocal, 0)
			// 遍历所以的发布者
			for publisherId, subscribers := range this.publishers {
				if publisherId == id {
					continue
				}
				pub, ok := this.webrtcPeers[publisherId]
				if ok {
					// 加入每个发布者的订阅者列表中
					_, ok = subscribers[id]
					if !ok {
						subscribers[id] = webrtcPeer
					}
					// 获取发布者的轨道，创建本地轨道
					trackRemotes := pub.GetTrackRemotes("")
					if trackRemotes != nil && len(trackRemotes) > 0 {
						for _, trackRemote := range trackRemotes {
							c := trackRemote.Codec().RTPCodecCapability
							trackLocal, err := pub.CreateTrack(&c, trackRemote.ID(), trackRemote.StreamID())
							if err != nil {
								logger.Sugar.Errorf(err.Error())
								return err
							}
							trackLocals = append(trackLocals, trackLocal)
						}
					}
				}
			}
			for _, trackLocal := range trackLocals {
				_, err := webrtcPeer.AddTrack(trackLocal)
				if err != nil {
					logger.Sugar.Errorf(err.Error())
					return err
				}
			}
		} else {
			logger.Sugar.Errorf("id:%v is exist", id)

			return errors.New("Exist")
		}
	}

	return nil

}

/**
一个服务器上的所有sfu的集合，每个sfu都属于不同的room
*/
type SfuPool struct {
	sfus map[string]*Sfu
	lock sync.Mutex
}

/**
sfu池
*/
var sfuPool *SfuPool = &SfuPool{sfus: make(map[string]*Sfu)}

func GetSfuPool() *SfuPool {
	return sfuPool
}

/**
根据roomId查找是否有sfu，如果存在就加入，不存在，则创建一个sfu并加入
*/
func (this *SfuPool) Join(webrtcPeer IWebrtcPeer) (*Sfu, error) {
	router := webrtcPeer.Router()
	if router == nil {
		return nil, errors.New("NoRouter")
	}
	if router.RoomId == "" {
		return nil, errors.New("NoRoomId")
	}
	this.lock.Lock()
	defer this.lock.Unlock()
	sfu, ok := this.sfus[router.RoomId]
	if !ok {
		sfu = Create(router.RoomId)
		this.sfus[router.RoomId] = sfu
	}
	err := sfu.join(webrtcPeer)
	if err == nil {
		logger.Sugar.Infof("sfu:%v successfully join peer:%v", router.RoomId, webrtcPeer.Id())
	}

	return sfu, err
}

func (this *SfuPool) Get(roomId string) *Sfu {
	sfu, ok := this.sfus[roomId]
	if ok {
		return sfu
	}

	return nil
}

/**
关闭IWebrtcPeer
*/
func (this *SfuPool) Leave(webrtcPeer IWebrtcPeer) error {
	router := webrtcPeer.Router()
	if router == nil {
		return errors.New("NoRouter")
	}
	if router.RoomId == "" {
		return errors.New("NoRoomId")
	}
	this.lock.Lock()
	defer this.lock.Unlock()
	sfu, ok := this.sfus[router.RoomId]
	if ok {
		err := sfu.leave(webrtcPeer)
		if err == nil {
			logger.Sugar.Infof("sfu:%v successfully left peer:%v", router.RoomId, webrtcPeer.Id())
			if len(sfu.webrtcPeers) == 0 {
				delete(this.sfus, router.RoomId)
				logger.Sugar.Infof("sfu:%v last peer left and closed", router.RoomId)
			}
		}

		return err
	} else {
		logger.Sugar.Errorf("id:%v is not exist", router.RoomId)

		return errors.New("NotExist")
	}

	return nil
}

func (this *SfuPool) Destroy(roomId string) error {
	this.lock.Lock()
	defer this.lock.Unlock()
	_, ok := this.sfus[roomId]
	if ok {
		delete(this.sfus, roomId)
		logger.Sugar.Infof("sfu:%v was destroyed", roomId)
	} else {
		logger.Sugar.Errorf("id:%v is not exist", roomId)

		return errors.New("NotExist")
	}

	return nil
}
