package main

import (
	"crypto/tls"
	"encoding/json"
	"log"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/spf13/viper"
	"github.com/willf/bloom"
)

// general config and vars
var (
	ListenAddr             string
	ListenPort             string
	ServerName             string
	SSLCertPath            string
	SSLKeyPath             string
	HTTPSReadTimeout       string
	HTTPSWriteTimeout      string
	HTTPSIdleTimeout       string
	HTTPSReadHeaderTimeout string
	FilterPath             string
	MessageStatus          string
	PasswordFoundMessage   string
	VersionStatus          string
	FileStatusTemp         []string
	FileStatus             string
	FalsePositiveRate      float64
	NumberOfElements       float64
	NumberOfHashFunctions  uint
	BloomfilterFile        *bloom.BloomFilter
	Bloomm                 float64
	ResponseHeaders        = map[string]string{}
	hex                    = "0123456789ABCDEF"
)

// StatusMessage : message for the status endpoint
type StatusMessage struct {
	Status  string `json:"status"`
	Message string `json:"message"`
	Version string `json:"version"`
	File    string `json:"file"`
}

// MessagePasswordFound : message for response when password is found
type MessagePasswordFound struct {
	Message string `json:"message"`
}

// ParseConfigServer : parse the configuration file
func ParseConfigServer() {
	if len(os.Args) == 3 && os.Args[1] == "--config" {
		workingdir, err := os.Getwd()
		if err != nil {
			log.Fatalf(err.Error())
		}
		viper.AddConfigPath(workingdir)
		viper.SetConfigName(os.Args[2])
		log.Printf("======================================================================================================")
		log.Printf("Using config: " + os.Args[2])
		log.Printf("======================================================================================================")

	} else {
		viper.AddConfigPath("./configs")
		viper.SetConfigName("blooming-password-server.conf")
		log.Printf("======================================================================================================")
		log.Printf("Using default config: " + "./configs/blooming-password-server.conf")
		log.Printf("======================================================================================================")
	}
	viper.SetConfigType("json")
	if err := viper.ReadInConfig(); err != nil {
		log.Fatalf("Error reading config file, %s", err)
	}
	ListenAddr = viper.GetString("ListenAddr")
	ListenPort = viper.GetString("ListenPort")
	ServerName = viper.GetString("ServerName")
	SSLCertPath = viper.GetString("SSLCertPath")
	SSLKeyPath = viper.GetString("SSLKeyPath")
	HTTPSReadTimeout = viper.GetString("HTTPSReadTimeout")
	HTTPSWriteTimeout = viper.GetString("HTTPSWriteTimeout")
	HTTPSIdleTimeout = viper.GetString("HTTPSIdleTimeout")
	HTTPSReadHeaderTimeout = viper.GetString("HTTPSReadHeaderTimeout")
	FilterPath = viper.GetString("FilterPath")
	MessageStatus = viper.GetString("MessageStatus")
	PasswordFoundMessage = viper.GetString("PasswordFoundMessage")
	VersionStatus = viper.GetString("VersionStatus")
	FileStatus = viper.GetString("FileStatus")
	FalsePositiveRate = viper.GetFloat64("FalsePositiveRate")
	NumberOfElements = viper.GetFloat64("NumberOfElements")
	NumberOfHashFunctions = viper.GetUint("NumberOfElements")
	ResponseHeaders = viper.GetStringMapString("ResponseHeaders")
	ResponseHeaders["Server"] = ServerName + " " + VersionStatus
	FileStatusTemp = strings.Split(FilterPath, "/")
	FileStatus = FileStatusTemp[len(FileStatusTemp)-1]
}

func hexOnly(hash string) bool {
	for _, c := range hash {
		if !strings.Contains(hex, string(c)) {
			return false
		}
	}
	return true
}

// AddHeadersToHTTPResponse : add headers from the config to the response
func AddHeadersToHTTPResponse(w http.ResponseWriter, contenttypejson bool) {
	// avoid race condition on map write
	var ResponseHeadersContentType = map[string]string{}
	if contenttypejson == true {
		ResponseHeadersContentType["Content-Type"] = "application/json; charset=UTF-8"
	} else {
		ResponseHeadersContentType["Content-Type"] = "text/plain; charset=utf-8"
	}
	for HeaderName, HeaderValue := range ResponseHeadersContentType {
		w.Header().Set(HeaderName, HeaderValue)
	}
	for HeaderName, HeaderValue := range ResponseHeaders {
		w.Header().Set(HeaderName, HeaderValue)
	}
}

// check - check the bloom filter for the hash
// return 200 if found, 400 on bad request or 418 if not found
func check(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	hash := strings.ToUpper(vars["hash"])
	if len(hash) != 16 || !hexOnly(hash) {
		AddHeadersToHTTPResponse(w, false)
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
	} else if BloomfilterFile.Test([]byte(hash)) {
		messagepasswordfound := MessagePasswordFound{
			Message: PasswordFoundMessage,
		}
		AddHeadersToHTTPResponse(w, true)
		err := json.NewEncoder(w).Encode(messagepasswordfound)
		if err != nil {
			AddHeadersToHTTPResponse(w, false)
			http.Error(w, err.Error(), 500)
			return
		}
	} else {
		AddHeadersToHTTPResponse(w, false)
		http.Error(w, http.StatusText(http.StatusTeapot), http.StatusTeapot)
	}
}

// index - have a blank index page
func index(w http.ResponseWriter, r *http.Request) {
	AddHeadersToHTTPResponse(w, false)
}

// status - display status OK json
func status(w http.ResponseWriter, r *http.Request) {
	statusmessage := StatusMessage{
		Status:  "OK",
		Message: MessageStatus,
		Version: VersionStatus,
		File:    FileStatus,
	}
	AddHeadersToHTTPResponse(w, true)
	err := json.NewEncoder(w).Encode(statusmessage)
	if err != nil {
		AddHeadersToHTTPResponse(w, false)
		http.Error(w, err.Error(), 500)
		return
	}
}

// default 404
func notFound(w http.ResponseWriter, r *http.Request) {
	AddHeadersToHTTPResponse(w, false)
	http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
}

func main() {
	// parse config
	ParseConfigServer()
	// create bloom filter
	Bloomm = math.Ceil((NumberOfElements * math.Log(FalsePositiveRate)) / math.Log(1.0/math.Pow(2.0, math.Log(2.0))))
	BloomfilterFile = bloom.New(uint(Bloomm), NumberOfHashFunctions)
	// load bloom filter
	f, err := os.Open(filepath.Clean(FilterPath))
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	bytesRead, err := BloomfilterFile.ReadFrom(f)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("Bytes read from bloom filter("+FilterPath+"): %d\n", bytesRead)
	log.Printf(ServerName + " Server started on " + ListenAddr + ":" + ListenPort)
	log.Printf("======================================================================================================")
	log.Printf("Logs bellow:")
	log.Printf("------------------------------------------------------------------------------------------------------\n\n")
	// create router & routes
	router := mux.NewRouter()
	router.HandleFunc("/", index).Schemes("https").Methods("GET")
	router.HandleFunc("/check/sha1/{hash}", check).Schemes("https").Methods("GET")
	router.HandleFunc("/check/sha1/{hash}/", check).Schemes("https").Methods("GET")
	router.HandleFunc("/status", status).Schemes("https").Methods("GET")
	router.HandleFunc("/status/", status).Schemes("https").Methods("GET")
	router.NotFoundHandler = http.HandlerFunc(notFound)
	// don't use ProxyHeaders unless running behind a proxy or a LB.
	handler := handlers.CombinedLoggingHandler(os.Stdout, router)
	// server routes over TLS
	// TLS config
	cfg := &tls.Config{
		MinVersion:               tls.VersionTLS12,
		CurvePreferences:         []tls.CurveID{tls.CurveP256, tls.X25519},
		PreferServerCipherSuites: true,
		CipherSuites: []uint16{
			tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305,
			tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,
		},
	}
	ReadTimeout, _ := time.ParseDuration(HTTPSReadTimeout)
	WriteTimeout, _ := time.ParseDuration(HTTPSWriteTimeout)
	IdleTimeout, _ := time.ParseDuration(HTTPSIdleTimeout)
	ReadHeaderTimeout, _ := time.ParseDuration(HTTPSReadHeaderTimeout)
	server := &http.Server{
		ReadTimeout:       ReadTimeout,
		WriteTimeout:      WriteTimeout,
		IdleTimeout:       IdleTimeout,
		ReadHeaderTimeout: ReadHeaderTimeout,
		TLSConfig:         cfg,
		TLSNextProto:      make(map[string]func(*http.Server, *tls.Conn, http.Handler), 0),
		Addr:              ListenAddr + ":" + ListenPort,
		Handler:           handler,
	}
	log.Fatal(server.ListenAndServeTLS(SSLCertPath, SSLKeyPath))
}
