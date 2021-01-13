package http

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/url"
	"time"
)

var timeout = time.Duration(20 * time.Second)

func dialTimeout(ctx context.Context, network, addr string) (net.Conn, error) {
	return net.DialTimeout(network, addr, timeout)
}

// 设置http client的参数
var tr = &http.Transport{
	//使用带超时的连接函数
	DialContext: dialTimeout,
	// DialTLSContext: dialTimeout,
	//建立连接后读超时
	ResponseHeaderTimeout: time.Second * 2,
	TLSClientConfig:       &tls.Config{InsecureSkipVerify: true},
}

func Get(urlpath string, params url.Values) (string, error) {
	client := &http.Client{
		Transport: tr,
		//总超时，包含连接读写
		Timeout: timeout,
	}
	url, _ := url.Parse(urlpath)
	//如果参数中有中文参数,这个方法会进行URLEncode
	url.RawQuery = params.Encode()
	uri := url.String()
	req, err := http.NewRequest("GET", uri, nil)
	//req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := client.Do(req)
	if err != nil {
		return string(""), err
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return string(""), err
	}

	return string(body), nil
}

func Post(urlpath string, data interface{}, params url.Values) (interface{}, error) {
	client := &http.Client{
		Transport: tr,
		//总超时，包含连接读写
		Timeout: timeout,
	}
	json, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	url, _ := url.Parse(urlpath)
	//如果参数中有中文参数,这个方法会进行URLEncode
	url.RawQuery = params.Encode()
	uri := url.String()
	req, err := http.NewRequest("POST", uri, bytes.NewReader(json))
	//req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()
	// 获取服务器端读到的数据
	log.Printf("Status = ", resp.Status)         // 状态
	log.Printf("StatusCode = ", resp.StatusCode) // 状态码
	log.Printf("Header = ", resp.Header)         // 响应头部
	log.Printf("Body = ", resp.Body)             // 响应包体
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return string(body), nil
}
