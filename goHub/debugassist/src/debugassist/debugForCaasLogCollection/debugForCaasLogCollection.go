package debugForCaasLogCollection

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"time"
	"net/http"
	"nokia.com/square/debugassist/commonData"
	"nokia.com/square/debugassist/debugLogger"
	"k8s.io/metrics/pkg/apis/metrics/v1beta1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/dynamic"
	metrics "k8s.io/metrics/pkg/client/clientset/versioned"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	kubernetes "k8s.io/client-go/kubernetes"
	rest "k8s.io/client-go/rest"
)

type CaasK8sInterface struct {
	KubeConfig	*rest.Config
	KubeClientSet	*kubernetes.Clientset
	MetricClientSet	*metrics.Clientset
	Namespace	string
	Token		string
}

type LogFile struct {
	TimeStamp string      `json:"TimeStamp"`
	Command   string      `json:"Command"`
	Body      interface{} `json:"Body"`
}

type SplitResult struct {
	SpecificItems interface{}
	OtherItems    interface{}
}

const CTX_TIMEOUT	= time.Second * 60
const LI_LOG_DIR	= "lilog"
const LOG_DIRECTORY	= "/logstore/debugassist"
const NETWORK_API_GROUP	= "k8s.cni.cncf.io"
const NETWORK_RESOURCE	= "network-attachment-definitions"
const SERVICE_TYPE	= "serviceType"

var caasBuilder          strings.Builder
var requestBodyData      commonData.TemplateStruct
var specServicePodMap    commonData.Multimap
var serviceToPodMultiMap commonData.Multimap

func ClearManagedFields(items interface{}) interface{} {
	switch items := items.(type) {
	case *appsv1.StatefulSetList:
		for state := range items.Items {
			items.Items[state].ObjectMeta.ManagedFields = []metav1.ManagedFieldsEntry{}
		}
	case *appsv1.DeploymentList:
		for deploy := range items.Items {
			items.Items[deploy].ObjectMeta.ManagedFields = []metav1.ManagedFieldsEntry{}
		}
	case *corev1.EventList:
		for event := range items.Items {
			items.Items[event].ObjectMeta.ManagedFields = []metav1.ManagedFieldsEntry{}
		}
	default:
		debugLogger.Log.Info("Unexpected type:", reflect.TypeOf(items))
	}
	return items
}

func CreationTime(path string) int64 {
	info, _ := os.Stat(path)
	return info.ModTime().UnixNano()
}

func DirectorySize(path string) (int64, error) {
	var size int64
	err := filepath.Walk(path, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			size += info.Size()
		}
		return nil
	})
	return size, err
}

func GetItemLabels(item interface{}) map[string]string {
	switch item := item.(type) {
	case *appsv1.Deployment:
		return item.Spec.Template.ObjectMeta.Labels
	case *appsv1.StatefulSet:
		return item.Spec.Template.ObjectMeta.Labels
	}
	return nil
}

func Int64Ptr(i int64) *int64 {
	return &i
}

func RemoveNonRunningPods(pods *corev1.PodList) *corev1.PodList {
	runningPods := &corev1.PodList{}

	for _, pod := range pods.Items {
		if pod.Status.Phase == corev1.PodRunning {
			runningPods.Items = append(runningPods.Items, pod)
		} else {
			debugLogger.Log.Info("The Status of pod: %s is not Running.", pod.Name)
		}
	}
	return runningPods
}

func RemoveSessionDirIfOverLimitSize(debugSessionName string) {
	limitSizeMap := commonData.GetLimitSizeMapInfo()
	maxSizeInMB := limitSizeMap["DGAT"]
	maxSize := int64(maxSizeInMB) * 1024 * 1024
	debugLogger.Log.Info("--INFO--RemoveSessionDirIfOverLimitSize--: the maxSize: %d", maxSize)

	totalSize, err := DirectorySize(LOG_DIRECTORY)
	if err != nil {
		debugLogger.Log.Error("--ERROR--RemoveSessionDirIfOverLimitSize--: Get %s size failed.", LOG_DIRECTORY)
		return
	}

	if totalSize < maxSize {
		debugLogger.Log.Info("Directory size is %d bytes, which is less than %d MB.\n", totalSize, maxSizeInMB)
		return
	}

	dirList, err := filepath.Glob(filepath.Join(LOG_DIRECTORY, "debugSession-*"))
	if err != nil || len(dirList) == 0 {
		debugLogger.Log.Info("--INFO--RemoveSessionDirIfOverLimitSize--: No debugSession directory under:%s.", LOG_DIRECTORY)
		return
	}

	sort.Slice(dirList, func(i, j int) bool {
		return CreationTime(dirList[i]) < CreationTime(dirList[j])
	})

	for _, dir := range dirList {
		if totalSize < maxSize {
			break
		}
		if filepath.Base(dir) != debugSessionName {
			debugLogger.Log.Info("--INFO--RemoveSessionDirIfOverLimitSize--: Remove directory: %s.", dir)
			os.RemoveAll(dir)
			totalSize, _ = DirectorySize(LOG_DIRECTORY)
		}
	}
}

func SpecificServicePodName(involvedObjectName string) bool {
	objectParts := strings.Split(involvedObjectName, "-")
	if len(objectParts) < 3 {
		return false
	}

	serviceType := strings.ToLower(objectParts[1])
	if commonData.IsTargetStringInList(requestBodyData.LiServices, serviceType) {
		return true
	}

	return false
}

func SplitDeployItemsByLabel(result interface{}, labelKey string) SplitResult {
	deployments, ok := result.(*appsv1.DeploymentList)
	if !ok {
		debugLogger.Log.Error("Failed to get deployment item list.")
		return SplitResult{
			SpecificItems: nil,
			OtherItems:    result,
		}
	}

	specificDeploy := make([]interface{}, 0)
	otherDeploy    := make([]interface{}, 0)
	for _, deployment := range deployments.Items {
		depItemLabelValue, _ := deployment.Spec.Template.ObjectMeta.Labels[labelKey]
		if depItemLabelValue != "" && commonData.IsTargetStringInList(requestBodyData.LiServices, depItemLabelValue) {
			specificDeploy = append(specificDeploy, deployment)
		} else {
			otherDeploy = append(otherDeploy, deployment)
		}
	}

	return SplitResult{
		SpecificItems: specificDeploy,
		OtherItems:    otherDeploy,
	}
}

func SplitResourceLogByLabels(result interface{}, apiName string, labelKey string) SplitResult {
	switch apiName {
	case "deployments":
		return SplitDeployItemsByLabel(result, labelKey)
	case "statefulsets":
		return SplitStateItemsByLabel(result, labelKey)
	case "events":
		return SplitEventsByPodNamePrefix(result)
	default:
		debugLogger.Log.Info("Not supported apiName: %s", apiName)
		return SplitResult{
			SpecificItems: nil,
			OtherItems:    []interface{}{result},
		}
	}
}

func SplitEventsByPodNamePrefix(result interface{}) SplitResult {
	events, ok := result.(*corev1.EventList)
	if !ok {
		debugLogger.Log.Error("Failed to get event item list.")
		return SplitResult{
			SpecificItems: nil,
			OtherItems:    result,
		}
	}

	specificEvents := make([]interface{}, 0)
	otherEvents    := make([]interface{}, 0)
	for _, event := range events.Items {
		involvedObject := event.InvolvedObject
		if event.InvolvedObject.Kind == "Pod" {
			podName := involvedObject.Name
			if SpecificServicePodName(podName) {
				specificEvents = append(specificEvents, event)
			} else {
				otherEvents = append(otherEvents, event)
			}
		}
	}

	return SplitResult{
		SpecificItems: specificEvents,
		OtherItems:    otherEvents,
	}
}

func SplitStateItemsByLabel(result interface{}, labelKey string) SplitResult {
	statefulSets, ok := result.(*appsv1.StatefulSetList)
	if !ok {
		debugLogger.Log.Error("Failed to get statefulset item list.")
		return SplitResult{
			SpecificItems: nil,
			OtherItems:    result,
		}
	}

	specificState := make([]interface{}, 0)
	otherState    := make([]interface{}, 0)
	for _, statefulSet := range statefulSets.Items {
		stateItemLabelValue, _ := statefulSet.Spec.Template.ObjectMeta.Labels[labelKey]
		if stateItemLabelValue != "" && commonData.IsTargetStringInList(requestBodyData.LiServices, stateItemLabelValue) {
			specificState = append(specificState, statefulSet)
		} else {
			otherState = append(otherState, statefulSet)
		}
	}

	return SplitResult{
		SpecificItems: specificState,
		OtherItems:    otherState,
	}
}

func UpdatePathForLiService(inputService, fileDir string) string {
	outputDir := fileDir
	if commonData.IsTargetStringInList(requestBodyData.LiServices, inputService) {
		outputDir = filepath.Join(LOG_DIRECTORY, requestBodyData.SessionName, LI_LOG_DIR)
	}
	return outputDir
}

func UpdateServiceNameList(configServices, inputServices []string) []string {
	if len(inputServices) == 0 && len(configServices) > 0 && !commonData.IsTargetStringInList(configServices, "All") {
		inputServices = append(inputServices, configServices...)
	}
	return inputServices
}

func WriteResultToFile(data interface{}, outputDir string, fileName string) string {
	if _, err := os.Stat(outputDir); os.IsNotExist(err) {
		if err := os.MkdirAll(outputDir, 0755); err != nil {
			return fmt.Sprintf("Failed to create directory: %s", outputDir)
		}
	}

	filePath := filepath.Join(outputDir, fileName)
	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Sprintf("Failed to create file: %s under: %s", fileName, outputDir)
	}
	defer file.Close()

	fileNameWithoutExt := strings.TrimSuffix(filepath.Base(fileName), filepath.Ext(fileName))
	logFile := LogFile{
		TimeStamp: time.Now().UTC().Format(time.RFC3339),
		Command:   fileNameWithoutExt,
	}
	switch data.(type) {
	case []string:
		stringData := strings.Join(data.([]string), "\n")
		logFile.Body = strings.Split(stringData, "\n")
	default:
		logFile.Body = data
	}
	logData, err := json.MarshalIndent(logFile, "", "  ")
	if err != nil {
		return fmt.Sprintf("Failed to marshal log data to JSON for file: %s. Error: %s", fileName, err)
	}

	_, err = file.Write(logData)
	if err != nil {
		return fmt.Sprintf("Failed to write log data to file: %s. Error: %s", fileName, err)
	}

	_, err = file.WriteString("\n")
	if err != nil {
		return fmt.Sprintf("Failed to write newline character to file: %s. Error: %s", fileName, err)
	}

	debugLogger.Log.Info("--INFO--WriteResultToFile--: Succeed to write logs into: %s", filePath)

	return ""
}

func Init() (*CaasK8sInterface, string) {
	var (
		kubeConfig	*rest.Config
		kubeClientSet	*kubernetes.Clientset
		metricClientSet	*metrics.Clientset
	)

	cToken, err := ioutil.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/token")
	if err != nil {
		return nil, fmt.Sprintf("Unable to read token file. ERR = %s", err)
	}
	debugLogger.Log.Info("--INFO--Init--: Token - %s", cToken)

	cNamespace, err := ioutil.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/namespace")
	if err != nil {
		return nil, fmt.Sprintf("Unable to read namespace file. ERR = %s", err)
	}
	debugLogger.Log.Info("--INFO--Init--: Namespace - %s", cNamespace)

	kubeConfig, kubeClientSet, err = commonData.GetClientset()
	if err != nil {
		return nil, fmt.Sprintf("Unable to create kubernetes clientSet. ERR = %s", err)
	}
	debugLogger.Log.Info("--INFO--Init--: Succeed to create Kubernetes REST client.")

	metricClientSet, err = metrics.NewForConfig(kubeConfig)
	if err != nil {
		return nil, fmt.Sprintf("Unable to create metrics clientSet. ERR = %s", err)
	}
	debugLogger.Log.Info("--INFO--Init--: Succeed to create Metrics REST client.")

	return &CaasK8sInterface{
		KubeConfig:		kubeConfig,
		KubeClientSet:		kubeClientSet,
		MetricClientSet:	metricClientSet,
		Namespace:	string(cNamespace),
		Token:		string(cToken),
	}, ""
}

func (c *CaasK8sInterface) GetPods(serviceNames []string, fileDir string) string {
	var (
		ctx	context.Context
		cancel	context.CancelFunc
		errors	[]error
	)
	ctx, cancel = context.WithTimeout(context.Background(), CTX_TIMEOUT)
	defer cancel()

	if len(serviceNames) == 0 {
		serviceNames = commonData.GetServicesStringList(serviceToPodMultiMap)
	}
	for _, serviceName := range serviceNames {
		pods, err := commonData.GetPods(ctx, c.KubeClientSet, serviceName, c.Namespace, specServicePodMap)
		if err != nil {
			errors = append(errors, err)
			continue
		}

		fileName := fmt.Sprintf("%s-%s-%s.log", "kubectl-get-pods", c.Namespace, serviceName)
		for i := range pods.Items {
			pods.Items[i].ObjectMeta.ManagedFields = nil
		}
		outputDir := UpdatePathForLiService(serviceName, fileDir)
		if errMsg := WriteResultToFile(pods, outputDir, fileName); errMsg != "" {
			errors = append(errors, fmt.Errorf("%s", errMsg))
			continue
		}
		debugLogger.Log.Info("--INFO--GetPods--: Succeed to get pods for service: %s", serviceName)
	}

	if len(errors) > 0 {
		errMsgs := make([]string, len(errors))
		for i, err := range errors {
			errMsgs[i] = err.Error()
		}
		return fmt.Sprintf("Get pods failed: %s\n", strings.Join(errMsgs, "\n"))
	}

	caasBuilder.WriteString(fmt.Sprintf("Succeed to run command: kubectl get pods -n %s\n", c.Namespace))
	return ""
}

func (c *CaasK8sInterface) GetKubenetesResources(apiName string, fileDir string) string {
	var (
		ctx	context.Context
		cancel	context.CancelFunc
		result	interface{}
		err	error
	)
	ctx, cancel = context.WithTimeout(context.Background(), CTX_TIMEOUT)
	defer cancel()

	switch apiName {
	case "statefulsets":
		result, err = c.KubeClientSet.AppsV1().StatefulSets(c.Namespace).List(ctx, metav1.ListOptions{})
	case "deployments":
		result, err = c.KubeClientSet.AppsV1().Deployments(c.Namespace).List(ctx, metav1.ListOptions{})
	case "events":
		result, err = c.KubeClientSet.CoreV1().Events(c.Namespace).List(ctx, metav1.ListOptions{})
	default:
		return fmt.Sprintf("Invalid API name: %s", apiName)
	}
	if err != nil {
		return fmt.Sprintf("Unable to get %s for namespace: %s. Error: %s", apiName, c.Namespace, err)
	}

	fileName := fmt.Sprintf("%s-%s.log", "kubectl-get-"+apiName, c.Namespace)
	result = ClearManagedFields(result)

	debugLogger.Log.Info("Ready to call SplitResourceLogByLabels for %s", apiName)
	splitResult := SplitResourceLogByLabels(result, apiName, SERVICE_TYPE)
	if specificItems, ok := splitResult.SpecificItems.([]interface{}); ok && specificItems != nil && len(specificItems) > 0 {
		outputDir := filepath.Join(LOG_DIRECTORY, requestBodyData.SessionName, LI_LOG_DIR)
		if errMsg := WriteResultToFile(splitResult.SpecificItems, outputDir, fileName); errMsg != "" {
			return fmt.Sprintf("Failed to write specific items into file for ns: %s, api: %s. Error: %s", c.Namespace, apiName, errMsg)
		}
	}

	if errMsg := WriteResultToFile(splitResult.OtherItems, fileDir, fileName); errMsg != "" {
		return fmt.Sprintf("Failed to write get-%s result into file for namespace: %s. Error: %s", apiName, c.Namespace, errMsg)
	}
	debugLogger.Log.Info(fmt.Sprintf("--INFO--Get%s--: Succeed to get %s for namespace: %s", strings.Title(apiName), apiName, c.Namespace))
	caasBuilder.WriteString(fmt.Sprintf("Succeed to run command: kubectl get %s -n %s\n", apiName, c.Namespace))

	return ""
}

func (c *CaasK8sInterface) GetPodMetricsList(ctx context.Context, serviceName string) ([]*v1beta1.PodMetrics, error) {
	pods, err := commonData.GetPods(ctx, c.KubeClientSet, serviceName, c.Namespace, specServicePodMap)
	if err != nil {
		return nil, err
	}

	runningPods := RemoveNonRunningPods(pods)
	metricsClient := c.MetricClientSet.MetricsV1beta1()
	var podMetricsList []*v1beta1.PodMetrics
	for _, pod := range runningPods.Items {
		podCtx, podCancel := context.WithTimeout(ctx, CTX_TIMEOUT) // Increase the timeout for this specific pod
		defer podCancel()
		podMetrics, err := metricsClient.PodMetricses(c.Namespace).Get(podCtx, pod.Name, metav1.GetOptions{})
		if err != nil {
			return nil, fmt.Errorf("Unable to get pod metrics for service: %s. Error: %s", serviceName, err)
		}
		podMetricsList = append(podMetricsList, podMetrics)
	}

	return podMetricsList, nil
}

func (c *CaasK8sInterface) GetPodMetrics(serviceNames []string, fileDir string) string {
	var (
		ctx		context.Context
		cancel		context.CancelFunc
		errors		[]error
	)
	ctx, cancel = context.WithTimeout(context.Background(), CTX_TIMEOUT)
	defer cancel()

	if len(serviceNames) == 0 {
		serviceNames = commonData.GetServicesStringList(serviceToPodMultiMap)
	}
	for _, serviceName := range serviceNames {
		podMetricsList, err := c.GetPodMetricsList(ctx, serviceName)
		if err != nil {
			errors = append(errors, err)
			continue
		}
		outputDir := UpdatePathForLiService(serviceName, fileDir)
		fileName := fmt.Sprintf("%s-%s-%s.log", "kubectl-top-pod", c.Namespace, serviceName)
		if errMsg := WriteResultToFile(podMetricsList, outputDir, fileName); errMsg != "" {
			errors = append(errors, fmt.Errorf("%s", errMsg))
			continue
		}
		debugLogger.Log.Info("--INFO--GetPodMetrics--: Succeed to get top pod for service: %s", serviceName)
	}

	if len(errors) > 0 {
		errMsgs := make([]string, len(errors))
		for i, err := range errors {
			errMsgs[i] = err.Error()
		}
		return fmt.Sprintf("Get top pod failed: %s\n", strings.Join(errMsgs, "\n"))
	}

	caasBuilder.WriteString(fmt.Sprintf("Succeed to run command: kubectl top pod -n %s\n", c.Namespace))
	return ""
}

func (c *CaasK8sInterface) GetContainerLogs(ctx context.Context, podName string, containerName string) ([]byte, error) {
	podLogOpts := &corev1.PodLogOptions{
		Container:      containerName,
		Follow:         false,
		Previous:       false,
		Timestamps:     false,
		SinceSeconds:   Int64Ptr(48 * 60 * 60),
	}
	podLogs, err := c.KubeClientSet.CoreV1().Pods(c.Namespace).GetLogs(podName, podLogOpts).Stream(context.Background())
	if err != nil {
		return nil, fmt.Errorf("Unable to get logs for pod: %s, container:%s. Error: %s", podName, containerName, err)
	}
	defer podLogs.Close()

	logsBytes, err := ioutil.ReadAll(podLogs)
	if err != nil {
		return nil, fmt.Errorf("Failed to read logs for pod: %s, container: %s. Error: %s", podName, containerName, err)
	}

	return logsBytes, nil
}

func (c *CaasK8sInterface) GetLogsForPod(ctx context.Context, serviceName, fileDir string, pods *corev1.PodList, errors *[]error) bool {
	for _, pod := range pods.Items {
		podCtx, podCancel := context.WithTimeout(ctx, CTX_TIMEOUT) // Increase the timeout for this specific pod
		defer podCancel()

		var logs []string
		for _, container := range pod.Spec.Containers {
			logsBytes, err := c.GetContainerLogs(podCtx, pod.Name, container.Name)
			if err != nil {
				*errors = append(*errors, err)
				continue
			}

			logsString := string(logsBytes)
			if logsString != "" {
				logsWithContainerName := fmt.Sprintf("[%s] %s", container.Name, logsString)
				logs = append(logs, logsWithContainerName)
			}
		}

		if len(logs) <= 0 {
			debugLogger.Log.Info("--INFO--GetLogsForPod--: No service logs for service: %s, pod:%s", serviceName, pod.Name)
			continue
		}

		fileName := fmt.Sprintf("%s-%s-%s-%s.log", "kubectl-logs", c.Namespace, serviceName, pod.Name)
		if errMsg := WriteResultToFile(logs, fileDir, fileName); errMsg != "" {
			*errors = append(*errors, fmt.Errorf("%s", errMsg))
			continue
		}

		debugLogger.Log.Info("--INFO--GetLogsForPod--: Succeed to get pod logs for service: %s", serviceName)
	}

	return len(*errors) == 0
}

func (c *CaasK8sInterface) RetrieveLogsForService(ctx context.Context, serviceName string, fileDir string) string{
	var errors []error
	pods, err := commonData.GetPods(ctx, c.KubeClientSet, serviceName, c.Namespace, specServicePodMap)
	if err != nil {
		return fmt.Sprintf("Failed to get pods for service: %s when RetrieveLogsForService. Error: %s", serviceName, err)
	}

	runningPods := RemoveNonRunningPods(pods)
	if success := c.GetLogsForPod(ctx, serviceName, fileDir, runningPods, &errors); !success {
		errors = append(errors, fmt.Errorf("Failed to process logs for service: %s", serviceName))
	}

	if len(errors) > 0 {
		errMsgs := make([]string, len(errors))
		for i, err := range errors {
			errMsgs[i] = err.Error()
		}
		return fmt.Sprintf("Retrieve logs failed: %s\n", strings.Join(errMsgs, "\n"))
	}

	return ""
}

func (c *CaasK8sInterface) GetServiceLogs(serviceNames []string, fileDir string) string {
	var (
		ctx		context.Context
		cancel		context.CancelFunc
		errors		[]error
	)
	ctx, cancel = context.WithTimeout(context.Background(), CTX_TIMEOUT)
	defer cancel()

	if len(serviceNames) == 0 {
		serviceNames = commonData.GetServicesStringList(serviceToPodMultiMap)
	}
	for _, serviceName := range serviceNames {
		if commonData.IsTargetStringInList(requestBodyData.LimitServices, serviceName) {
			debugLogger.Log.Info("--INFO--GetServiceLogs--: No need to get pod logs for service: %s", serviceName)
			continue
		}
		outputDir := UpdatePathForLiService(serviceName, fileDir)
		if errMsg := c.RetrieveLogsForService(ctx, serviceName, outputDir); errMsg != "" {
			errors = append(errors, fmt.Errorf("%s", errMsg))
		}
	}

	if len(errors) > 0 {
		errMsgs := make([]string, len(errors))
		for i, err := range errors {
			errMsgs[i] = err.Error()
		}
		return fmt.Sprintf("Get logs failed: %s\n", strings.Join(errMsgs, "\n"))
	}
	caasBuilder.WriteString(fmt.Sprintf("Succeed to run command: kubectl logs -n %s \n", c.Namespace))
	return ""
}

func (c *CaasK8sInterface) GetGroupResources(apiGroup string, resource string, fileDir string) string {
	kubeConfig, _, err := commonData.GetClientset()
	clientset, err := dynamic.NewForConfig(kubeConfig)
	if err != nil {
		return fmt.Sprintf("Failed to create client for GetKubernetesResources: %s", err.Error())
	}

	gvr := schema.GroupVersionResource{
		Group:    apiGroup,
		Version:  "v1",
		Resource: resource,
	}
	list, err := clientset.Resource(gvr).Namespace(c.Namespace).List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return fmt.Sprintf("Failed to get the %s list: %s", resource, err.Error())
	}

	specificItems := make([]interface{}, 0)
	otherItems := make([]interface{}, 0)
	for _, listItem := range list.Items {
		unstructured.RemoveNestedField(listItem.Object, "metadata", "managedFields")
		nameString := listItem.GetName()
		if len(requestBodyData.LiLanNames) > 0 &&
			commonData.IsTargetStringInList(requestBodyData.LiLanNames, nameString) {
			specificItems = append(specificItems, listItem)
		} else {
			otherItems = append(otherItems, listItem)
		}
	}

	fileName := fmt.Sprintf("%s-%s.log", "kubectl-get-"+resource, c.Namespace)
	if len(specificItems) > 0 {
		outputDir := filepath.Join(LOG_DIRECTORY, requestBodyData.SessionName, LI_LOG_DIR)
		if errMsg := WriteResultToFile(specificItems, outputDir, fileName); errMsg != "" {
			return fmt.Sprintf("Failed to write specific name item into file for namespace: %s. Error: %s", c.Namespace, errMsg)
		}
	}

	if errMsg := WriteResultToFile(otherItems, fileDir, fileName); errMsg != "" {
		return fmt.Sprintf("Failed to write get-%s result into file for namespace: %s. Error: %s", resource, c.Namespace, errMsg)
	}
	debugLogger.Log.Info(fmt.Sprintf("--INFO--GetGroupResources--: Succeed to get %s for namespace: %s", resource, c.Namespace))
	caasBuilder.WriteString(fmt.Sprintf("Succeed to run command: kubectl get network-attachment-%s \n", "definitions.k8s.cni.cncf.io"))

	return ""
}

func ExeCaasLogCollection(requestBody commonData.TemplateStruct, caasRespData chan commonData.HttpServerResponse, servicesListMap commonData.Multimap, typeData commonData.GetType, servicePodsMap commonData.Multimap) {
	var (
		errorMsgs	[]string
		runCmdSuccess	= true
		responseData	= commonData.HttpServerResponse{http.StatusOK, "", ""}
	)

	// 1. Create k8s clientset
	caasK8sInterface, initErrMsg := Init()
	if initErrMsg != "" {
		responseData.ResultCode = http.StatusInternalServerError
		responseData.ResponseMessage = initErrMsg
		caasRespData <- responseData
		return
	}

	// 2. Create output directory for input session with timestamp
	requestBodyData = requestBody
	outputDir := filepath.Join(LOG_DIRECTORY, requestBodyData.SessionName, "caaslog")

	// 3. Call each command and write the result to a specific file
	typeData.ServiceName = UpdateServiceNameList(requestBodyData.Services, typeData.ServiceName)
	specServicePodMap = servicePodsMap
	serviceToPodMultiMap = servicesListMap
	for _, caasCommand := range requestBodyData.CaasCommands {
		var errMsg string
		switch strings.TrimSpace(caasCommand) {
		case "get pods":
			errMsg = caasK8sInterface.GetPods(typeData.ServiceName, outputDir)
		case "get events":
			errMsg = caasK8sInterface.GetKubenetesResources("events", outputDir)
		case "get deployments":
			errMsg = caasK8sInterface.GetKubenetesResources("deployments", outputDir)
		case "get statefulsets":
			errMsg = caasK8sInterface.GetKubenetesResources("statefulsets", outputDir)
		case "get network":
			errMsg = caasK8sInterface.GetGroupResources(NETWORK_API_GROUP, NETWORK_RESOURCE, outputDir)
		case "top pod":
			errMsg = caasK8sInterface.GetPodMetrics(typeData.ServiceName, outputDir)
		case "logs":
			errMsg = caasK8sInterface.GetServiceLogs(typeData.ServiceName, outputDir)
		default:
			errorMsgs = append(errorMsgs, fmt.Sprintf("ERROR: Invalid Caas command: %s", caasCommand))
			continue
		}

		if errMsg != "" {
			errorMsgs = append(errorMsgs, fmt.Sprintf("Run command: %s failed: %s", caasCommand, errMsg))
			runCmdSuccess = false
		}
	}

	// 4. remove debug session directories which over the limit size: 500M
	RemoveSessionDirIfOverLimitSize(requestBodyData.SessionName)

	if !runCmdSuccess {
		if len(errorMsgs) == len(requestBodyData.CaasCommands) {
			debugLogger.Log.Error("--ERROR--ExeCaasLogCollection--: All commands running failed.")
			responseData.ResultCode = http.StatusInternalServerError
		} else {
			debugLogger.Log.Error("--ERROR--ExeCaasLogCollection--: Some commands running failed.")
			responseData.ResultCode = http.StatusPartialContent
		}
		responseData.ResponseMessage = fmt.Sprintf("Run commands failed: %s\n", strings.Join(errorMsgs, "\n"))
	}
	caasBuilder.WriteString("\n")
	responseData.BodyData = caasBuilder.String()
	caasBuilder = strings.Builder{}

	caasRespData <- responseData
}

