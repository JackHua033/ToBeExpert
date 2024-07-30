package debugForGet

import (
	"bytes"
	"encoding/json"
	"net/http"
	"nokia.com/square/debugassist/commonData"
	"nokia.com/square/debugassist/debugLogger"
	"strings"
)

func isIntersect(slice1, slice2 []string) bool {
	m := make(map[string]int)
	for _, v := range slice1 {
		m[v]++
	}

	for _, v := range slice2 {
		times := m[v]
		if times != 0 {
			return true
		}
	}
	return false
}

func isMatchedLabels(inputLabels []string, commandsLabels []string) bool {
	if (len(inputLabels) == 0 && commonData.IsContain(commandsLabels, "internal")) {
		// Commands with internal label are not returned in json templates
		return false
	} else {
		return (len(inputLabels) == 0 || isIntersect(commandsLabels, inputLabels))
	}
}

func getSpecCommandsLine(subMapC commonData.Multimap, containerCommand commonData.ContainerCommands, input commonData.GetType) {
	for _, commands := range containerCommand.Commands {
		if (isMatchedLabels(input.GenerateLabel, commands.Labels) || strings.TrimSpace(commands.Name) == "commonCommands") {
			if len(commands.Options) != 0 {
				for _, options := range commands.Options {
					commandLine := commands.Name + " " + options
					subMapC.Add(containerCommand.Name, commandLine)
				}
			} else {
				commandLine := commands.Name
				subMapC.Add(containerCommand.Name, commandLine)
			}
		}
	}
}

func getSpecCommandsInfo(specificCommands []commonData.SpecificCommand, input commonData.GetType) map[string]interface{} {
	mainMapC := map[string]interface{}{}

	for _, specCommands := range specificCommands {
		if len(input.ServiceName) != 0 && !commonData.IsContain(input.ServiceName, specCommands.ServiceName) {
			continue
		}
		var subMapC commonData.Multimap
		existMap, ok := mainMapC[specCommands.ServiceName]
		if !ok {
			subMapC = make(commonData.Multimap)
		} else {
			subMapC = existMap.(commonData.Multimap)
		}
		for _, containerCommand := range specCommands.Containers {
			getSpecCommandsLine(subMapC, containerCommand, input)
		}
		if !ok {
			mainMapC[specCommands.ServiceName] = subMapC
		}
	}
	return mainMapC
}

func generateSpecificCommands(specificCommands []commonData.SpecificCommand, input commonData.GetType) []commonData.SpecCommandsStruct {
	var specCommands []commonData.SpecCommandsStruct
	portMapInfo := commonData.GetPortMapInfo()
	debugLogger.Log.Info("portMapInfo is: ", portMapInfo)
	specCommandsMap := getSpecCommandsInfo(specificCommands, input)
	debugLogger.Log.Info("specCommandsMap is: ", specCommandsMap)
	for serviceName, specCommandsMap := range specCommandsMap {
		containerToPortMap, ok := portMapInfo[serviceName]
		if !ok {
			debugLogger.Log.Info(serviceName, " service doesn't support debug log collection. Move on to the next service.")
			continue
		}
		var specCommandsTmp commonData.SpecCommandsStruct
		specCommandsTmp.ServiceName = serviceName
		subMap := specCommandsMap.(commonData.Multimap)
		rceSubMap := containerToPortMap.(map[string]int)
		for containerName, commands := range subMap {
			_, ok := rceSubMap[containerName]
			if !ok {
				debugLogger.Log.Info("For", serviceName, "service,", containerName, "container doesn't support debug log collection")
				continue
			}
			var containersTmp commonData.ContainersStruct
			containersTmp.Name = containerName
			containersTmp.Commands = append(containersTmp.Commands, commands...)
			specCommandsTmp.Containers = append(specCommandsTmp.Containers, containersTmp)
		}
		if specCommandsTmp.Containers == nil {
			debugLogger.Log.Info("For", serviceName, "service ,none of the containers defined in the log collection request are available. Move on to the next service.")
			continue
		}
		specCommands = append(specCommands, specCommandsTmp)
	}

	return specCommands
}

func generateCommonCommands(totalCommonCommands []commonData.CommandsAttrStruct, input commonData.GetType) []string {
	var commonCommands []string
	for _, commandInfo := range totalCommonCommands {
		if isMatchedLabels(input.GenerateLabel, commandInfo.Labels) {
			if len(commandInfo.Options) != 0 {
				for _, commandOption := range commandInfo.Options {
					commandLine := commandInfo.Name + " " + commandOption
					commonCommands = append(commonCommands, commandLine)
				}
			} else {
				commandLine := commandInfo.Name
				commonCommands = append(commonCommands, commandLine)
			}
		}
	}
	return commonCommands
}

func GetRequestJsonFile(input commonData.GetType) commonData.HttpServerResponse {
	var responseData = commonData.HttpServerResponse{http.StatusOK, "", ""}

	var debugRequestJson commonData.TemplateStruct

	// generate template
	// 1.add session name
	if len(input.GenerateLabel) == 0 && len(input.ServiceName) == 0 && len(input.Podid) == 0{
		debugRequestJson.SessionName = "##"
	} else {
		debugRequestJson.SessionName = "debugSession"
	}

	// 2.add cass commands
	for _, caasCommandsInfo := range commonData.CaasCommandsData.CaasCommands {
		if isMatchedLabels(input.GenerateLabel, caasCommandsInfo.Labels) {
			debugRequestJson.CaasCommands = append(debugRequestJson.CaasCommands, caasCommandsInfo.Name)
		}
	}
	// if get serviceName from -s option, then update the services with input services
	if len(input.ServiceName) > 0 {
		debugRequestJson.Services = input.ServiceName
	} else {
		debugRequestJson.Services = commonData.CaasCommandsData.Services
	}
	debugRequestJson.LiServices    = commonData.CaasCommandsData.LiServices
	debugRequestJson.LiLanNames    = commonData.CaasCommandsData.LiLanNames
	debugRequestJson.LimitServices = commonData.CaasCommandsData.LimitServices

	// 3.add common commands
	totalCommonCommands := append(commonData.CommonCommandsData.NativeCommands, commonData.CommonCommandsData.ScriptCommands...)
	debugRequestJson.CommonCommands = generateCommonCommands(totalCommonCommands, input)

	// 4.get service info, add service name/containers name/product specific commands
	specificCommands := append(commonData.ProductCommandsData.NativeCommands, commonData.ProductCommandsData.ScriptCommands...)
	debugRequestJson.SpecCommands = generateSpecificCommands(specificCommands, input)

	// 6.Generate json
	byteArray := bytes.NewBuffer([]byte{})
	encoder := json.NewEncoder(byteArray)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	err := encoder.Encode(debugRequestJson)
	if err != nil {
		byteArray1, _ := json.MarshalIndent(debugRequestJson, "", "  ")
		responseData.BodyData = string(byteArray1)
		debugLogger.Log.Error(err)
	} else {
	    responseData.BodyData = byteArray.String()
	}
	return responseData
}
