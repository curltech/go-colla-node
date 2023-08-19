package channel

import (
	"bufio"
	"errors"
	"fmt"
	"github.com/curltech/go-colla-core/config"
	"github.com/curltech/go-colla-core/logger"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"net"
	"sync"
	"time"
)

type P2PMessage struct {
	messageType int
	data        string
}

type Channel struct {
	stream    network.Stream
	rw        *bufio.ReadWriter
	inChan    chan *P2PMessage
	outChan   chan *P2PMessage
	closeChan chan byte

	mutex    sync.Mutex // 对closeChan关闭上锁
	isClosed bool       // 防止closeChan被关闭多次

	handler func(data string)
}

/*
*
存放所有流，键值是两个节点的id
*/
var channelPool sync.Map

func GetChannel(stream network.Stream, peer *peer.AddrInfo, handler func(data string)) (*Channel, error) {
	var chanBufferSize, _ = config.GetInt("p2p.chanBufferSize", 1024)
	peerId := peer.ID.Pretty()
	ch, ok := channelPool.Load(peerId)
	if ok {
		logger.Sugar.Infof("peer:%v exist", peerId)

		return ch.(*Channel), nil
	}
	channel := &Channel{
		stream:    stream,
		rw:        bufio.NewReadWriter(bufio.NewReader(stream), bufio.NewWriter(stream)),
		inChan:    make(chan *P2PMessage, chanBufferSize),
		outChan:   make(chan *P2PMessage, chanBufferSize),
		closeChan: make(chan byte, 1),
		handler:   handler,
	}
	channelPool.Store(peerId, channel)
	// 启动读协程
	go channel.loopRead()
	// 启动写协程
	go channel.loopWrite()

	//go channel.loopHeartbeat()

	return channel, nil
}

func (channel *Channel) Read() (msg *P2PMessage, err error) {
	select {
	case msg = <-channel.inChan:
	case <-channel.closeChan:
		err = errors.New("channel is closeed")
	}
	if channel.handler != nil {
		channel.handler(msg.data)
	}
	return
}

func (channel *Channel) Write(messageType int, data string) (err error) {
	select {
	case channel.outChan <- &P2PMessage{messageType, data}:
	case <-channel.closeChan:
		err = errors.New("channel is closeed")
	}
	return
}

func (channel *Channel) Close() {
	// 线程安全，可多次调用
	channel.stream.Close()
	// 利用标记，让closeChan只关闭一次
	channel.mutex.Lock()
	defer channel.mutex.Unlock()
	if !channel.isClosed {
		close(channel.closeChan)
		channel.isClosed = true
	}
	channelPool.Delete(channel.stream.ID())
}

// 内部实现
func (channel *Channel) loopRead() {
	var (
		data string
		err  error
	)
	var readTimeout, _ = config.GetInt("p2p.readTimeout", 0)
	for {
		if channel.isClosed {
			logger.Sugar.Errorf("websocket connection:%v is closed!", channel.stream.ID())
			return
		}
		if readTimeout > 0 {
			channel.stream.SetReadDeadline(time.Now().Add(time.Millisecond * time.Duration(readTimeout)))
		} else {
			channel.stream.SetReadDeadline(time.Time{})
		}
		data, err = channel.rw.ReadString('\n')
		if err != nil {
			// 判断是不是超时
			if netErr, ok := err.(net.Error); ok {
				if netErr.Timeout() {
					logger.Sugar.Errorf("ReadMessage timeout remote: %v\n", channel.stream.ID())
				}
			}
			channel.Close()
		} else {
			//阻塞在这里，等待inChan有空闲位置
			select {
			case channel.inChan <- &P2PMessage{
				messageType: 1,
				data:        data,
			}:
			case <-channel.closeChan: // closeChan 感知 channel断开
				channel.Close()
			}
		}
	}
}

func (channel *Channel) loopWrite() {
	var (
		msg *P2PMessage
		err error
	)
	var writeTimeout, _ = config.GetInt("p2p.writeTimeout", 0)
	for {
		select {
		case msg = <-channel.outChan:
		case <-channel.closeChan:
			channel.Close()
		}
		if channel.isClosed {
			logger.Sugar.Errorf("websocket connection:%v is closed!", channel.stream.ID())
			return
		}
		if writeTimeout > 0 {
			channel.stream.SetWriteDeadline(time.Now().Add(time.Millisecond * time.Duration(writeTimeout)))
		} else {
			channel.stream.SetWriteDeadline(time.Time{})
		}
		_, err = channel.rw.WriteString(fmt.Sprintf("%s\n", msg.data))
		if err != nil {
			logger.Sugar.Errorf("Error writing to buffer")
			channel.Close()
		}
		err = channel.rw.Flush()
		if err != nil {
			logger.Sugar.Errorf("Error flushing buffer")
			channel.Close()
		}
	}
}

// 发送存活心跳
func (channel *Channel) loopHeartbeat() {
	var heartbeatInterval, _ = config.GetInt64("p2p.heartbeatInteval", 2)
	for {
		time.Sleep(time.Duration(heartbeatInterval) * time.Second)
		if err := channel.Write(1, "heartbeat from server"); err != nil {
			logger.Sugar.Errorf("heartbeat fail")
			channel.Close()
			break
		}
	}
}
