package httpproxy

import (
	"errors"
	"github.com/curltech/go-colla-core/config"
	"github.com/curltech/go-colla-core/logger"
	"github.com/curltech/go-colla-node/transport/util"
	"github.com/valyala/fasthttp"
	proxy "github.com/yeqown/fasthttp-reverse-proxy/v2"
	"strings"
	"time"
)

type ProxyServer struct {
	mode           string
	from           string
	weights        map[string]proxy.Weight
	redirect       bool
	ReverseProxy   *proxy.ReverseProxy
	WSReverseProxy *proxy.WSReverseProxy
}

var ProxyPool map[string]*ProxyServer = make(map[string]*ProxyServer, 0)

func init() {
	parseMapping()
}

func parseMapping() error {
	mappings, exist := config.ProxyParams.Mapping.([]interface{})
	if !exist {
		return errors.New("NotExist")
	}
	for _, mapping := range mappings {
		maps, success := mapping.(map[string]interface{})
		if !success {
			logger.Sugar.Errorf("mapping failure")
			continue
		}
		mode, success := maps["mode"].(string)
		if !success {
			mode = "http"
		}
		from, success := maps["from"].(string)
		if !success {
			logger.Sugar.Errorf("from error")
			continue
		}
		redirect, success := maps["redirect"].(bool)
		logger.Sugar.Infof("%v%v%v", mode, from, redirect)
		tos, success := maps["to"].([]interface{})
		if !success {
			logger.Sugar.Errorf("to error")
			continue
		}
		ps := &ProxyServer{mode: mode, from: from, redirect: redirect}
		for _, ts := range tos {
			to, success := ts.(map[string]interface{})
			if !success {
				logger.Sugar.Errorf("to error")
				continue
			}
			addr, success := to["address"].(string)
			if !success {
				logger.Sugar.Errorf("to address error")
				continue
			}
			weight, success := to["weight"].(int)
			if !success {
				weight = 100
			}
			logger.Sugar.Infof("%v%v", addr, weight)
			w := uint(weight)
			if ps.weights == nil {
				ps.weights = make(map[string]proxy.Weight, 0)
			}
			ps.weights[addr] = proxy.Weight(w)
		}
		if mode == "http" || mode == "https" {
			reverseProxy := proxy.NewReverseProxy("", proxy.WithBalancer(ps.weights), proxy.WithTimeout(5*time.Second))
			ps.ReverseProxy = reverseProxy
		} else if mode == "ws" || mode == "wss" {
			for key, _ := range ps.weights {
				ks := strings.Split(key, "/")
				if len(ks) == 2 {
					host := ks[0]
					path := "/" + ks[1]
					wsReverseProxy := proxy.NewWSReverseProxy(host, path)
					ps.WSReverseProxy = wsReverseProxy
					break
				}
			}
		}
		ProxyPool[from] = ps
	}
	return nil
}

func (proxyServer *ProxyServer) ProxyHandler(ctx *fasthttp.RequestCtx) {
	if proxyServer.ReverseProxy != nil {
		proxyServer.ReverseProxy.ServeHTTP(ctx)
	} else if proxyServer.WSReverseProxy != nil {
		path := string(ctx.Path())
		for key, _ := range proxyServer.weights {
			ks := strings.Split(key, "/")
			if len(ks) == 2 {
				p := "/" + ks[1]
				if p == path {
					proxyServer.WSReverseProxy.ServeHTTP(ctx)
				} else {
					ctx.Error("Unsupported path", fasthttp.StatusNotFound)
				}
				break
			}
		}
	}
}

func Start() error {
	//不启动代理
	if config.ProxyParams.Mode == "none" {
		return nil
	}
	//启动代理
	for addr, proxyServer := range ProxyPool {
		if proxyServer.mode == "http" || proxyServer.mode == "ws" {
			if err := fasthttp.ListenAndServe(addr, proxyServer.ProxyHandler); err != nil {
				logger.Sugar.Errorf("%v", err.Error())
				return err
			}
		} else if proxyServer.mode == "https" || proxyServer.mode == "wss" {
			if config.TlsParams.Domain != "" {
				util.FastHttpLetsEncrypt(addr, config.TlsParams.Domain, proxyServer.ProxyHandler)
			} else {
				// 没有域名，使用自己生成的证书
				if config.ProxyParams.Mode == "tls" {
					util.FastHttpListenAndServeTLS(addr, config.TlsParams.Cert, config.TlsParams.Key, proxyServer.ProxyHandler)
				}
			}
		}
	}
	return nil
}
