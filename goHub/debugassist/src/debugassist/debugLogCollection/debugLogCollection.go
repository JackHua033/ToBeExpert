package debugLogCollection

import (
	"os"
	"context"
	"io/ioutil"
        "nokia.com/square/debugassist/commonData"
        "nokia.com/square/debugassist/debugLogger"
        "nokia.com/square/debugassist/debugForAppLogCollection"
        "nokia.com/square/debugassist/debugForCaasLogCollection"
        "nokia.com/square/debugassist/debugForGet"
        "net/http"
	"encoding/json"
	"path/filepath"
	"fmt"
	"time"
	corev1 "k8s.io/api/core/v1"
)

func sendStatusOkChanData(httpResponse chan commonData.HttpServerResponse) {
        var defaultOkResponseData = commonData.HttpServerResponse{http.StatusOK, "", ""}
        httpResponse <- defaultOkResponseData
        return
}

func writeSummaryLogs(caasHttpResponse, appHttpResponse commonData.HttpServerResponse, sessionName string) string {
	summaryLogFileName := filepath.Join("/logstore/debugassist", sessionName, "Summary.log")
	err := os.MkdirAll(filepath.Dir(summaryLogFileName), 0755)
	if err != nil {
		return fmt.Sprintf("Failed to create directory for file:%s; Error: %s", summaryLogFileName, err)
	}
	summaryLogFile, err := os.OpenFile(summaryLogFileName, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Sprintf("Failed to create file: %s; Error: %s", summaryLogFileName, err)
	}
	defer summaryLogFile.Close()

	appLogEntry := fmt.Sprintf("=== Application Commands Logs ===\n\n%s\n", appHttpResponse.BodyData)
	_, err = summaryLogFile.WriteString(appLogEntry)
	if err != nil {
		return fmt.Sprintf("Failed to write application logs to file: %s. Error: %s", summaryLogFileName, err)
	}
	caasLogEntry := fmt.Sprintf("=== Caas Commands Logs ===\n\n%s\n\n%s\n", caasHttpResponse.BodyData, caasHttpResponse.ResponseMessage)
	_, err = summaryLogFile.WriteString(caasLogEntry)
	if err != nil {
		return fmt.Sprintf("Failed to write caas logs to file: %s. Error: %s", summaryLogFileName, err)
	}

	return ""
}

func combineResponse(caasHttpResponse, appHttpResponse commonData.HttpServerResponse, sessionName string) commonData.HttpServerResponse {
	var responseData commonData.HttpServerResponse
	if caasHttpResponse.ResultCode != http.StatusOK && appHttpResponse.ResultCode != http.StatusOK {
		responseData.ResultCode = http.StatusInternalServerError
		responseData.ResponseMessage = fmt.Sprintf(
			"\n  Caas commands executed failed, error code is: %d, error msg is: %s;"+
			"\n  Application commands executed failed, error code is: %d, error msg is: %s.",
			caasHttpResponse.ResultCode, caasHttpResponse.ResponseMessage,
			appHttpResponse.ResultCode, appHttpResponse.ResponseMessage)
	} else if caasHttpResponse.ResultCode != http.StatusOK && appHttpResponse.ResultCode == http.StatusOK {
		responseData.ResultCode = http.StatusPartialContent
		responseData.ResponseMessage = fmt.Sprintf(
			"\n  Caas commands executed failed, error code is: %d, error msg is: %s;"+
			"\n  Application commands executed successfully.",
			caasHttpResponse.ResultCode, caasHttpResponse.ResponseMessage)
	} else if caasHttpResponse.ResultCode == http.StatusOK && appHttpResponse.ResultCode != http.StatusOK {
		responseData.ResultCode = http.StatusPartialContent
		responseData.ResponseMessage = fmt.Sprintf(
			"\n  Application commands executed failed, error code is: %d, error msg is: %s;"+
			"\n  Caas commands executed successfully.",
			appHttpResponse.ResultCode, appHttpResponse.ResponseMessage)
	} else {
		debugLogger.Log.Info("All commands executed successfully")
		responseData.ResponseMessage = fmt.Sprintf("HTTP request processed successfully.")
		responseData.ResultCode = http.StatusOK
	}
	errMsg := writeSummaryLogs(caasHttpResponse, appHttpResponse, sessionName)
	if errMsg != "" {
		responseData.ResponseMessage += "; " + errMsg
	}
	responseData.ResponseMessage = fmt.Sprintf("%s\n  Please check the result logs with session name %s.\n",responseData.ResponseMessage, sessionName)
	return responseData
}

func GetPodsStatusData(pods corev1.PodList, cmaStatusData *[]commonData.PodStatusData) error {
	for _, pod := range pods.Items {
		var podIps []string
		for _, podIP := range pod.Status.PodIPs {
			podIps = append(podIps, podIP.IP)
		}

		var containerStatus []commonData.ContainerStatusData
		for _, container := range pod.Status.ContainerStatuses {
			containerStatus = append(containerStatus, commonData.ContainerStatusData{
				ContainerName: container.Name,
				Ready:         fmt.Sprintf("%t", container.Ready),
				State:         container.State.String(),
			})
		}

		*cmaStatusData = append(*cmaStatusData, commonData.PodStatusData{
			PodName:    pod.Name,
			Annotation: pod.Annotations,
			Labels: commonData.LabelsData{
				Name:            pod.Labels["app.kubernetes.io/name"],
				PodTemplateHash: pod.Labels["pod-template-hash"],
				ServiceType:     pod.Labels["serviceType"],
				VnfMajorRelease: pod.Labels["vnfMajorRelease"],
				VnfMinorRelease: pod.Labels["vnfMinorRelease"],
				VnfName:         pod.Labels["vnfName"],
				VnfType:         pod.Labels["vnfType"],
				VnfcType:        pod.Labels["vnfcType"],
			},
			HostIp:          pod.Status.HostIP,
			Phase:           string(pod.Status.Phase),
			PodIp:           pod.Status.PodIP,
			PodIps:          podIps,
			ContainerStatus: containerStatus,
		})
	}
	return nil
}

func GetPodInfoForTopology() ([]commonData.PodType, string) {
	var ctx	context.Context
	var cancel context.CancelFunc
	var podTypeList []commonData.PodType

	cNamespace, err := ioutil.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/namespace")
	if err != nil {
		return nil, fmt.Sprintf("Unable to read namespace file. ERR = %s", err)
	}
	_, kubeClientSet, err := commonData.GetClientset()
	ctx, cancel = context.WithTimeout(context.Background(), time.Duration(time.Millisecond*900))
	defer cancel()

	pods, err := commonData.GetPods(ctx, kubeClientSet, "All", string(cNamespace), make(commonData.Multimap))
	if err != nil {
		return nil, fmt.Sprintf("Failed to get pods: %s", err.Error())
	}

	serviceList := []string{"UDMSDM","UDMUECM","UDMUEAUTH","UDMEES","UDMMTS","UDMNIDD","UDMNIM","UDMPP","UDMSIM","UDMSIDF","UDMARPF","UDMTFR"}
	databaseList := []string{"CCAS", "CDB"}
	podTypeMap := make(map[string][]commonData.PodInfo)
	for _, pod := range pods.Items {
		var containerInfoList []commonData.ContainerInfo
		for _, container := range pod.Status.ContainerStatuses {
			var containerStatus string
			if container.State.Running != nil {
				containerStatus = "Running"
			} else if container.State.Terminated != nil {
				containerStatus = "Terminated"
			} else if container.State.Waiting != nil {
				containerStatus = "Waiting"
			} else {
				containerStatus = "Unknown"
			}
			containerInfo := commonData.ContainerInfo{
				Name:   container.Name,
				Status: containerStatus,
			}
			containerInfoList = append(containerInfoList, containerInfo)
		}

		podInfo := commonData.PodInfo{
			Name:        pod.Name,
			CreateTime:  pod.CreationTimestamp.Time,
			Containers:  containerInfoList,
		}
		serviceType := pod.ObjectMeta.Labels["serviceType"]
		podType := "Auxiliary"
		if commonData.IsTargetStringInList(serviceList, serviceType) {
			podType = "3GPP Services"
		} else if commonData.IsTargetStringInList(databaseList, serviceType) {
			podType = "State Data Storage"
		}
		podTypeMap[podType] = append(podTypeMap[podType], podInfo)
	}
	for podType, podInfoList := range podTypeMap {
		podData := commonData.PodType{
			Type:    podType,
			PodList: podInfoList,
		}
		podTypeList = append(podTypeList, podData)
	}


	return podTypeList, ""
}

func GetCnfInfoFromCma() ([]commonData.PodStatusData, string) {
        errMsg := ""
	var ctx	context.Context
	var cancel context.CancelFunc
	var cmaStatusData []commonData.PodStatusData

	const errFormat = "ERR = %s"

	cNamespace, err := ioutil.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/namespace")
	if err != nil {
		return nil, fmt.Sprintf("Unable to read namespace file. ERR = %s", err)
	}
	_, kubeClientSet, err := commonData.GetClientset()
	ctx, cancel = context.WithTimeout(context.Background(), time.Duration(time.Millisecond*900))
	defer cancel()

	pods, err := commonData.GetPods(ctx, kubeClientSet, "All", string(cNamespace), make(commonData.Multimap))
	err = GetPodsStatusData(*pods, &cmaStatusData)
	if err != nil {
		errMsg = fmt.Sprintf("Failed to get pods data. " + fmt.Sprintf(errFormat, err))
		return nil, errMsg
	}

	return cmaStatusData, errMsg
}

func getPodInfoFromCmadata(cnfInfoData []commonData.PodStatusData) (commonData.Multimap, map[string]string, map[string]string) {
	var serviceToPodMultiMap = make(commonData.Multimap)
	var PodIpMap = make(map[string]string)
        var PodNameMap = make(map[string]string)

	for _, cnfInfo := range cnfInfoData {
		if cnfInfo.Phase == "Running" {
			serviceName := cnfInfo.Labels.ServiceType
			podName := cnfInfo.PodName
			serviceToPodMultiMap.Add(serviceName, podName)
			PodIpMap[podName] = cnfInfo.PodIp
                        PodNameMap[podName] = serviceName
		}
	}

	return serviceToPodMultiMap, PodIpMap, PodNameMap
}

func addLabels(uniqueLabels map[string]bool, labels []string) {
	for _, label := range labels {
		uniqueLabels[label] = true
	}
}

func getValidLabelsFromCommands() map[string]bool {
	uniqueLabels := make(map[string]bool)

	// iterate over CaasCommandsData and add labels to uniqueLabels
	for _, caasCommand := range commonData.CaasCommandsData.CaasCommands {
		addLabels(uniqueLabels, caasCommand.Labels)
	}

	// iterate over CommonCommandsData and add labels to uniqueLabels
	for _, command := range commonData.CommonCommandsData.NativeCommands {
		addLabels(uniqueLabels, command.Labels)
	}
	for _, command := range commonData.CommonCommandsData.ScriptCommands {
		addLabels(uniqueLabels, command.Labels)
	}

	// iterate over ProductCommandsData and add labels to uniqueLabels
	for _, commands := range commonData.ProductCommandsData.NativeCommands {
		for _, container := range commands.Containers {
			for _, command := range container.Commands {
				addLabels(uniqueLabels, command.Labels)
			}
		}
	}
	for _, commands := range commonData.ProductCommandsData.ScriptCommands {
		for _, container := range commands.Containers {
			for _, command := range container.Commands {
				addLabels(uniqueLabels, command.Labels)
			}
		}
	}

	return uniqueLabels
}

func inputPodidValidation(PodNameMap map[string]string, typeData *commonData.GetType) (bool, string, commonData.Multimap) {
        var specServiceToPodMultiMap = make(commonData.Multimap)
	for _, podName := range typeData.Podid {
                serviceName, ok := PodNameMap[podName]
		if !ok {
			return false, podName, specServiceToPodMultiMap
		}
                specServiceToPodMultiMap.Add(serviceName, podName)
                if !commonData.IsContain(typeData.ServiceName, serviceName) {
                        debugLogger.Log.Info("Add service %s for pod id %s to input services.", serviceName, podName)
                        typeData.ServiceName = append(typeData.ServiceName, serviceName)
                }
	}

	return true, "", specServiceToPodMultiMap
}

func mapToString(m map[string]bool) string {
	var str string
	for key, _ := range m {
		str += key + ","
	}
	return str[:len(str)-1]
}

func validateGenerateLabels(typeData commonData.GetType) commonData.HttpServerResponse {
	var responseData commonData.HttpServerResponse
	if len(typeData.GenerateLabel) > 0 {
		var validLabels = getValidLabelsFromCommands()
		for _, label := range typeData.GenerateLabel {
			_, ok := validLabels[label]
			if !ok {
				responseData.ResultCode = http.StatusBadRequest
				str := mapToString(validLabels)
				responseData.ResponseMessage = fmt.Sprintf("The input label %s is invalid. Valid label are %s", label, str)
				return responseData
			}
		}
	}
	return responseData
}

func validateServiceNames(typeData commonData.GetType, serviceToPodMultiMap commonData.Multimap) commonData.HttpServerResponse {
	var responseData commonData.HttpServerResponse
	if len(typeData.ServiceName) > 0 {
		validServiceNames := commonData.GetServicesStringList(serviceToPodMultiMap)
		for _, service := range typeData.ServiceName {
			if !commonData.IsTargetStringInList(validServiceNames, service) {
				responseData.ResultCode = http.StatusBadRequest
				responseData.ResponseMessage = fmt.Sprintf("The input service %s is invalid. Valid service are %v", service, validServiceNames)
				return responseData
			}
		}
	}
	return responseData
}

func validatePodId(typeData *commonData.GetType, PodNameMap map[string]string) (bool, commonData.Multimap, commonData.HttpServerResponse) {
	var isPodidValid bool
	var podId string
	specServiceToPodMultiMap := make(commonData.Multimap)
	var responseData commonData.HttpServerResponse

	if len(typeData.Podid) > 0 {
		/* 1. If the input podids don't exist or don't running, return failure directly,
		   2. Update the input services info for get plugin
		   3. Generate specServiceToPodMultiMap which stores information about the input podids belong to which service
		*/
		isPodidValid, podId, specServiceToPodMultiMap = inputPodidValidation(PodNameMap, typeData)
		if !isPodidValid {
			responseData.ResultCode = http.StatusBadRequest
			responseData.ResponseMessage = fmt.Sprintf("The input pod id %s doesn't exist or is not running.", podId)
			return false, nil, responseData
		}
	}
	return true, specServiceToPodMultiMap, responseData
}

func validateInputCommands(requestBody commonData.TemplateStruct) commonData.HttpServerResponse {
	var responseData commonData.HttpServerResponse
	if len(requestBody.CaasCommands) == 0 && len(requestBody.SpecCommands) == 0 {
		responseData.ResultCode = http.StatusBadRequest
        responseData.ResponseMessage = "There is no valid caas commands or cnf application commands to execute."
		return responseData
	}
	return responseData
}

func TriggerDebugCollection(commandsData []byte, typeData commonData.GetType) commonData.HttpServerResponse {
        var responseData commonData.HttpServerResponse
        var isPodidValid bool
        specServiceToPodMultiMap := make(commonData.Multimap)
        caasDebugCollectionResponse := make(chan commonData.HttpServerResponse)
        appDebugCollectionResponse := make(chan commonData.HttpServerResponse)

	// Get podInfo for poc
	if len(typeData.ServiceName) > 0 && commonData.IsTargetStringInList(typeData.ServiceName, "topology") {
		pocInfo, pocerr := GetPodInfoForTopology()
		if pocerr != "" {
			errorJSON1, _ := json.Marshal(map[string]interface{}{
				"error": pocerr,
			})
		        responseData.ResultCode = http.StatusBadRequest
		        responseData.ResponseMessage = string(errorJSON1)
		} else {
			jsonData, err := json.Marshal(pocInfo)
			if err != nil {
				errorMessage2 := fmt.Sprintf("Failed to marshal pod information to JSON: %v", err)
				errorJSON2, _ := json.Marshal(map[string]interface{}{
					"error": errorMessage2,
				})
				responseData.ResultCode = http.StatusBadRequest
				responseData.ResponseMessage = string(errorJSON2)
			} else {
				responseData.ResultCode = http.StatusOK
				responseData.ResponseMessage = string(jsonData)
			}
		}
		return responseData
	}


        // Get cnfinfo
	cnfInfoData, cmaerrmsg := GetCnfInfoFromCma()
	if cmaerrmsg != "" {
                responseData.ResultCode = http.StatusServiceUnavailable
                responseData.ResponseMessage = cmaerrmsg
                return responseData
	}

        serviceToPodMultiMap, PodIpMap, PodNameMap := getPodInfoFromCmadata(cnfInfoData)
		isPodidValid, specServiceToPodMultiMap, responseData = validatePodId(&typeData, PodNameMap)
		if !isPodidValid {
			return responseData
		}

        responseData = validateGenerateLabels(typeData)
        if responseData.ResultCode != 0 {
                return responseData
        }

        responseData = validateServiceNames(typeData, serviceToPodMultiMap)
        if responseData.ResultCode != 0 {
                return responseData
        }

        if len(commandsData) > 0 {
                debugLogger.Log.Info("Directly trigger debug log collection")
        } else if len(typeData.ServiceName) > 0 || len(typeData.GenerateLabel) > 0 || len(typeData.Podid) > 0 {
                //Get the commands data with input types
                debugLogger.Log.Info("Generate json content with typedata %s.", typeData)
                debugCollectionResponse := debugForGet.GetRequestJsonFile(typeData)
                commandsData = []byte(debugCollectionResponse.BodyData)
                debugLogger.Log.Info("Get debug log collection request body with input data: ", typeData)
        } else {
                responseData.ResultCode = http.StatusNotFound
                responseData.ResponseMessage = "There is no valid input request data or generate request indication."
                return responseData
        }
        debugLogger.Log.Info("Debug log collection commdsData is %s", string(commandsData))
        var requestBody commonData.TemplateStruct
		err := json.Unmarshal(commandsData, &requestBody)
		if err != nil {
			responseData.ResultCode = http.StatusBadRequest
			responseData.ResponseMessage = fmt.Sprintf("Request body unmarshal failed with error: %s", err)
			return responseData
		}
		responseData = validateInputCommands(requestBody)
        if responseData.ResultCode != 0 {
                return responseData
        }
		if requestBody.SessionName == "" {
			debugLogger.Log.Info("Input session Name is null, defined as default name: debugSession.")
			requestBody.SessionName = "debugSession"
		}
        requestBody.SessionName = requestBody.SessionName + "-" + time.Unix(time.Now().Unix(), 0).Format("20060102150405")
        debugLogger.Log.Info("Trigger debug log collection: requestBody is \n", requestBody)
        if len(requestBody.CaasCommands) > 0 {
            debugLogger.Log.Info("Trigger caas debug log collection")
            go debugForCaasLogCollection.ExeCaasLogCollection(requestBody, caasDebugCollectionResponse, serviceToPodMultiMap, typeData, specServiceToPodMultiMap)
        } else {
            // No need to execute caas log collection, set caas http response as status_ok
            debugLogger.Log.Info("Set caas log collection response as status_ok")
            go sendStatusOkChanData(caasDebugCollectionResponse)
        }
        if len(requestBody.SpecCommands) > 0 {
            debugLogger.Log.Info("Trigger application debug log collection")
            go debugForAppLogCollection.ExeDebugLogCollection(requestBody, cnfInfoData, serviceToPodMultiMap, specServiceToPodMultiMap, PodIpMap, appDebugCollectionResponse)
        } else {
            // No need to execute app log collection, set app http response as status_ok
            debugLogger.Log.Info("Set app log collection response as status_ok")
            go sendStatusOkChanData(appDebugCollectionResponse)
        }
        caasHttpResponse := <- caasDebugCollectionResponse
        appHttpResponse := <- appDebugCollectionResponse

        close(caasDebugCollectionResponse)
        close(appDebugCollectionResponse)

        responseData = combineResponse(caasHttpResponse, appHttpResponse, requestBody.SessionName)

        return responseData
}
