package commonData

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
        "nokia.com/square/debugassist/debugLogger"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	corev1 "k8s.io/api/core/v1"
)

var  CertificateFiles CertificateFilesData
var CaasCommandsData CaasCommandsStruct
var CommonCommandsData CommonCommandsStruct
var CommonConfigData CommonConfigStruct
var ProductCommandsData ProductCommandsStruct
var RceServerData []RceServerStruct
var ConfigFilePath string

// multimap type and methods defination
type Multimap map[string][]string

func (multimap Multimap) Add(key, value string) {
	if len(multimap[key]) == 0 {
		multimap[key] = []string{value}
	} else {
		multimap[key] = append(multimap[key], value)
	}
}

func (multimap Multimap) Get(key string) []string {
	if multimap == nil {
		return nil
	}
	values := multimap[key]
	return values
}

func IsContain(items []string, item string) bool {
	for _, eachItem := range items {
		// Ignore case
		if strings.EqualFold(eachItem, item) {
			return true
		}
	}
	return false
}

func GetEnvValueWithDefault(envName string, defaultValue string) (string){
	retValue := defaultValue
	tmpValue := os.Getenv(envName)
	if len(tmpValue) > 0 {
		retValue = tmpValue
	}
	return retValue
}

func CheckConfigFilePath() (*string){
	ConfigFilePath = GetEnvValueWithDefault("DEBUG_ASSIST_MOUNT_PATH", "/opt/debugassist/")
	debugLogger.Log.Info("ConfigFilePath is %s", ConfigFilePath)
	if _, err := os.Stat(ConfigFilePath); err != nil {
		errMsg := fmt.Sprintf("The path %s for debugassist config files does not exist: %s", ConfigFilePath, err)
		return &errMsg
	}
	return nil
}
func GetCaasCommands() (*string){
	caasCommandFile := fmt.Sprintf("%s%s", ConfigFilePath ,"debugAssistCaasCommands.json")
	if _, err := os.Stat(caasCommandFile); err != nil {
		debugLogger.Log.Info("There is no caas command config file %s, don't need to decode caas commands.", caasCommandFile)
		return nil
	}
	caasCommandBytes, err := ioutil.ReadFile(caasCommandFile)
	if err != nil {
		errMsg := fmt.Sprintf("Unable to load debug assist caas commands json file! Error is %s", err)
        return &errMsg
	}
	err = json.Unmarshal(caasCommandBytes, &CaasCommandsData)
	if err != nil {
		errMsg := fmt.Sprintf("debug assist caas commands JSON decode error! Error is %s", err)
		return &errMsg
	}
	return nil
}

func GetCommonCommands() (*string){
	commonCommandFile := fmt.Sprintf("%s%s", ConfigFilePath ,"debugAssistCommonCommands.json")
	if _, err := os.Stat(commonCommandFile); err != nil {
		debugLogger.Log.Info("There is no common command config file %s, don't need to decode common commands.", commonCommandFile)
		return nil
	}
	commCommandBytes, err := ioutil.ReadFile(commonCommandFile)
	if err != nil {
		errMsg := fmt.Sprintf("Unable to load debug assist common commands json file! Error is %s", err)
		return &errMsg
	}
	err = json.Unmarshal(commCommandBytes, &CommonCommandsData)
	if err != nil {
		errMsg := fmt.Sprintf("debug assist common commands JSON decode error! Error is %s", err)
		return &errMsg
	}
	return nil
}

func GetProductCommands() (*string){
	productCommandFile := fmt.Sprintf("%s%s", ConfigFilePath ,"debugAssistCnfCommands.json")
	if _, err := os.Stat(productCommandFile); err != nil {
		debugLogger.Log.Info("There is no product command config file %s, don't need to decode product commands.", productCommandFile)
		return nil
	}
	productCommandBytes, err := ioutil.ReadFile(productCommandFile)
	if err != nil {
		errMsg := fmt.Sprintf("Unable to load debug assist product commands json file! Error is %s", err)
		return &errMsg
	}
	err = json.Unmarshal(productCommandBytes, &ProductCommandsData)
	if err != nil {
		errMsg := fmt.Sprintf("debug assist product commands JSON decode error! Error is %s", err)
		return &errMsg
	}

	return nil
}

func GetRceServer() (*string){
	rceServerFile := fmt.Sprintf("%s%s", ConfigFilePath ,"debugAssistRceServer.json")
	if _, err := os.Stat(rceServerFile); err != nil {
		debugLogger.Log.Info("There is no rce server config file %s, don't need to decode rce server configs.", rceServerFile)
		return nil
	}

	rceServerBytes, err := ioutil.ReadFile(rceServerFile)
	if err != nil {
		errMsg := fmt.Sprintf("Unable to load debug assist rce server json file! Error is %s", err)
		return &errMsg
	}
	err = json.Unmarshal(rceServerBytes, &RceServerData)
	if err != nil {
		errMsg := fmt.Sprintf("debug assist rce server JSON decode error! Error is %s", err)
		return &errMsg
	}
        return nil
}

func GetCommonConfig() (*string){
	commonConfigFile := fmt.Sprintf("%s%s", ConfigFilePath ,"debugAssistConfig.json")
	if _, err := os.Stat(commonConfigFile); err != nil {
		debugLogger.Log.Info("There is no common config file %s, don't need to decode common configs.", commonConfigFile)
		return nil
	}

	commonConfigBytes, err := ioutil.ReadFile(commonConfigFile)
	if err != nil {
		errMsg := fmt.Sprintf("Unable to load debug assist common config json file! Error is %s", err)
		return &errMsg
	}
	err = json.Unmarshal(commonConfigBytes, &CommonConfigData)
	if err != nil {
		errMsg := fmt.Sprintf("debug assist common config JSON decode error! Error is %s", err)
		return &errMsg
	}
        return nil
}

func GetLimitSizeMapInfo() map[string]int {
	limitSizeMap := make(map[string]int)
	for _, serviceConfig := range CommonConfigData.ConfigPerContainer {
		for _, container := range serviceConfig.Containers {
			limitSizeMap[serviceConfig.ServiceName] = container.LimitSize
		}
	}
	return limitSizeMap
}

func GetPortMapInfo() map[string]interface{} {
	mainMapC := map[string]interface{}{}

	for _, rceServer := range RceServerData {
		subMapC := make(map[string]int)
		for _, containerName := range rceServer.Containers {
			subMapC[containerName.Name] = containerName.Port
		}
		mainMapC[rceServer.ServiceName] = subMapC
	}
	return mainMapC
}

func GetClientset() (*rest.Config, *kubernetes.Clientset, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, nil, err
	}

	clientSet, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, nil, err
	}

	return config, clientSet, nil
}

func FilterPodByName(pods *corev1.PodList, podName string) []corev1.Pod {
	var filteredPods []corev1.Pod
	for _, pod := range pods.Items {
		if pod.ObjectMeta.Name == podName {
			filteredPods = append(filteredPods, pod)
		}
	}
	return filteredPods
}

func GetPods(ctx context.Context, clientSet kubernetes.Interface, serviceName string, namespace string, specServicePodsMap Multimap) (*corev1.PodList, error) {
	listOptions := metav1.ListOptions{LabelSelector: ""}
	if serviceName != "All" {
		listOptions.LabelSelector = fmt.Sprintf("serviceType=%s", serviceName)
	}

	pods, err := clientSet.CoreV1().Pods(namespace).List(ctx, listOptions)
	if err != nil {
		return nil, fmt.Errorf("Failed to get pods for service: %s. Error: %s", serviceName, err)
	}

	if len(specServicePodsMap) == 0 {
		return pods, nil
	}

	filteredPods := make([]corev1.Pod, 0)
	podNames, ok := specServicePodsMap[serviceName]
	if !ok {
		debugLogger.Log.Info("No pods in specServicePodsMap for service: %s", serviceName)
		return pods, nil
	}
	for _, podName := range podNames {
		filteredPods = append(filteredPods, FilterPodByName(pods, podName)...)
	}

	return &corev1.PodList{
		Items: filteredPods,
	}, nil
}

func GetServicesStringList(serviceToPodMultiMap Multimap) []string {
        var servicesStringList []string
        for service := range serviceToPodMultiMap {
                if service != "" {
                        servicesStringList = append(servicesStringList, service)
                }
        }
        return servicesStringList
}

func IsTargetStringInList(stringList []string, targetString string) bool {
	for _, item := range stringList {
		if item == targetString {
			return true
		}
	}
	return false
}
