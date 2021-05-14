package fasthttp

/**
自己实现的基于fasthttp websocket的独立服务端，能够处理超大连接数
*/
import (
	"errors"
	"fmt"
	"github.com/curltech/go-colla-core/config"
	"github.com/curltech/go-colla-core/logger"
	"github.com/curltech/go-colla-core/util/security"
	"github.com/curltech/go-colla-node/transport/util"
	websocket "github.com/fasthttp/websocket"
	"github.com/phachon/fasthttpsession"
	"github.com/phachon/fasthttpsession/memory"
	"github.com/valyala/fasthttp"
	"io/ioutil"
	"mime"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type WebsocketMessage struct {
	messageType int
	data        []byte
}

type Connection struct {
	Session   fasthttpsession.SessionStore
	WsConnect *websocket.Conn
	inChan    chan *WebsocketMessage
	outChan   chan *WebsocketMessage
	closeChan chan byte

	mutex    sync.Mutex // 对closeChan关闭上锁
	isClosed bool       // 防止closeChan被关闭多次
}

var connectionPool sync.Map
var messageHandler func(data []byte, conn *Connection) ([]byte, error) = defaultMessageHandle

// 默认的 session 全局配置
var session = fasthttpsession.NewSession(fasthttpsession.NewDefaultConfig())

func init() {
	mode := config.ServerWebsocketParams.Mode
	if mode == "standalone" {
		go Start()
	}
}

const maxUploadSize = 100 * 1024 * 1014 // 100 MB
const uploadPath = "./tmp"

func Start() {
	err := session.SetProvider("memory", &memory.Config{})
	if err != nil {
		logger.Sugar.Infof("session error %v", err.Error())
		return
	}

	var listenAddr = config.ServerWebsocketParams.Address

	tlsmode := config.TlsParams.Mode
	if tlsmode == "cert" {
		cert := config.TlsParams.Cert
		key := config.TlsParams.Key
		err = util.FastHttpListenAndServeTLS(listenAddr, cert, key, requestHandler)
	} else {
		// 假如域名存在，使用LetsEncrypt certificates
		if config.TlsParams.Domain != "" {
			util.FastHttpLetsEncrypt(listenAddr, config.TlsParams.Domain, requestHandler)
		} else {
			err = fasthttp.ListenAndServe(listenAddr, requestHandler)
		}
	}
	if err != nil {
		logger.Sugar.Errorf("Start websocket server fail:%v", err.Error())
	}
}

func requestHandler(ctx *fasthttp.RequestCtx) {
	var websocketPath = config.ServerWebsocketParams.Path
	switch string(ctx.Path()) {
	case websocketPath:
		logger.Sugar.Infof("Start standalone websocket server %v", websocketPath)
		websocketHandler(ctx)
	case "/upload":
		uploadFileHandler(ctx)
	default:
		ctx.Error("Unsupported path", http.StatusNotFound)
	}
}

func errorf(ctx *fasthttp.RequestCtx, msg string, code int) {
	logger.Sugar.Errorf(msg+",error code:%v", code)
	ctx.Error(msg, code)
}

func uploadFileHandler(ctx *fasthttp.RequestCtx) {
	ctx.Request.Header.Set("Access-Control-Allow-Origin", "*")

	//设置内存大小
	//获取上传的文件组
	form, err := ctx.MultipartForm()
	if err != nil {
		errorf(ctx, "FILE_TOO_BIG", http.StatusBadRequest)
		return
	}
	if form != nil && form.File != nil {
		for key, fhs := range form.File {
			file, err := fhs[0].Open()
			if err != nil {
				errorf(ctx, "INVALID_FILE:"+key, fasthttp.StatusBadRequest)
				continue
			} else {
				defer file.Close()
				fileBytes, err := ioutil.ReadAll(file)
				if err != nil {
					errorf(ctx, "INVALID_FILE", fasthttp.StatusBadRequest)
					continue
				}
				filetype := http.DetectContentType(fileBytes)
				if filetype != "image/jpeg" && filetype != "image/jpg" &&
					filetype != "image/gif" && filetype != "image/png" &&
					filetype != "application/pdf" {
					errorf(ctx, "INVALID_FILE_TYPE", fasthttp.StatusBadRequest)
					continue
				}
				fileName := security.UUID()
				fileEndings, err := mime.ExtensionsByType(filetype)
				if err != nil {
					errorf(ctx, "CANT_READ_FILE_TYPE", fasthttp.StatusInternalServerError)
					continue
				}
				newPath := filepath.Join(uploadPath, fileName+fileEndings[0])
				logger.Sugar.Infof("FileType: %s, File: %s\n", filetype, newPath)
				newFile, err := os.Create(newPath)
				if err != nil {
					errorf(ctx, "CANT_WRITE_FILE", fasthttp.StatusInternalServerError)
					continue
				}
				defer newFile.Close()
				if _, err := newFile.Write(fileBytes); err != nil {
					errorf(ctx, "CANT_WRITE_FILE", fasthttp.StatusInternalServerError)
					continue
				}
			}
		}
	}

	ctx.WriteString("SUCCESS")

}

func websocketHandler(ctx *fasthttp.RequestCtx) {
	upgrader := websocket.FastHTTPUpgrader{
		CheckOrigin: func(ctx *fasthttp.RequestCtx) bool { return true },
	}
	err := upgrader.Upgrade(ctx, func(conn *websocket.Conn) {
		// start session
		sessionStore, err := session.Start(ctx)
		if err != nil {
			ctx.SetBodyString(err.Error())
			return
		}
		// 必须 defer sessionStore.save(ctx)
		defer sessionStore.Save(ctx)
		sessionStore.Set("name", "fasthttpsession")
		ctx.SetBodyString(fmt.Sprintf("fasthttpsession setted key name= %s ok", sessionStore.Get("name").(string)))

		var chanBufferSize, _ = config.GetInt("websocket.chanBufferSize", 1024)
		connection := &Connection{
			Session:   sessionStore,
			WsConnect: conn,
			inChan:    make(chan *WebsocketMessage, chanBufferSize),
			outChan:   make(chan *WebsocketMessage, chanBufferSize),
			closeChan: make(chan byte, 1),
		}
		sessionId := sessionStore.GetSessionId()
		connectionPool.Store(sessionId, connection)
		// 启动读协程
		go connection.loopRead()
		// 启动写协程
		go connection.loopWrite()

		go connection.loopHeartbeat()
	})

	if err != nil {
		if _, ok := err.(websocket.HandshakeError); ok {
			logger.Sugar.Errorf(err.Error())
		}
		return
	}

	logger.Sugar.Infof("conn done")
}

/**
主动发回信息
*/
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
注册读取原生数据的处理器，receiver.HandleChainMessage是选择之一
*/
func Regist(handler func(data []byte, conn *Connection) ([]byte, error)) {
	messageHandler = handler
}

/**
读取原生数据的处理器处理handler
*/
func defaultMessageHandle(data []byte, conn *Connection) ([]byte, error) {
	remoteAddr := conn.WsConnect.RemoteAddr()
	sessId := conn.Session.GetSessionId()
	logger.Sugar.Infof("receive remote addr:%v,sessionId:%v data:%v", remoteAddr.String(), sessId, len(data))

	return nil, nil
}

func (conn *Connection) read() (msg *WebsocketMessage, err error) {
	select {
	case msg = <-conn.inChan:
		if messageHandler != nil {
			response, err := messageHandler(msg.data, conn)
			if err != nil {
				return nil, err
			}
			if response != nil {
				conn.Write(2, response)
			}
		}
	case <-conn.closeChan:
		err = errors.New("connection is closeed")
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
	conn.WsConnect.Close()
	// 利用标记，让closeChan只关闭一次
	conn.mutex.Lock()
	defer conn.mutex.Unlock()
	if !conn.isClosed {
		close(conn.closeChan)
		conn.isClosed = true
	}
	connectionPool.Delete(conn.Session.GetSessionId())
}

// 内部实现
func (conn *Connection) loopRead() {
	var (
		messageType int
		data        []byte
		err         error
	)
	var readTimeout, _ = config.GetInt("websocket.readTimeout", 0)
	for {
		if conn.isClosed {
			logger.Sugar.Errorf("websocket connection:%v is closed!", conn.Session.GetSessionId())
			return
		}
		if readTimeout > 0 {
			conn.WsConnect.SetReadDeadline(time.Now().Add(time.Millisecond * time.Duration(readTimeout)))
		} else {
			conn.WsConnect.SetReadDeadline(time.Time{})
		}
		messageType, data, err = conn.WsConnect.ReadMessage()
		if err != nil {
			// 判断是不是超时
			if netErr, ok := err.(net.Error); ok {
				if netErr.Timeout() {
					logger.Sugar.Errorf("ReadMessage timeout remote: %v\n", conn.WsConnect.RemoteAddr())
				}
			}
			// 其他错误，如果是 1001 和 1000 就不打印日志
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
				logger.Sugar.Errorf("ReadMessage other remote:%v error: %v \n", conn.WsConnect.RemoteAddr(), err)
			}
			conn.Close()
			return
		} else {
			//阻塞在这里，等待inChan有空闲位置
			select {
			case conn.inChan <- &WebsocketMessage{
				messageType: messageType,
				data:        data,
			}:
				conn.read()
			case <-conn.closeChan: // closeChan 感知 conn断开
				conn.Close()
				return
			}
		}
	}
}

func (conn *Connection) loopWrite() {
	var (
		msg *WebsocketMessage
		err error
	)
	if conn.isClosed {
		logger.Sugar.Errorf("websocket connection:%v is closed!", conn.Session.GetSessionId())
		return
	}
	var writeTimeout, _ = config.GetInt("websocket.writeTimeout", 0)
	for {
		if conn.isClosed {
			logger.Sugar.Errorf("websocket connection:%v is closed!", conn.Session.GetSessionId())
			return
		}
		select {
		case msg = <-conn.outChan:
		case <-conn.closeChan:
			conn.Close()
			return
		}
		if msg != nil {
			if writeTimeout > 0 {
				conn.WsConnect.SetWriteDeadline(time.Now().Add(time.Millisecond * time.Duration(writeTimeout)))
			} else {
				conn.WsConnect.SetWriteDeadline(time.Time{})
			}
			err = conn.WsConnect.WriteMessage(msg.messageType, msg.data)
			if err != nil {
				// 判断是不是超时
				if netErr, ok := err.(net.Error); ok {
					if netErr.Timeout() {
						logger.Sugar.Errorf("WriteMessage timeout remote: %v\n", conn.WsConnect.RemoteAddr())
					}
				}
				// 其他错误，如果是 1001 和 1000 就不打印日志
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
					logger.Sugar.Errorf("WriteMessage other remote:%v error: %v \n", conn.WsConnect.RemoteAddr(), err)
				}
				conn.Close()
				return
			}
		}
	}
}

// 发送存活心跳
func (conn *Connection) loopHeartbeat() {
	var heartbeatInterval = config.ServerWebsocketParams.HeartbeatInteval
	if conn.isClosed {
		logger.Sugar.Errorf("websocket connection:%v is closed!", conn.Session.GetSessionId())
		return
	}
	for {
		time.Sleep(time.Duration(heartbeatInterval) * time.Second)
		if err := conn.Write(websocket.BinaryMessage, []byte("heartbeat from server")); err != nil {
			logger.Sugar.Errorf("heartbeat fail:%v", err.Error())
			conn.Close()
			break
		}
	}
}
