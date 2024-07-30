// Copyright 2021 Square, Inc.

package main

import (
	"fmt"
        "nokia.com/square/debugassist/debugLogger"
        "nokia.com/square/debugassist/commonData"
        "nokia.com/square/debugassist/debugLogCollection"
        "nokia.com/square/debugassist/debugForGet"
	"os"
        "os/signal"
	"path/filepath"
	"sync/atomic"
        "syscall"
        "net/http"
        "io/ioutil"
        "crypto/tls"
        "crypto/x509"
        "time"
        "runtime"
        "strings"
        "encoding/json"
)
const KEY_SERVICE = "service"
const KEY_LABEL = "label"
const POD_ID = "podid"
const KEY_SPLIT_STR = ","
const TIME_LAYOUT = "20060102150405" //default format in go

var templateRequestString string
var getTemplateStatus = false
var processingFlag int32

func runDebugRequestForPost(commandsData []byte, typeData commonData.GetType, w http.ResponseWriter)(){
        debugLogger.Log.Info("runDebugRequestForPost starting ...")
	checkAndRemoveExpiredSession()

        httpResponse := debugLogCollection.TriggerDebugCollection(commandsData, typeData)
        handleHttpResponseWriter(httpResponse.ResultCode, httpResponse.ResponseMessage, w)

        debugLogger.Log.Info("runDebugRequestForPost finished, resultCode: %d, message: %s", httpResponse.ResultCode, httpResponse.ResponseMessage)
        return
}

func runDebugRequestForGet(w http.ResponseWriter) {
        debugLogger.Log.Info("runDebugRequestForGet started.")
	checkAndRemoveExpiredSession()
        if getTemplateStatus {
                w.WriteHeader(http.StatusOK)
                w.Header().Set("Content-Type", "application/octet-stream")
                w.Write([]byte(templateRequestString))
                debugLogger.Log.Info("Use Generated template successed, %s", templateRequestString)
        } else {
                getType := commonData.GetType{[]string{}, []string{}, []string{}}
                debugCollectionResponse := debugForGet.GetRequestJsonFile(getType)
                if debugCollectionResponse.ResultCode == http.StatusOK {
                        w.WriteHeader(http.StatusOK)
                        w.Header().Set("Content-Type", "application/octet-stream")
                        w.Write([]byte(debugCollectionResponse.BodyData))
                        getTemplateStatus = true
                        templateRequestString = debugCollectionResponse.BodyData
                        debugLogger.Log.Info("runDebugRequestForGet successed, %s", debugCollectionResponse.BodyData)
                }else{
                        handleHttpResponseWriter(debugCollectionResponse.ResultCode, debugCollectionResponse.ResponseMessage, w)
                }
        }
        debugLogger.Log.Info("runDebugRequestForGet finished.")
        return
}

func handleHttpResponseWriter(resultCode int, data string, w http.ResponseWriter){
        if resultCode != http.StatusOK{
                debugLogger.Log.Error(data)
        }else{
                if data == "" {
                        data = "HTTP request processed successfully."
                }
                debugLogger.Log.Info(data)
        }
        w.WriteHeader(resultCode)
        w.Write([]byte(data))

}

func parseUrlQueryInfo(r *http.Request) (commonData.GetType, commonData.HttpServerResponse) {
        var typeData commonData.GetType
        var hasReqTypesFlag = false
        var httpResponseData = commonData.HttpServerResponse{http.StatusOK, "", ""}
        keysData := r.URL.Query()
        if len(keysData) > 0{
                newKeysData := make(map[string][]string, len(keysData))
                for key, value := range keysData {
                        newKeysData[strings.ToLower(key)] = value
                }
                // Set the default label as "default", will be modified by KEY_LABEL indication(if exist)
                typeData.GenerateLabel = []string{"default"}
                if services, ok := newKeysData[KEY_SERVICE]; ok {
                        hasReqTypesFlag = true
                        typeData.ServiceName = strings.Split(services[0],KEY_SPLIT_STR)
                        debugLogger.Log.Info("typeData.ServiceName is %s", typeData.ServiceName)
                }
                if labels, ok := newKeysData[KEY_LABEL]; ok {
                        hasReqTypesFlag = true
                        typeData.GenerateLabel = strings.Split(labels[0],KEY_SPLIT_STR)
                        debugLogger.Log.Info("typeData.GenerateLabel is %s", typeData.GenerateLabel)
                }
                if podids, ok := newKeysData[POD_ID]; ok {
                        hasReqTypesFlag = true
                        typeData.Podid = strings.Split(podids[0],KEY_SPLIT_STR)
                        debugLogger.Log.Info("typeData.Podid is %s", typeData.Podid)
                }
                if !hasReqTypesFlag {
                        httpResponseData.ResponseMessage = "Invalid parameters in URL"
                        httpResponseData.ResultCode = http.StatusBadRequest
                        return typeData, httpResponseData
                }
        } else {
                // Collect default log base on default label
                typeData.GenerateLabel = []string{"default"}
        }
        return typeData, httpResponseData
}

func parseHttpRequestForPost(r *http.Request)([]byte, commonData.GetType, commonData.HttpServerResponse){
        var (
                hasReqFileFlag = false
                commandsData []byte
                typeData commonData.GetType
                httpResponseData = commonData.HttpServerResponse{http.StatusOK, "", ""}
        )

        r.ParseMultipartForm(32 << 20)
        errorMsg := "Receiving file failed with error:"
        responseMsg := ""
       srcFile, _, err := r.FormFile("-f")
       if err == nil {
               readData, err2 := ioutil.ReadAll(srcFile)
               debugLogger.Log.Info("Received data is %s", readData)
               if err2 != nil {
                        responseMsg = fmt.Sprintf("%s %s", errorMsg, err2)
                        debugLogger.Log.Error(responseMsg)
               }
               if json.Valid(readData) != true {
                        responseMsg = fmt.Sprintf("Receiving file is not valid json data")
                        debugLogger.Log.Error(responseMsg)
               } else {
                      hasReqFileFlag = true
                      commandsData = readData
               }
        }
        if responseMsg != "" {
                httpResponseData.ResultCode = http.StatusNoContent
                httpResponseData.ResponseMessage = responseMsg
                return commandsData, typeData, httpResponseData
        }


        if !hasReqFileFlag {
                typeData, httpResponseData = parseUrlQueryInfo(r)
        }
        return commandsData, typeData, httpResponseData
}

func HttpHandler(w http.ResponseWriter, r *http.Request) {
        if !atomic.CompareAndSwapInt32(&processingFlag, 0, 1) {
                errorMessage := "Error: only one concurrent debugsession is supported.\n"
                handleHttpResponseWriter(http.StatusServiceUnavailable, errorMessage, w)
                r.Body.Close()
                return
        }
        defer func() {
                atomic.StoreInt32(&processingFlag, 0)
        }()

        debugLogger.Log.Info("HttpHandler method content is %s, formdata is %s, URL.Query() is %s \n", r.Method, r.Form, r.URL)
        if r.URL.Path == "/api/ssd/v1/config" {
                if r.Method == "POST" {
                        commandsData, typeData, httpResponseData := parseHttpRequestForPost(r)
                        if httpResponseData.ResultCode != http.StatusOK {
                                handleHttpResponseWriter(httpResponseData.ResultCode, httpResponseData.ResponseMessage, w)
                                debugLogger.Log.Error("HTTP post request failed with :%s", httpResponseData.ResponseMessage)
                                r.Body.Close()
                                return
                        }
                        runDebugRequestForPost(commandsData, typeData, w)
                }else if r.Method == "GET" {
                        runDebugRequestForGet(w)
                }else {
                        w.WriteHeader(http.StatusBadRequest)
                }
        //if the URL doesn't match, return 404
        } else {
                w.WriteHeader(http.StatusNotFound)
                debugLogger.Log.Error("HttpHandler invalid URL path: %s", r.URL.Path)
        }
        r.Body.Close()
        runtime.GC()
        return
}

func updateCertFilesName(){
	certsPath := commonData.GetEnvValueWithDefault("DEBUG_ASSIST_CERTS_MOUNT_PATH", "/certs/")
        cacertFileName := commonData.GetEnvValueWithDefault("DEBUG_ASSIST_CACERTS_FILE", "cacert.pem")
        certFileName := commonData.GetEnvValueWithDefault("DEBUG_ASSIST_CERTS_FILE", "cert.pem")
        keyFileName := commonData.GetEnvValueWithDefault("DEBUG_ASSIST_KEY_FILE", "key.pem")

	commonData.CertificateFiles.CACERT_FILE = fmt.Sprintf("%s%s",certsPath, cacertFileName)
	commonData.CertificateFiles.CERT_FILE = fmt.Sprintf("%s%s",certsPath, certFileName)
	commonData.CertificateFiles.KEY_FILE = fmt.Sprintf("%s%s",certsPath, keyFileName)

        debugLogger.Log.Info("Expected certificates:%s %s %s", commonData.CertificateFiles.CACERT_FILE, commonData.CertificateFiles.CERT_FILE, commonData.CertificateFiles.KEY_FILE)
	return
}

func checkCertFile() {
        httpServerTLSEnabled := os.Getenv("DEBUG_ASSIST_HTTP_SERVER_TLS")
        rceServerTLSEnabled := os.Getenv("ENABLE_CLIFRAMEWORK_TLS")
        if !(len(httpServerTLSEnabled) > 0 && httpServerTLSEnabled == "true") && !(len(rceServerTLSEnabled) > 0 && rceServerTLSEnabled == "true") {
                return
        }
        updateCertFilesName()
        for{
                _, err1 := os.Stat(commonData.CertificateFiles.CACERT_FILE)
                _, err2 := os.Stat(commonData.CertificateFiles.CERT_FILE)
                _, err3 := os.Stat(commonData.CertificateFiles.KEY_FILE)

                if err1 != nil {
                        debugLogger.Log.Error("checkCertFile read CA file failed: %s\n", err1)
                        time.Sleep(time.Second)
                        continue
                }

                if err2 != nil {
                        debugLogger.Log.Error("checkCertFile read Certificate file failed: %s\n", err2)
                        time.Sleep(time.Second)
                        continue
                }

                if err3 != nil {
                        debugLogger.Log.Error("checkCertFile read key file failed: %s\n", err3)
                        time.Sleep(time.Second)
                        continue
                }

                debugLogger.Log.Info("Sucess to read CA/Certificate/Key file.\n")
                return
        }
}

func initConfigFile() (*string){
        errForPath := commonData.CheckConfigFilePath()
        if errForPath != nil {
                return errForPath
        }
        err := commonData.GetCaasCommands()
        if err != nil {
                return err
        }
        err1 := commonData.GetCommonCommands()
        if err1 != nil {
                return err1
        }
        err2 := commonData.GetProductCommands()
        if err2 != nil {
                return err2
        }
        err3 := commonData.GetRceServer()
        if err3 != nil {
                return err3
        }
        err4 := commonData.GetCommonConfig()
        if err4 != nil {
                return err4
        }
        return nil
}

func checkAndRemoveExpiredSession() {
	dirList, err := filepath.Glob("/logstore/debugassist/debugSession-*")
	if err != nil || len(dirList) == 0 {
                debugLogger.Log.Info("There are no debugsession directories under /logstore/debugassist/.\n")
		return
	}

	for _, dirName := range dirList {
		if isOlderThanDay(dirName, commonData.CommonConfigData.DelTime) {
			err := os.RemoveAll(dirName)
			if err != nil {
				debugLogger.Log.Error("Remove old debug session: %s failed: %s\n", dirName, err)
				continue
			}
			debugLogger.Log.Info("Debug session: %s is over than %d seconds, remove it.\n", dirName, commonData.CommonConfigData.DelTime)
		}
	}
	return
}

func isOlderThanDay(dirName string, timeLimit int64) bool {
	sessionTime, _ := time.Parse(TIME_LAYOUT, dirName[len("/logstore/debugassist/debugSession-"):])
	currentTime, _ := time.Parse(TIME_LAYOUT, time.Now().Format(TIME_LAYOUT))
	duration := currentTime.Sub(sessionTime).Seconds()
	return int64(duration) > timeLimit
}

func SetupHttpServer() (*string){
        http.HandleFunc("/", HttpHandler)

        daemonPort := "8090"
        if len(os.Getenv("DEBUG_ASSIST_SERVER_PORT")) > 0 {
                daemonPort = os.Getenv("DEBUG_ASSIST_SERVER_PORT")
        }
        var address = fmt.Sprintf(":%s", daemonPort)

        //If TLS is enabled, the http server listen with TLS required, or else http server listen without TLS required
        var err error
        isHttpServerTLSEnabled := false
        tmpValue := os.Getenv("DEBUG_ASSIST_HTTP_SERVER_TLS")
        if len(tmpValue) > 0 && tmpValue == "true" {
                isHttpServerTLSEnabled = true
        }

        if isHttpServerTLSEnabled {
                debugLogger.Log.Info("TLS enabled is true, start to load CA files.")
                pool := x509.NewCertPool()
                caCrt, err := ioutil.ReadFile(commonData.CertificateFiles.CACERT_FILE)
                if err != nil {
                        errMsg := fmt.Sprintf("read CA file failed: %s", err)
                        return &errMsg
                }
                pool.AppendCertsFromPEM(caCrt)
                server := &http.Server{
                        Addr:   address,
                        TLSConfig: &tls.Config{
                                ClientCAs:  pool,
                                ClientAuth: tls.RequireAndVerifyClientCert,
                        },
                }
                err = server.ListenAndServeTLS(commonData.CertificateFiles.CERT_FILE, commonData.CertificateFiles.KEY_FILE)
        }else {
                server := &http.Server{Addr: address}
                err = server.ListenAndServe()
        }

        if err != nil {
                errMsg := fmt.Sprintf("Start debugassist failed. err = %s", err)
                return &errMsg
        }

        debugLogger.Log.Info("Start debugassist process started successfully.")
        return nil
}

func main() {

        const MAXCPUS = 12
        num := runtime.NumCPU()
        if num > MAXCPUS {
                runtime.GOMAXPROCS(MAXCPUS)
        }

        var sigCh = make(chan os.Signal, 1)
        signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

        debugLogger.InitLogging("debugassist")

        debugLogger.Log.Info("Starting debugassist process")

        configErr := initConfigFile()
        if configErr != nil {
                debugLogger.Log.Error("debugassist not startup, error : %s", *configErr)
                time.Sleep(100 * time.Microsecond)
                return
        }
        checkCertFile()
        err := SetupHttpServer()
        if err != nil {
                debugLogger.Log.Error("debugassist not startup, error : %s", *err)
        }

        <-sigCh
}

