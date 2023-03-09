package pipe

import (
	"bufio"
	"github.com/curltech/go-colla-core/config"
	"github.com/curltech/go-colla-core/logger"
	"github.com/libp2p/go-libp2p-core/network"
	"net"
	"sync"
	"time"
)

type P2PMessage struct {
	messageType int
	data        []byte
}

type Pipe struct {
	stream  network.Stream
	handler func(data []byte, pipe *Pipe) ([]byte, error)
	direct  string
	rw      *bufio.ReadWriter
	inChan  chan []byte
	sync    bool
	mutex   sync.Mutex
}

/**
handler是读到数据时的处理器
*/
func CreatePipe(stream network.Stream, handler func(data []byte, pipe *Pipe) ([]byte, error), direct string) (*Pipe, error) {
	pipe := &Pipe{
		stream:  stream,
		handler: handler,
		direct:  direct,
	}
	// Create a buffered stream so that read and writes are non blocking.
	pipe.rw = bufio.NewReadWriter(bufio.NewReader(stream), bufio.NewWriter(stream))

	// 启动读协程
	go pipe.Read()

	return pipe, nil
}

func (pipe *Pipe) GetStream() network.Stream {
	return pipe.stream
}

func (pipe *Pipe) Close() error {
	return pipe.stream.Close()
}

func (pipe *Pipe) Reset() error {
	pipe.Close()

	return pipe.stream.Reset()
}

// 内部实现
func (pipe *Pipe) Read() []byte {
	var (
		data []byte
		err  error
	)
	var readTimeout = config.Libp2pParams.ReadTimeout
	if readTimeout > 0 {
		pipe.stream.SetReadDeadline(time.Now().Add(time.Millisecond * time.Duration(readTimeout)))
	} else {
		pipe.stream.SetReadDeadline(time.Time{})
	}
	data, err = pipe.rw.ReadBytes('\n')
	logger.Sugar.Infof("Read data length:%v", len(data))
	if err != nil {
		// 判断是不是超时
		if netErr, ok := err.(net.Error); ok {
			if netErr.Timeout() {
				logger.Sugar.Errorf("ReadMessage timeout remote: %v\n", pipe.stream.ID())
			}
		}
	} else {
		data = pipe.read(data)
		if data != nil && pipe.handler != nil {
			data, err = pipe.handler(data, pipe)
			if err != nil {
				logger.Sugar.Errorf("Error pipe.handler")
			}
		}

	}

	return data
}

func (pipe *Pipe) read(data []byte) []byte {
	pipe.mutex.Lock()
	defer pipe.mutex.Unlock()
	if pipe.sync {
		pipe.inChan <- data
		pipe.sync = false

		return nil
	}

	return data
}

func (pipe *Pipe) Write(data []byte, sync bool) (*Pipe, <-chan []byte, error) {
	var (
		err error
	)
	var writeTimeout = config.Libp2pParams.WriteTimeout
	if writeTimeout > 0 {
		pipe.stream.SetWriteDeadline(time.Now().Add(time.Millisecond * time.Duration(writeTimeout)))
	} else {
		pipe.stream.SetWriteDeadline(time.Time{})
	}
	data = append(data, '\n')
	streamId := pipe.stream.ID()
	connId := pipe.stream.Conn().ID()
	logger.Sugar.Debugf("streamId:%v, connId:%v", streamId, connId)
	_, err = pipe.stream.Write(data)
	if err != nil {
		logger.Sugar.Errorf("Error writing to buffer")

		return pipe, nil, err
	}
	err = pipe.rw.Flush()
	if err != nil {
		logger.Sugar.Errorf("Error Flush to buffer")

		return pipe, nil, err
	}
	pipe.mutex.Lock()
	defer pipe.mutex.Unlock()
	pipe.sync = sync
	if pipe.sync {
		return pipe, pipe.inChan, err
	}

	return pipe, nil, err
}
