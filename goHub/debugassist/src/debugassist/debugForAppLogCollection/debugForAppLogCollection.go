package debugForAppLogCollection

import (
	"nokia.com/square/common/rce"
	"nokia.com/square/common/rce/pb"
	"nokia.com/square/debugassist/commonData"
	"nokia.com/square/debugassist/debugLogger"
	"net/http"
	"crypto/tls"
	"fmt"
	"os"
	"sync"
	"strings"
	"time"
)

type containerConnStruct struct {
	name     string
	port     int
	commands []string
}

type hostPodPair struct {
	serviceName       string
	podName           string
	hostIp            string
	containerConnInfo []containerConnStruct
}

type ClientData struct {
	id             string
	connectionName string
	data           rce.Client
}

var mu sync.RWMutex
var storeConnData []ClientData
var commandStatus map[string]pb.Status
var doneChan chan struct{}

func putClientData(id string, connectionName string, data rce.Client) int {
	mu.Lock()
	ldata := ClientData{id: id, data: data, connectionName: connectionName}
	storeConnData = append(storeConnData, ldata)
	len := len(storeConnData)
	mu.Unlock()
	return len
}

func getClientData(i int) ClientData {
	mu.RLock()
	data := storeConnData[i]
	mu.RUnlock()
	return data
}

func getStatus(connectionName string) pb.Status {
	mu.RLock()
	value := commandStatus[connectionName]
	mu.RUnlock()
	return value
}

func putStatus(connectionName string, value pb.Status) {
	mu.Lock()
	commandStatus[connectionName] = value
	mu.Unlock()
}

func getSize() int {
	mu.RLock()
	size := len(storeConnData)
	mu.RUnlock()
	return size
}

func getContainerCommands(specCommands commonData.SpecCommandsStruct, subMap map[string]int, commonCommands []string) []containerConnStruct{
	var containerConnInfo []containerConnStruct
	for _, containerCommands := range specCommands.Containers {
		var oneContainerConnInfo containerConnStruct
		cliPort, ok := subMap[containerCommands.Name]
		if !ok {
			debugLogger.Log.Info("For", specCommands.ServiceName, "service,", containerCommands.Name, "container doesn't support debug log collection")
			continue
		}
		oneContainerConnInfo.name = containerCommands.Name
		oneContainerConnInfo.port = cliPort
		for _, command := range containerCommands.Commands {
			if strings.TrimSpace(command) == "commonCommands" {
				oneContainerConnInfo.commands = append(oneContainerConnInfo.commands, commonCommands...)
			} else {
				oneContainerConnInfo.commands = append(oneContainerConnInfo.commands, command)
			}
		}
		if len(oneContainerConnInfo.commands) > 0 {
			containerConnInfo = append(containerConnInfo, oneContainerConnInfo)
		}
	}

	return containerConnInfo
}

func getConnInfoForCli(requestBody commonData.TemplateStruct, cnfInfoData []commonData.PodStatusData, portMapInfo map[string]interface{}, serviceToPodMultiMap commonData.Multimap, specServiceToPodMultiMap commonData.Multimap, PodIpMap map[string]string) ([]hostPodPair, int) {
	connInfo := make([]hostPodPair, 0)
	commonCommands := requestBody.CommonCommands
	numEntry := 0
	debugLogger.Log.Info("serviceToPodMultiMap is %s", serviceToPodMultiMap)
	debugLogger.Log.Info("specServiceToPodMultiMap is %s", specServiceToPodMultiMap)
	debugLogger.Log.Info("PodIpMap is %s", PodIpMap)

	for _, specCommands := range requestBody.SpecCommands {
		if len(requestBody.Services) > 0 && !commonData.IsContain(requestBody.Services, "all") && !commonData.IsContain(requestBody.Services, specCommands.ServiceName) {
			debugLogger.Log.Info("%s service doesn't in configured service list. Move on to the next service.", specCommands.ServiceName)
			continue
		}
		containerToPortMap, ok := portMapInfo[specCommands.ServiceName]
		if !ok {
			debugLogger.Log.Info(specCommands.ServiceName, " service doesn't support debug log collection. Move on to the next service.")
			continue
		}
		subMap := containerToPortMap.(map[string]int)
		containerConnInfo := getContainerCommands(specCommands, subMap, commonCommands)
		if containerConnInfo == nil {
			debugLogger.Log.Info("For", specCommands.ServiceName, "service ,none of the containers defined in the log collection request are available. Move on to the next service.")
			continue
		}
		podidList := serviceToPodMultiMap[specCommands.ServiceName] // Send debug request to all pods that belong to this service
		specPodidList, ok := specServiceToPodMultiMap[specCommands.ServiceName]
		if ok { // If user specify podid via http request, then only send debug request to specific pods
			podidList = specPodidList
		}
		for _, podName := range podidList {
			var oneConnInfo hostPodPair
			oneConnInfo.serviceName = specCommands.ServiceName
			oneConnInfo.podName = podName
			oneConnInfo.hostIp = PodIpMap[podName]
			if oneConnInfo.hostIp == "" {
				debugLogger.Log.Info(oneConnInfo.podName, " pod doesn't have a valid IP address!")
				continue
			}
			oneConnInfo.containerConnInfo = containerConnInfo
			numEntry += len(containerConnInfo)
			connInfo = append(connInfo, oneConnInfo)
		}
	}

	return connInfo, numEntry
}

func connHandler(wg *sync.WaitGroup, hostpodInfo hostPodPair, containerConnInfo containerConnStruct, tlsConfig *tls.Config, SessionName string, rceServerConnFailures *[]string) {
	defer wg.Done()
	cmd := "debug" // remote whitelist command
	var args []string

	if hostpodInfo.hostIp == "" {
		return
	}
	portString := fmt.Sprintf("%d", containerConnInfo.port)
	debugLogger.Log.Info("Try to connect RCE server, pod is %s, ip is %s, port is %s.", hostpodInfo.podName, hostpodInfo.hostIp, portString)
	client := rce.NewClient(tlsConfig)
	if err := client.Open(hostpodInfo.hostIp, portString); err != nil {
		debugLogger.Log.Error("Cannot conn to rce server, pod is %s, ip is %s, port is %s, with error is: %s", hostpodInfo.podName, hostpodInfo.hostIp, portString, err)
		*rceServerConnFailures = append(*rceServerConnFailures, hostpodInfo.podName)
		return
	}
	defer client.Close() // *** Remember to close the client connection! ***

	// Start remote command
	args = append(args, SessionName)
	args = append(args, hostpodInfo.serviceName)
	args = append(args, containerConnInfo.name)
	args = append(args, containerConnInfo.commands...)
	debugLogger.Log.Info("Send rce commands to pod, pod is %s, ip is %s, port is %s, agrs is %s.", hostpodInfo.podName, hostpodInfo.hostIp, portString, args)

	id, err := client.Start(cmd, args)
	if err != nil {
		debugLogger.Log.Error("Failed to send rce commands to pod, pod is %s, ip is %s, port is %s, agrs is %s, with error is: %s", hostpodInfo.podName, hostpodInfo.hostIp, portString, args, err)
		*rceServerConnFailures = append(*rceServerConnFailures, hostpodInfo.podName)
		return
	}

	connectionName := hostpodInfo.podName + "-" + containerConnInfo.name
	putClientData(id, connectionName, client)
	// ----------------------------------------------------------------------
	// Wait for remote command
	// ----------------------------------------------------------------------
	// In the simplest case, we could call client.Wait(id) (below) and block
	// until the command finishes. But for this example we do something more
	// realistic: we presume the command might take a little while, so we call
	// client.GetStatus(id) every 2 seconds. If the command takes <2s, then
	// this loop does nothing. But if the command takes >2s, then the loop
	// streams the STDOUT and STDERR of the command.

	var finalStatus *pb.Status
	var finalErr error
	finalStatus, finalErr = client.Wait(id) // block waiting for command to finish
	if finalErr != nil {
	        debugLogger.Log.Error("client.Wait: %s for %s", finalErr, hostpodInfo.podName)
	        return
	}
	putStatus(connectionName, *finalStatus)
	debugLogger.Log.Info("Got the status for pod %s\n", connectionName)
}

func loadTls() (*tls.Config, error) {

        tlsFiles := rce.TLSFiles{
                CACert: commonData.CertificateFiles.CACERT_FILE,
                Cert:   commonData.CertificateFiles.CERT_FILE,
                Key:    commonData.CertificateFiles.KEY_FILE,
        }
        tlsConfig, err := tlsFiles.TLSConfig()

        return tlsConfig,err
}

func getCommandStatuses() {
	size := getSize()
	for i := 0; i < size; i++ {
		connData := getClientData(i)
		value := getStatus(connData.connectionName)
		if value.State == pb.STATE_COMPLETE {
			continue
		}
		status, err := connData.data.GetStatus(connData.id)
		if err != nil {
			debugLogger.Log.Info("Failed to get status for connection:%s with error: %s", connData.connectionName, err)
			continue
		}
		debugLogger.Log.Info("Put the status for connection: %s", connData.connectionName)
		putStatus(connData.connectionName, *status)
	}
}

func ConvertTimestampToRFC3339(timestamp int64) string {
	t := time.Unix(timestamp/1e9, timestamp%1e9)
	return t.Format(time.RFC3339)
}

func processCommandStatuses() string {
	var appBuilder strings.Builder

	appBuilder.WriteString(fmt.Sprintf("Number of pod: %d\n", len(storeConnData)))
	for connectionName, fstatus := range commandStatus {
		debugLogger.Log.Info("Write summary logs for connection: %s", connectionName)
		lnfmt := "%13s: %v\n"
		appBuilder.WriteString(fmt.Sprintf(lnfmt, "connectionName", connectionName))
		appBuilder.WriteString(fmt.Sprintf(lnfmt, "ID", fstatus.ID))
		appBuilder.WriteString(fmt.Sprintf(lnfmt, "Name", fstatus.Name))
		appBuilder.WriteString(fmt.Sprintf(lnfmt, "State", fstatus.State))
		appBuilder.WriteString(fmt.Sprintf(lnfmt, "PID", fstatus.PID))
		appBuilder.WriteString(fmt.Sprintf(lnfmt, "StartTime", ConvertTimestampToRFC3339(fstatus.StartTime)))
		appBuilder.WriteString(fmt.Sprintf(lnfmt, "StopTime", ConvertTimestampToRFC3339(fstatus.StopTime)))
		appBuilder.WriteString(fmt.Sprintf(lnfmt, "ExitCode", fstatus.ExitCode))
		appBuilder.WriteString(fmt.Sprintf(lnfmt, "CommandOutput", ""))
		for _, line := range fstatus.Stdout {
			appBuilder.WriteString(fmt.Sprintf(lnfmt, "", line))
		}
		appBuilder.WriteString("\n")
	}

	appSummaryLogs := appBuilder.String()
	return appSummaryLogs
}

func sendDebugReqToCliserver(requestBody commonData.TemplateStruct, cnfInfoData []commonData.PodStatusData, serviceToPodMultiMap commonData.Multimap, specServiceToPodMultiMap commonData.Multimap, PodIpMap map[string]string, portMapInfo map[string]interface{}, rceServerConnFailures *[]string) string {

	wg := sync.WaitGroup{}
	connInfo, numEntry := getConnInfoForCli(requestBody, cnfInfoData, portMapInfo, serviceToPodMultiMap, specServiceToPodMultiMap, PodIpMap)
        debugLogger.Log.Info("Trigger cliserver num is %d", numEntry)
	wg.Add(numEntry)

	isInterPodTLSEnabled := false
	tmpValue := os.Getenv("ENABLE_CLIFRAMEWORK_TLS")
	if len(tmpValue) > 0 && tmpValue == "true" {
		isInterPodTLSEnabled = true
	}

	var tlsConfig *tls.Config

	if isInterPodTLSEnabled {
		tlsConfig, _ = loadTls()
	}

	commandStatus = make(map[string]pb.Status)
	storeConnData = make([]ClientData, 0)

	for _, hostpodInfo := range connInfo {
		for _, containerConnInfo := range hostpodInfo.containerConnInfo {
			go connHandler(&wg, hostpodInfo, containerConnInfo, tlsConfig, requestBody.SessionName, rceServerConnFailures)
		}
	}

	doneChan = make(chan struct{})
	go func() {
		ticker := time.NewTicker(2 * time.Second)
		for {
			select {
			case <-doneChan:
				return
			case <-ticker.C:
				getCommandStatuses()
			}
		}
	}()

	wg.Wait()
	close(doneChan)
	debugLogger.Log.Info("All log collection requests have sent to cliserver")

	responseMsg := processCommandStatuses()
	return responseMsg
}

func ExeDebugLogCollection(requestBody commonData.TemplateStruct, cnfInfoData []commonData.PodStatusData, serviceToPodMultiMap commonData.Multimap, specServiceToPodMultiMap commonData.Multimap, PodIpMap map[string]string, debugCollectionResponse chan commonData.HttpServerResponse) {
	var responseData = commonData.HttpServerResponse{http.StatusOK, "", ""}
	// 1.Get pod ip and cliserver listen port
	portMapInfo := commonData.GetPortMapInfo()

	// 2.Send rce message
	rceServerConnFailures := []string{}
	cliResponseMsg := sendDebugReqToCliserver(requestBody, cnfInfoData, serviceToPodMultiMap, specServiceToPodMultiMap, PodIpMap, portMapInfo, &rceServerConnFailures)
	responseData.BodyData = cliResponseMsg
	if strings.Contains(cliResponseMsg, "fail") || strings.Contains(cliResponseMsg, "Partial") {
		responseData.ResponseMessage = fmt.Sprintf("Run application commands with failure in some pods.")
		responseData.ResultCode = http.StatusPartialContent
	}

	if len(rceServerConnFailures) > 0 {
		responseData.ResponseMessage += fmt.Sprintf("Connect rce server or send rce command failures occurred in the following pods: %v", rceServerConnFailures)
		responseData.ResultCode = http.StatusPartialContent
	}

	debugCollectionResponse <- responseData
}
