package http

import (
	"github.com/haoxingxing/OpenNG/logging"
	"github.com/haoxingxing/OpenNG/utils"
	"crypto/tls"
	"errors"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strconv"
	"strings"
	"time"

	ngtls "github.com/haoxingxing/OpenNG/tls"

	"github.com/dlclark/regexp2"
)

type Httphost struct {
	Id         string
	ServerName utils.GroupRegexp
	Proxy      *httputil.ReverseProxy
	Backend    string
}

//ng:generate def obj httpproxy
type httpproxy struct {
	hosts []*Httphost
	buf   *utils.BufferedLookup
}

func (*httpproxy) HandleHTTPInternal(*HttpCtx) Ret {
	return RequestEnd
}
func (*httpproxy) PathsInternal() []*regexp2.Regexp {
	return nil
}

// @RetVal *httpproxy proxier
//
//ng:generate def func NewHTTPProxier
func NewHTTPProxier() *httpproxy {
	hpx := &httpproxy{
		hosts: make([]*Httphost, 0),
		buf:   nil,
	}
	hpx.buf = utils.NewBufferedLookup(func(host string) interface{} {
		for _, t := range hpx.hosts {
			if t.ServerName.MatchString(host) {
				return t
			}
		}
		return nil
	})
	return hpx
}

func (h *httpproxy) HandleHTTP(ctx *HttpCtx) Ret {
	_host := h.buf.Lookup(ctx.Req.Host)
	if _host == nil {
		return Continue
	}
	host := _host.(*Httphost)

	defer func() {
		recover()
	}()
	host.Proxy.ServeHTTP(ctx.Resp, ctx.Req)
	return RequestEnd
}

var catchallexp = []*regexp2.Regexp{regexp2.MustCompile("^.*$", 0)}

func (h *httpproxy) Hosts() []*regexp2.Regexp {
	return catchallexp
}

// @RetVal []*Httphost hosts of proxy
//
//ng:generate def func httpproxy::GetHosts
func (hpx *httpproxy) GetHosts() []*Httphost {
	return hpx.hosts
}

// @Param string id id of proxy
// @RetVal error
//
//ng:generate def func httpproxy::Delete
func (hpx *httpproxy) Delete(id string) error {
	for i, v := range hpx.hosts {
		if v.Id == id {
			hpx.hosts = append(hpx.hosts[:i], hpx.hosts[i+1:]...)
			hpx.buf.Refresh()
			return nil
		}
	}
	return errors.New("not found")
}

// @Param int index index to insert
// @Param string id id of proxy
// @Param []string host list of host
// @Param string backend backend of proxy
// @OptionalParam int=0 TransportArgs::MaxConnsPerHost max connections per host
// @OptionalParam bool=false TransportArgs::InsecureSkipVerify skip verify
//
//ng:generate def func httpproxy::Insert
func (hpx *httpproxy) Insert(index int, id string, hosts []string, backend string, MaxConnsPerHost int, InsecureSkipVerify bool) error {
	buf := Httphost{
		Id:         id,
		ServerName: utils.MustCompileRegexp(ngtls.Dnsname2Regexp(hosts)),
		Backend:    backend,
	}

	var tlsc = tls.Config{}
	var tpa = &http.Transport{
		TLSClientConfig: &tlsc,
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          0,
		IdleConnTimeout:       0,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}
	tpa.MaxConnsPerHost = MaxConnsPerHost
	tlsc.InsecureSkipVerify = InsecureSkipVerify
	u, _ := url.Parse(backend)
	buf.Proxy = &httputil.ReverseProxy{
		ErrorHandler: func(rw http.ResponseWriter, r *http.Request, e error) {
			// rw.(*NgResponseWriter).Header().Add("X-Ng-Proxy-Err", strconv.Quote(e.Error()))
			http.Error(rw, "Bad Gateway\n"+strconv.Quote(e.Error()), http.StatusBadGateway)
			logging.Println("sys", "httpproxy", r.Host, "->", id, e)
		},
		Director: func(req *http.Request) {
			req.URL.Scheme = u.Scheme
			req.URL.Host = u.Host
			if _, ok := req.Header["User-Agent"]; !ok {
				req.Header.Set("User-Agent", "")
			}
			ip, _, _ := net.SplitHostPort(req.RemoteAddr)
			req.Header.Add("X-Real-IP", ip)
			req.Header.Add("X-Forwarded-For", ip)

		},
		Transport:     tpa,
		FlushInterval: 0,
	}
	hpx.hosts = insert(hpx.hosts, index, &buf)
	hpx.buf.Refresh()
	return nil
}
func insert(a []*Httphost, index int, value *Httphost) []*Httphost {
	if index < 0 {
		panic("index out of range")
	}
	if len(a) == index {
		a = append(a, value)
	} else {
		a = append(a[:index+1], a[index:]...)
		a[index] = value
	}
	return a
}

// @Param string id id of proxy
// @Param []string host list of host
// @Param string backend backend of proxy
// @OptionalParam int=0 TransportArgs::MaxConnsPerHost max connections per host
// @OptionalParam bool=false TransportArgs::InsecureSkipVerify skip verify
//
//ng:generate def func httpproxy::Add
func (hpx *httpproxy) Add(id string, hosts []string, backend string, MaxConnsPerHost int, InsecureSkipVerify bool) error {
	return hpx.Insert(len(hpx.hosts), id, hosts, backend, MaxConnsPerHost, InsecureSkipVerify)
}

//ng:generate def func httpproxy::Reset
func (hpx *httpproxy) Reset() error {
	hpx.hosts = make([]*Httphost, 0)
	hpx.buf.Refresh()
	return nil
}

var internals = map[string]*httputil.ReverseProxy{}

// @RetVal interface{} internel_proxier
//
//ng:generate def func NewInternalProxier
func NewInternalProxier() Service {
	return NewServiceHolder([]*regexp2.Regexp{}, func(ctx *HttpCtx) Ret {
		cfg := strings.Split(ctx.Req.Host, ".")[0] // https--10-1-1-6--443
		f1 := strings.Split(cfg, "--")
		target := strings.Replace(f1[1], "-", ".", -1)
		target = f1[0] + "://" + target + ":" + f1[2]

		a, ok := internals[target]

		if !ok {
			c, _ := url.Parse(target)
			a = httputil.NewSingleHostReverseProxy(c)
			if f1[0] == "https" {
				a.Transport = &http.Transport{
					TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
					DialContext: (&net.Dialer{
						Timeout:   30 * time.Second,
						KeepAlive: 30 * time.Second,
					}).DialContext,
					ForceAttemptHTTP2:     true,
					MaxIdleConns:          100,
					IdleConnTimeout:       900 * time.Second,
					TLSHandshakeTimeout:   10 * time.Second,
					ExpectContinueTimeout: 1 * time.Second}
			}
			a.ErrorHandler = func(rw http.ResponseWriter, r *http.Request, e error) {
				//rw.Header().Add("X-Ng-Proxy-Err", strconv.Quote(e.Error()))
				if rw.(*NgResponseWriter).code == 0 {
					http.Error(rw, "Bad Gateway\n"+strconv.Quote(e.Error()), http.StatusBadGateway)
				}
				logging.Println("sys", "while doing internal proxy: ", e.Error())
			}
			internals[target] = a
		}
		a.ServeHTTP(ctx.Resp, ctx.Req)
		return RequestEnd
	}, nil, nil)
}
