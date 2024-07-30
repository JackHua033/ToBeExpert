package commonData

import (
	"time"
)

type CertificateFilesData struct {
	CACERT_FILE string
	CERT_FILE string
	KEY_FILE string
}
//Data struct for debug request
type ContainersStruct struct {
	Name     string   `json:"name,omitempty"`
	Commands []string `json:"commands,omitempty"`
}

type SpecCommandsStruct struct {
	ServiceName string             `json:"serviceName,omitempty"`
	Containers  []ContainersStruct `json:"containers,omitempty"`
}

type TemplateStruct struct {
	SessionName    string               `json:"sessionName,omitempty"`
	CaasCommands   []string             `json:"caasCommands,omitempty"`
	Services       []string             `json:"services,omitempty"`
	LiServices     []string             `json:"liServices,omitempty"`
	LiLanNames     []string             `json:"liLanNames,omitempty"`
	LimitServices  []string             `json:"limitServices,omitempty"`
	CommonCommands []string             `json:"commonCommands,omitempty"`
	SpecCommands   []SpecCommandsStruct `json:"specCommands,omitempty"`
}

//Data struct for caas commands
type CaasCommandsAttr struct {
	Name    string   `json:"name"`
	Labels  []string `json:"labels"`
}

type CaasCommandsStruct struct {
	Services      []string           `json:"services"`
	LiServices    []string           `json:"liServices"`
	LiLanNames    []string           `json:"liLanNames"`
	LimitServices []string           `json:"limitServices"`
	CaasCommands  []CaasCommandsAttr `json:"caasCommands"`
}

//Data struct for commands
type CommandsAttrStruct struct {
	Name    string   `json:"name"`
	Path    string   `json:"path"`
	Labels  []string `json:"labels"`
	Options []string `json:"options"`
}

//Data struct for common commands
type CommonCommandsStruct struct {
	NativeCommands []CommandsAttrStruct `json:"nativeCommands"`
	ScriptCommands []CommandsAttrStruct `json:"scriptCommands"`
}

//Data struct for product commands
type ContainerCommands struct {
	Name     string               `json:"name"`
	Commands []CommandsAttrStruct `json:"commands"`
}

type SpecificCommand struct {
	ServiceName string              `json:"serviceName"`
	Containers  []ContainerCommands `json:"containers"`
}

type ProductCommandsStruct struct {
	NativeCommands []SpecificCommand `json:"nativeCommands"`
	ScriptCommands []SpecificCommand `json:"scriptCommands"`
}

//Data struct for CMA rest server
type ContainerStatusData struct {
	ContainerName string `json:"name"`
	Ready         string `json:"ready"`
	State         string `json:"state"`
}

type LabelsData struct {
	Name            string `json:"app.kubernetes.io/name"`
	PodTemplateHash string `json:"pod-template-hash"`
	ServiceType     string `json:"serviceType"`
	VnfMajorRelease string `json:"vnfMajorRelease"`
	VnfMinorRelease string `json:"vnfMinorRelease"`
	VnfName         string `json:"vnfName"`
	VnfType         string `json:"vnfType"`
	VnfcType        string `json:"vnfcType"`
}

type PodStatusData struct {
	PodName         string                `json:"podname"`
	Annotation      map[string]string     `json:"annotation"`
	Labels          LabelsData            `json:"labels"`
	HostIp          string                `json:"hostip"`
	Phase           string                `json:"phase"`
	PodIp           string                `json:"podip"`
	PodIps          []string              `json:"podips"`
	ContainerStatus []ContainerStatusData `json:"containerstatus"`
}

//Data struct for RCE server
type ContainersRceStruct struct {
	Name string `json:"name"`
	Port int    `json:"port"`
}

type RceServerStruct struct {
	ServiceName string                `json:"serviceName"`
	Containers  []ContainersRceStruct `json:"containers"`
}

type HttpServerResponse struct {
	ResultCode   int
	ResponseMessage string
	BodyData     string
}

type GetType struct {
	GenerateLabel []string
	ServiceName   []string
	Podid         []string
}

//Data struct for common config
type ContainerConf struct {
	Name		string `json:"name"`
	LimitSize	int    `json:"limitSizeInMBOfRemoveLog"`
	LogPath		string `json:"logPath"`
}

type ServiceConfig struct {
	ServiceName	string		`json:"serviceName"`
	Containers	[]ContainerConf	`json:"containers"`
}

type CommonConfigStruct struct {
	DelTime			int64	`json:"delTimeForRemoveLogs"`
	commandsTimeout		int64	`json:"commandsExecuteTimeout"`
	ConfigPerContainer	[]ServiceConfig `json:"configPerContainer"`
}

// Create a struct to hold the container information
type ContainerInfo struct {
	Name   string `json:"container_name"`
	Status string `json:"status"`
}

// Create a struct to hold the pod information
type PodInfo struct {
	Name        string           `json:"pod_name"`
	CreateTime  time.Time        `json:"create_time"`
	Containers  []ContainerInfo  `json:"containers"`
}

// Create a struct to hold the pod type
type PodType struct {
	Type    string
	PodList []PodInfo
}
