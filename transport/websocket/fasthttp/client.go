package fasthttp

/**
基于fasthttp websocket的客户端
*/
import (
	"github.com/curltech/go-colla-core/logger"
	"github.com/fasthttp/websocket"
	"net/url"
	"sync"
	"time"
)

type client struct {
	conn        *websocket.Conn
	schema      string
	addr        *string
	path        string
	sendMsgChan chan string
	recvMsgChan chan string
	isAlive     bool
	timeout     time.Duration
}

// 构造函数
func NewWsClient(schema string, ip string, port string, path string, timeout time.Duration) *client {
	addrString := ip + ":" + port
	var sendChan = make(chan string, 10)
	var recvChan = make(chan string, 10)
	var conn *websocket.Conn
	return &client{
		schema:      schema,
		addr:        &addrString,
		path:        path,
		conn:        conn,
		sendMsgChan: sendChan,
		recvMsgChan: recvChan,
		isAlive:     false,
		timeout:     timeout,
	}
}

// 链接服务端
func (this *client) dail() {
	var err error
	u := url.URL{Scheme: "ws", Host: *this.addr, Path: this.path}
	logger.Sugar.Infof("connecting to %s", u.String())
	this.conn, _, err = websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		logger.Sugar.Errorf(err.Error())
		return

	}
	this.isAlive = true
	logger.Sugar.Infof("connecting to %s 链接成功！！！", u.String())

}

// 发送消息
func (this *client) sendMsgThread() {
	go func() {
		for {
			msg := <-this.sendMsgChan
			err := this.conn.WriteMessage(websocket.TextMessage, []byte(msg))
			if err != nil {
				logger.Sugar.Errorf("write:", err.Error())
				continue
			}
		}
	}()
}

// 读取消息
func (this *client) readMsgThread() {
	go func() {
		for {
			if this.conn != nil {
				_, message, err := this.conn.ReadMessage()
				if err != nil {
					logger.Sugar.Errorf("read:", err.Error())
					this.isAlive = false
					// 出现错误，退出读取，尝试重连
					break
				}
				logger.Sugar.Infof("recv: %s", message)
				// 需要读取数据，不然会阻塞
				this.recvMsgChan <- string(message)
			}

		}
	}()
}

// 开启服务并重连
func (this *client) open() {
	for {
		if this.isAlive == false {
			this.dail()
			this.sendMsgThread()
			this.readMsgThread()
		}
		time.Sleep(time.Second * this.timeout)
	}
}

func (this *client) Connect(ip string, port string, path string, timeout time.Duration) {
	this = NewWsClient("wss", ip, port, path, timeout)
	this.open()
	var wg sync.WaitGroup
	wg.Add(1)
	wg.Wait()
}
