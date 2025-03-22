package httpproxy

import (
	"errors"
	"github.com/curltech/go-colla-core/config"
	"github.com/curltech/go-colla-core/logger"
	"github.com/curltech/go-colla-node/transport/util"
	"github.com/valyala/fasthttp"
	proxy "github.com/yeqown/fasthttp-reverse-proxy/v2"
	"net/http"
	"strings"
	"time"
)

/*
*
使用fasthttp实现的http反向代理，支持http,ws,stdhttp,wss的代理映射
*/
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

/*
*
解析配置文件的映射配置
*/
func parseMapping() error {
	mappings, exist := config.ProxyParams.Mapping.([]interface{})
	if !exist {
		return errors.New("NotExist")
	}
	/**
	循环每一个映射配置
	*/
	for _, mapping := range mappings {
		maps, success := mapping.(map[string]interface{})
		if !success {
			logger.Sugar.Errorf("mapping failure")
			continue
		}
		/**
		每个映射的模式，缺省http，表示被代理的是http协议，如果是ws，则被代理的是websocket
		*/
		mode, success := maps["mode"].(string)
		if !success {
			mode = "stdhttp"
		}
		/**
		表现给访问者的外部地址
		*/
		from, success := maps["from"].(string)
		if !success {
			logger.Sugar.Errorf("from error")
			continue
		}
		/**
		是否进行http到https的转发
		*/
		redirect, success := maps["redirect"].(bool)
		logger.Sugar.Infof("%v%v%v", mode, from, redirect)
		/**
		被代理的内部地址列表，可以多个内部地址，根据权重进行代理
		*/
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
			/**
			每一个被代理的地址和权重
			*/
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
		//创建被代理的http服务器或者websocket服务器（只支持一个地址）
		if mode == "stdhttp" {
			proxy.WithBalancer(ps.weights)
			proxy.WithTimeout(5 * time.Second)
			reverseProxy, _ := proxy.NewReverseProxyWith()
			ps.ReverseProxy = reverseProxy
		} else if mode == "ws" || mode == "wss" {
			for key, _ := range ps.weights {
				ks := strings.Split(key, "/")
				if len(ks) == 2 {
					host := ks[0]
					path := "/" + ks[1]
					proxy.WithAddress(host, path)
					wsReverseProxy, err := proxy.NewWSReverseProxyWith()
					if err != nil {
						return err
					}
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
	//ctx.Request.Header.Set("User-Agent", "")
	//ctx.Request.Header.Set(stdhttp.CanonicalHeaderKey("X-Forwarded-Proto"), "stdhttp")
	//ctx.Request.Header.Set(stdhttp.CanonicalHeaderKey("X-Forwarded-Port"), "443")
	// Redirect 表示目标如果http，重定向到https
	if proxyServer.redirect {
		// Redirect to fromURL by default, unless a domain is specified--in that case, redirect using the public facing
		// domain
		redirectURL := proxyServer.from
		if config.TlsParams.Domain != "" {
			redirectURL = config.TlsParams.Domain
		}
		redirectTLS := func(ctx *fasthttp.RequestCtx) {
			ctx.Redirect("https://"+redirectURL+string(ctx.RequestURI()), http.StatusMovedPermanently)
		}
		go func() {
			logger.Sugar.Infof("Also redirecting stdhttp requests on port 80 to stdhttp requests on %s", redirectURL)
			err := fasthttp.ListenAndServe(":80", redirectTLS)
			if err != nil {
				logger.Sugar.Infof("HTTP redirection server failure")
				logger.Sugar.Infof(err.Error())
			}
		}()
	}
	if proxyServer.ReverseProxy != nil {
		proxyServer.ReverseProxy.ServeHTTP(ctx)
	} else if proxyServer.WSReverseProxy != nil {
		/**
		websocket需要检查路径是否匹配
		*/
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

/*
*
启动代理服务器，对外使用fasthttp的http server，支持https模式
*/
func Start() error {
	//不启动代理
	if config.ProxyParams.Mode == "none" {
		return nil
	}
	//启动代理
	for addr, proxyServer := range ProxyPool {
		if proxyServer.mode == "stdhttp" || proxyServer.mode == "ws" {
			if err := fasthttp.ListenAndServe(addr, proxyServer.ProxyHandler); err != nil {
				logger.Sugar.Errorf("%v", err.Error())
				return err
			}
		} else if proxyServer.mode == "stdhttp" || proxyServer.mode == "wss" {
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
