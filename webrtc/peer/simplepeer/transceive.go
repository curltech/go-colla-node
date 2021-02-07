package simplepeer

import (
	"github.com/curltech/go-colla-core/logger"
	"github.com/pion/webrtc/v3"
	"io"
	"time"
)

const MessageSize = 15

/**
创建detach的api，其datachannel为detach
*/
func (this *SimplePeer) createDetachApi() (*webrtc.API, error) {
	// Since this behavior diverges from the WebRTC API it has to be
	// enabled using a settings engine. Mixing both detached and the
	// OnMessage DataChannel API is not supported.

	// Create a SettingEngine and enable Detach
	s := webrtc.SettingEngine{}
	s.DetachDataChannels()

	// Create an API object with the engine
	api := webrtc.NewAPI(webrtc.WithSettingEngine(s))

	return api, nil
}

func (this *SimplePeer) detachOpen() error {
	// Detach the data channel
	raw, dErr := this.dataChannel.Detach()
	if dErr != nil {
		logger.Sugar.Errorf(dErr.Error())
		return dErr
	}

	// Handle reading from the data channel
	go readLoop(raw)

	// Handle writing to the data channel
	go writeLoop(raw)

	return nil
}

// ReadLoop shows how to read from the datachannel directly
func readLoop(d io.Reader) {
	for {
		buffer := make([]byte, MessageSize)
		n, err := d.Read(buffer)
		if err != nil {
			logger.Sugar.Infof("Datachannel closed; Exit the readloop:", err)
			return
		}

		logger.Sugar.Infof("Message from DataChannel: %s\n", string(buffer[:n]))
	}
}

// WriteLoop shows how to write to the datachannel directly
func writeLoop(d io.Writer) {
	for range time.NewTicker(5 * time.Second).C {
		message := "hello,胡劲松"
		logger.Sugar.Infof("Sending %s \n", message)

		_, err := d.Write([]byte(message))
		if err != nil {
			logger.Sugar.Errorf(err.Error())
		}
	}
}
