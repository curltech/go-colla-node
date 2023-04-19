package httpproxy

import (
	"github.com/curltech/go-colla-core/logger"
	"github.com/elazarl/goproxy"
	"net/http"
)

func StartProxy() {
	proxy := goproxy.NewProxyHttpServer()
	proxy.Verbose = true
	proxy.OnRequest().DoFunc(
		func(r *http.Request, ctx *goproxy.ProxyCtx) (*http.Request, *http.Response) {
			r.Header.Set("X-GoProxy", "yxorPoG-X")
			return r, nil
		})
	//proxy.OnRequest(goproxy.DstHostIs("www.reddit.com")).DoFunc(
	//	func(r *http.Request, ctx *goproxy.ProxyCtx) (*http.Request, *http.Response) {
	//		if h, _, _ := time.Now().Clock(); h >= 8 && h <= 17 {
	//			return r, goproxy.NewResponse(r,
	//				goproxy.ContentTypeText, http.StatusForbidden,
	//				"Don't waste your time!")
	//		}
	//		return r, nil
	//	})
	err := http.ListenAndServe(":8080", proxy)
	logger.Sugar.Errorf(err.Error())
}
