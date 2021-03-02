package websocket

/**
基于gorilla websocket的独立服务端，能够处理超大连接数
*/
import (
	"errors"
	"fmt"
	"github.com/curltech/go-colla-core/config"
	"github.com/curltech/go-colla-core/logger"
	session2 "github.com/curltech/go-colla-core/session"
	"github.com/gorilla/websocket"
	"net"
	"net/http"
	"sync"
	"time"
)

type WebsocketMessage struct {
	messageType int
	data        []byte
}

type Connection struct {
	session   *session2.Session
	wsConnect *websocket.Conn
	inChan    chan *WebsocketMessage
	outChan   chan *WebsocketMessage
	closeChan chan byte

	mutex    sync.Mutex // 对closeChan关闭上锁
	isClosed bool       // 防止closeChan被关闭多次

	handler func(data []byte) ([]byte, error)
}

var connectionPool sync.Map

func init() {
	mode := config.ServerWebsocketParams.Mode
	if mode == "standalone" {
		go Start()
	}
}

func Start() {
	var websocketPath = config.ServerWebsocketParams.Path
	var listenAddr = config.ServerWebsocketParams.Address
	http.HandleFunc(websocketPath, websocketHandler)
	tlsmode := config.TlsParams.Mode
	if tlsmode == "cert" {
		cert := config.TlsParams.Cert
		key := config.TlsParams.Key
		http.ListenAndServeTLS(listenAddr, cert, key, nil)
	} else {
		_ = http.ListenAndServe(listenAddr, nil)
	}
}

func websocketHandler(w http.ResponseWriter, r *http.Request) {
	var readBufferSize = config.ServerWebsocketParams.WriteBufferSize
	var writeBufferSize = config.ServerWebsocketParams.ReadBufferSize
	upgrade := &websocket.Upgrader{
		ReadBufferSize:  readBufferSize,
		WriteBufferSize: writeBufferSize,
		CheckOrigin: func(r *http.Request) bool {
			if r.Method != "POST" {
				fmt.Println("method is not GET")
				return false
			}
			var websocketPath = config.ServerWebsocketParams.Path
			if r.URL.Path != websocketPath {
				fmt.Println("path error")
				return false
			}
			return true
		},
	}
	conn, err := upgrade.Upgrade(w, r, nil)
	if err != nil {
		fmt.Println("websocket error:", err)
		return
	}
	sessionManager := session2.GetDefault()
	session := sessionManager.Start(w, r)
	var chanBufferSize, _ = config.GetInt("websocket.chanBufferSize", 1024)
	connection := &Connection{
		session:   session,
		wsConnect: conn,
		inChan:    make(chan *WebsocketMessage, chanBufferSize),
		outChan:   make(chan *WebsocketMessage, chanBufferSize),
		closeChan: make(chan byte, 1),
	}
	sessionId := session.SessionID()
	connectionPool.Store(sessionId, conn)
	// 启动读协程
	go connection.loopRead()
	// 启动写协程
	go connection.loopWrite()

	go connection.loopHeartbeat()

	return
}

func SendRaw(sessionId string, data []byte) error {
	conn, ok := connectionPool.Load(sessionId)
	if !ok {
		return errors.New("NoConnection")
	}
	c, ok := conn.(*Connection)
	if !ok {
		return errors.New("NotConnectionType")
	}
	err := c.Write(websocket.BinaryMessage, data)
	if err != nil {
		return errors.New("WriteFail")
	}

	return nil
}

/**
注册读取数据的处理器，receiver.HandleChainMessage是合适的选择
*/
func (conn *Connection) Regist(handler func(data []byte) ([]byte, error)) {
	conn.handler = handler
}

func (conn *Connection) Read() (msg *WebsocketMessage, err error) {
	select {
	case msg = <-conn.inChan:
	case <-conn.closeChan:
		err = errors.New("connection is closeed")
	}
	if conn.handler != nil {
		response, err := conn.handler(msg.data)
		if err != nil {
			return nil, err
		}
		if response != nil {
			conn.Write(2, response)
		}
	}
	return
}

func (conn *Connection) Write(messageType int, data []byte) (err error) {
	select {
	case conn.outChan <- &WebsocketMessage{messageType, data}:
	case <-conn.closeChan:
		err = errors.New("connection is closeed")
	}
	return
}

func (conn *Connection) Close() {
	// 线程安全，可多次调用
	conn.wsConnect.Close()
	// 利用标记，让closeChan只关闭一次
	conn.mutex.Lock()
	defer conn.mutex.Unlock()
	if !conn.isClosed {
		close(conn.closeChan)
		conn.isClosed = true
	}
	connectionPool.Delete(conn.session.SessionID())
}

// 内部实现
func (conn *Connection) loopRead() {
	var (
		messageType int
		data        []byte
		err         error
	)
	var readTimeout, _ = config.GetInt("websocket.readTimeout", 5000)
	for {
		conn.wsConnect.SetReadDeadline(time.Now().Add(time.Millisecond * time.Duration(readTimeout)))
		messageType, data, err = conn.wsConnect.ReadMessage()
		if err != nil {
			// 判断是不是超时
			if netErr, ok := err.(net.Error); ok {
				if netErr.Timeout() {
					logger.Sugar.Errorf("ReadMessage timeout remote: %v\n", conn.wsConnect.RemoteAddr())
				}
			}
			// 其他错误，如果是 1001 和 1000 就不打印日志
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
				logger.Sugar.Errorf("ReadMessage other remote:%v error: %v \n", conn.wsConnect.RemoteAddr(), err)
			}
			conn.Close()
		} else {
			//阻塞在这里，等待inChan有空闲位置
			select {
			case conn.inChan <- &WebsocketMessage{
				messageType: messageType,
				data:        data,
			}:
			case <-conn.closeChan: // closeChan 感知 conn断开
				conn.Close()
			}
		}
	}
}

func (conn *Connection) loopWrite() {
	var (
		msg *WebsocketMessage
		err error
	)

	for {
		select {
		case msg = <-conn.outChan:
		case <-conn.closeChan:
			conn.Close()
		}
		if err = conn.wsConnect.WriteMessage(msg.messageType, msg.data); err != nil {
			conn.Close()
		}
	}
}

// 发送存活心跳
func (conn *Connection) loopHeartbeat() {
	var heartbeatInterval = config.ServerWebsocketParams.HeartbeatInteval
	for {
		time.Sleep(time.Duration(heartbeatInterval) * time.Second)
		if err := conn.Write(websocket.BinaryMessage, []byte("heartbeat from server")); err != nil {
			logger.Sugar.Errorf("heartbeat fail")
			conn.Close()
			break
		}
	}
}
