package stdhttp

import (
	"errors"
	"github.com/curltech/go-colla-core/config"
	"github.com/curltech/go-colla-core/logger"
	session2 "github.com/curltech/go-colla-core/session"
	"github.com/curltech/go-colla-core/util/security"
	"github.com/curltech/go-colla-node/transport/util"
	"github.com/gorilla/websocket"
	"io"
	"io/ioutil"
	"mime"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"
)

/*
自己实现的基于gorilla websocket的独立服务端，能够处理超大连接数
*/
type WebsocketMessage struct {
	messageType int
	data        []byte
}

type WebsocketConnection struct {
	Session   *session2.Session
	WsConnect *websocket.Conn
	inChan    chan *WebsocketMessage
	outChan   chan *WebsocketMessage
	closeChan chan byte

	mutex    sync.Mutex // 对closeChan关闭上锁
	isClosed bool       // 防止closeChan被关闭多次
}

// WebsocketConnectionPool connectsessionId与websocketconnection的映射
var WebsocketConnectionPool = make(map[string]*WebsocketConnection)

var mutex = sync.Mutex{}

func init() {
	mode := config.ServerWebsocketParams.Mode
	if mode == "standalone" {
		go Start()
	}
}

const maxUploadSize = 100 * 1024 * 1014 // 100 MB
const uploadPath = "./tmp"

func Start() {
	var websocketPath = config.ServerWebsocketParams.Path
	var listenAddr = config.ServerWebsocketParams.Address
	http.HandleFunc("/upload", uploadFileHandler)
	http.HandleFunc("/receive", receiveHandler)
	http.HandleFunc(websocketPath, websocketHandler)
	tlsmode := config.TlsParams.Mode
	var err error
	logger.Sugar.Infof("Start standalone websocket server %v %v", listenAddr, websocketPath)
	if tlsmode == "cert" {
		cert := config.TlsParams.Cert
		key := config.TlsParams.Key
		err = util.HttpListenAndServeTLS(listenAddr, cert, key, nil)
	} else {
		// 假如域名存在，使用LetsEncrypt certificates
		if config.TlsParams.Domain != "" {
			util.HttpLetsEncrypt(listenAddr, config.TlsParams.Domain, nil)
			//util.FastHttpLetsEncrypt(listenAddr,config.TlsParams.Domain,nil)
		} else {
			err = http.ListenAndServe(listenAddr, nil)
			//err = fasthttp.ListenAndServe(listenAddr,fastWebsocketHandler)
		}
	}
	if err != nil {
		logger.Sugar.Errorf("Start websocket server fail:%v", err.Error())
	}
}

func errorf(w http.ResponseWriter, msg string, code int) {
	logger.Sugar.Errorf(msg+",error code:%v", code)
	w.Write([]byte(msg))
}

var messageHandler func(data []byte, remotePeerId string, clientId string, connectSessionId string, remoteAddr string) ([]byte, error)
var disconnectedHandler func(connectSessionId string)

/*
*
注册读取原生数据的处理器
*/
func RegistMessageHandler(handler func(data []byte, remotePeerId string, clientId string, connectSessionId string, remoteAddr string) ([]byte, error)) {
	messageHandler = handler
}

func RegistDisconnectedHandler(handler func(connectSessionId string)) {
	disconnectedHandler = handler
}

// /https协议
func receiveHandler(w http.ResponseWriter, r *http.Request) {
	defer func() { _ = r.Body.Close() }()
	sessionManager := session2.GetDefault()
	session := sessionManager.Start(w, r)
	sessId := session.SessionID()
	remoteAddr := r.RemoteAddr
	logger.Sugar.Infof(sessId + remoteAddr)
	body, _ := io.ReadAll(r.Body)
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE, UPDATE")
	w.Header().Set("Access-Control-Allow-Headers", "Origin, X-Requested-With, X-Extra-Header, Content-Type, Accept, Authorization")
	w.Header().Set("Access-Control-Expose-Headers", "Content-Length, Access-Control-Allow-Origin, Access-Control-Allow-Headers, Cache-Control, Content-Language, Content-Type")
	w.Header().Set("Access-Control-Allow-Credentials", "true")
	w.Header().Set("Access-Control-Max-Age", "86400") // 可选
	data, err := messageHandler(body, "", "", sessId, remoteAddr)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
	} else {
		w.WriteHeader(http.StatusOK)
		w.Write(data)
	}
}

func uploadFileHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	r.Body = http.MaxBytesReader(w, r.Body, maxUploadSize)
	//设置内存大小
	//获取上传的文件组
	if r.MultipartForm == nil {
		err := r.ParseMultipartForm(maxUploadSize)
		if err != nil {
			errorf(w, "FILE_TOO_BIG", http.StatusBadRequest)
			return
		}
	}
	if r.MultipartForm != nil && r.MultipartForm.File != nil {
		for key, fhs := range r.MultipartForm.File {
			file, err := fhs[0].Open()
			if err != nil {
				errorf(w, "INVALID_FILE:"+key, http.StatusBadRequest)
				continue
			} else {
				defer file.Close()
				fileBytes, err := ioutil.ReadAll(file)
				if err != nil {
					errorf(w, "INVALID_FILE", http.StatusBadRequest)
					continue
				}
				filetype := http.DetectContentType(fileBytes)
				if filetype != "image/jpeg" && filetype != "image/jpg" &&
					filetype != "image/gif" && filetype != "image/png" &&
					filetype != "application/pdf" {
					errorf(w, "INVALID_FILE_TYPE", http.StatusBadRequest)
					continue
				}
				fileName := security.UUID()
				fileEndings, err := mime.ExtensionsByType(filetype)
				if err != nil {
					errorf(w, "CANT_READ_FILE_TYPE", http.StatusInternalServerError)
					continue
				}
				newPath := filepath.Join(uploadPath, fileName+fileEndings[0])
				logger.Sugar.Infof("FileType: %s, File: %s\n", filetype, newPath)
				newFile, err := os.Create(newPath)
				if err != nil {
					errorf(w, "CANT_WRITE_FILE", http.StatusInternalServerError)
					continue
				}
				defer newFile.Close()
				if _, err := newFile.Write(fileBytes); err != nil {
					errorf(w, "CANT_WRITE_FILE", http.StatusInternalServerError)
					continue
				}
			}
		}
	}

	w.Write([]byte("SUCCESS"))
}

func websocketHandler(w http.ResponseWriter, r *http.Request) {
	var readBufferSize = config.ServerWebsocketParams.WriteBufferSize
	var writeBufferSize = config.ServerWebsocketParams.ReadBufferSize
	upgrade := &websocket.Upgrader{
		ReadBufferSize:  readBufferSize,
		WriteBufferSize: writeBufferSize,
		CheckOrigin: func(r *http.Request) bool {
			if r.Method != "POST" && r.Method != "GET" {
				logger.Sugar.Errorf("method is not POST or GET")
				return false
			}
			var websocketPath = config.ServerWebsocketParams.Path
			if r.URL.Path != websocketPath {
				logger.Sugar.Errorf("path error")
				return false
			}
			return true
		},
	}
	conn, err := upgrade.Upgrade(w, r, nil)
	if err != nil {
		logger.Sugar.Errorf("websocket error:", err)
		return
	}
	sessionManager := session2.GetDefault()
	session := sessionManager.Start(w, r)
	var chanBufferSize, _ = config.GetInt("websocket.chanBufferSize", 1024)
	connection := &WebsocketConnection{
		Session:   session,
		WsConnect: conn,
		inChan:    make(chan *WebsocketMessage, chanBufferSize),
		outChan:   make(chan *WebsocketMessage, chanBufferSize),
		closeChan: make(chan byte, 1),
	}
	//建立会话与连接之间的映射
	sessionId := session.SessionID()
	mutex.Lock()
	defer mutex.Unlock()
	WebsocketConnectionPool[sessionId] = connection
	// 启动读协程
	go connection.loopRead()
	// 启动写协程
	go connection.loopWrite()

	go connection.loopHeartbeat()

	return
}

/*
*
主动发回信息，也是写入写出管道
*/
func SendRaw(sessionId string, data []byte) error {
	conn, ok := WebsocketConnectionPool[sessionId]
	if !ok {
		return errors.New("NoConnection")
	}
	err := conn.Write(websocket.BinaryMessage, data)
	if err != nil {
		return errors.New("WriteFail")
	}

	return nil
}

func (conn *WebsocketConnection) read() (msg *WebsocketMessage, err error) {
	select {
	case msg = <-conn.inChan:
		if messageHandler != nil {
			response, err := messageHandler(msg.data, "", "", conn.Session.SessionID(), conn.WsConnect.RemoteAddr().String())
			if err != nil {
				return nil, err
			}
			if response != nil {
				conn.Write(websocket.BinaryMessage, response)
			}
		}
	case <-conn.closeChan:
		logger.Sugar.Errorf("connection is closeed")
		conn.Close()
	}

	return
}

// 数据写入输出管道，然后通过无限循环的写出操作写出数据
func (conn *WebsocketConnection) Write(messageType int, data []byte) (err error) {
	select {
	case conn.outChan <- &WebsocketMessage{messageType, data}:
	case <-conn.closeChan:
		///前端关闭连接时会调用
		err = errors.New("connection is closeed")
	}
	return
}

func (conn *WebsocketConnection) Close() {
	// 线程安全，可多次调用
	conn.WsConnect.Close()
	// 利用标记，让closeChan只关闭一次
	conn.mutex.Lock()
	defer conn.mutex.Unlock()
	if !conn.isClosed {
		close(conn.closeChan)
		conn.isClosed = true
	}
	mutex.Lock()
	defer mutex.Unlock()
	delete(WebsocketConnectionPool, conn.Session.SessionID())
	disconnectedHandler(conn.Session.SessionID())
}

// 内部实现
func (conn *WebsocketConnection) loopRead() {
	var (
		messageType int
		data        []byte
		err         error
	)
	var readTimeout, _ = config.GetInt("websocket.readTimeout", 0)
	for {
		if conn.isClosed {
			logger.Sugar.Errorf("websocket connection:%v is closed!", conn.Session.SessionID())
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

// 无限循环的管道写出操作
func (conn *WebsocketConnection) loopWrite() {
	var (
		msg *WebsocketMessage
		err error
	)
	if conn.isClosed {
		logger.Sugar.Errorf("websocket connection:%v is closed!", conn.Session.SessionID())
		return
	}
	var writeTimeout, _ = config.GetInt("websocket.writeTimeout", 0)
	for {
		if conn.isClosed {
			logger.Sugar.Errorf("websocket connection:%v is closed!", conn.Session.SessionID())
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
func (conn *WebsocketConnection) loopHeartbeat() {
	var heartbeatInterval = config.ServerWebsocketParams.HeartbeatInteval
	if conn.isClosed {
		logger.Sugar.Errorf("websocket connection:%v is closed!", conn.Session.SessionID())
		return
	}
	var sessionId = conn.Session.SessionID()
	err := conn.Write(websocket.BinaryMessage, []byte("heartbeat:"+sessionId))
	if err != nil {
		logger.Sugar.Errorf("heartbeat fail:%v", err.Error())
		conn.Close()
	}
	for {
		time.Sleep(time.Duration(heartbeatInterval) * time.Second)
		var sessionId = conn.Session.SessionID()
		err := conn.Write(websocket.BinaryMessage, []byte("heartbeat:"+sessionId))
		if err != nil {
			logger.Sugar.Errorf("heartbeat fail:%v", err.Error())
			conn.Close()
			break
		}
	}
}
