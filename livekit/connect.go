package livekit

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/livekit/protocol/auth"
	"github.com/livekit/protocol/utils"
	"gopkg.in/yaml.v3"
)

//生成访问键和密码，可以访问服务器
func generateKeys() (string, string, error) {
	apiKey := utils.NewGuid(utils.APIKeyPrefix)
	secret := utils.RandomSecret()
	fmt.Println("API Key: ", apiKey)
	fmt.Println("Secret Key: ", secret)

	return apiKey, secret, nil
}

func printPorts() error {
	conf, err := getConfig()
	if err != nil {
		return err
	}

	udpPorts := make([]string, 0)
	tcpPorts := make([]string, 0)

	tcpPorts = append(tcpPorts, fmt.Sprintf("%d - HTTP service", conf.Port))
	if conf.RTC.TCPPort != 0 {
		tcpPorts = append(tcpPorts, fmt.Sprintf("%d - ICE/TCP", conf.RTC.TCPPort))
	}
	if conf.RTC.UDPPort != 0 {
		udpPorts = append(udpPorts, fmt.Sprintf("%d - ICE/UDP", conf.RTC.UDPPort))
	} else {
		udpPorts = append(udpPorts, fmt.Sprintf("%d-%d - ICE/UDP range", conf.RTC.ICEPortRangeStart, conf.RTC.ICEPortRangeEnd))
	}

	if conf.TURN.Enabled {
		tcpPorts = append(tcpPorts, strconv.Itoa(conf.TURN.TLSPort))
	}

	fmt.Println("TCP Ports")
	for _, p := range tcpPorts {
		fmt.Println(p)
	}

	fmt.Println("UDP Ports")
	for _, p := range udpPorts {
		fmt.Println(p)
	}
	return nil
}

//根据访问键和密码，为房间和房间用户创建token
func createToken(room string, identity string) (string, error) {
	conf, err := getConfig()
	if err != nil {
		return "", err
	}

	// use the first API key from config
	if len(conf.Keys) == 0 {
		// try to load from file
		if _, err := os.Stat(conf.KeyFile); err != nil {
			return "", err
		}
		f, err := os.Open(conf.KeyFile)
		if err != nil {
			return "", err
		}
		defer func() {
			_ = f.Close()
		}()
		decoder := yaml.NewDecoder(f)
		if err = decoder.Decode(conf.Keys); err != nil {
			return "", err
		}

		if len(conf.Keys) == 0 {
			return "", fmt.Errorf("keys are not configured")
		}
	}

	var apiKey string
	var apiSecret string
	for k, v := range conf.Keys {
		apiKey = k
		apiSecret = v
		break
	}

	grant := &auth.VideoGrant{
		RoomJoin: true,
		Room:     room,
	}
	at := auth.NewAccessToken(apiKey, apiSecret).
		AddGrant(grant).
		SetIdentity(identity).
		SetValidFor(30 * 24 * time.Hour)

	token, err := at.ToJWT()
	if err != nil {
		return "", err
	}

	fmt.Println("Token:", token)

	return token, nil
}
