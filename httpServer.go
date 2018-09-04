package florid

import (
	"crypto/md5"
	"encoding/hex"
	"github.com/ngaut/log"
	"net"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
)

import _ "net/http/pprof"

type httpServer struct {
	server *http.Server
}

var Server httpServer

func init() {
	Server = httpServer{}
}
func setResponseHeader(res http.ResponseWriter, key string, value string) {
	res.Header().Set(key, value)
}
func checkMd5Same(uriPath string, url string) bool {

	regmd5, _ := regexp.Compile(`\/F_(.*?)\/`)
	param := regmd5.FindStringSubmatch(uriPath)
	var md5Str string
	if len(param) >= 2 {
		md5Str = param[1]
	} else {
		md5Str = ""
	}

	uriArr := strings.Split(url, "?")
	if md5Str == "" || len(uriArr) < 2 {
		log.Warn("md5 check empty!")
		return false
	}
	uri := uriArr[1]
	h := md5.New()
	h.Write([]byte(uri + config.pwd))
	uriMd5 := strings.ToUpper(hex.EncodeToString(h.Sum(nil)))
	log.Info("md5check:", md5Str, uriMd5, uri)
	if md5Str == uriMd5 {
		return true
	}
	log.Warn("md5 check fail!", md5Str, uriMd5)
	return false
}
func (s *httpServer) ServeHTTP(res http.ResponseWriter, req *http.Request) {

	//backup url
	log.Info("request_url:", req.URL.String())

	err := req.ParseForm()
	if err != nil {
		log.Warn("req.ParseForm error:", err)
		setResponseHeader(res, "errorMessage", "req.ParseForm error")
		res.WriteHeader(http.StatusNotFound)
		return
	}

	//check dir
	uriPath := req.URL.Path

	regm, _ := regexp.Compile("(?i:/monitor.*?)")
	if regm.MatchString(uriPath) {
		log.Info("monitor check.")
		res.WriteHeader(http.StatusOK)
		res.Write([]byte("success"))
		return
	}

	reg, _ := regexp.Compile("(?i:/florid.*?)")
	if reg.MatchString(uriPath) {
		//check md5
		if config.md5 {
			isSame := checkMd5Same(uriPath, req.URL.String())
			if !isSame {
				setResponseHeader(res, "errorMessage", "md5 check error")
				res.WriteHeader(http.StatusNotFound)
				return
			}
		}
		ret := imageCallback(res, req)
		if ret == SUCCESS {
			setResponseHeader(res, "Content-Type", "image/jpeg")
			res.(http.Flusher).Flush()
		} else {
			log.Error("serverHTTP status:", ret, "：", retMessage[ret])
			setResponseHeader(res, "errorMessage", retMessage[ret])
			res.WriteHeader(http.StatusNotFound)
		}
		return
	}

	//404
	log.Error("page not find!")
	setResponseHeader(res, "errorMessage", "page not find!")
	res.WriteHeader(http.StatusNotFound)
}

//连接状态改变回调函数
func connState(conn net.Conn, stat http.ConnState) {
	//fmt.Println(stat)
}
func (s *httpServer) Run() {
	go func() {
		http.ListenAndServe("0.0.0.0:8082", nil)
	}()
	strPort := ":" + strconv.Itoa(config.port)
	s.server = &http.Server{
		Addr:         strPort,
		ReadTimeout:  time.Duration(config.rtime) * time.Millisecond,
		WriteTimeout: time.Duration(config.wtime) * time.Millisecond,
		Handler:      &httpServer{},
		ConnState:    connState,
	}
	err := s.server.ListenAndServe()
	if err != nil {
		log.Error("ListenAndServe port 8080 error!", err)
	}

}
