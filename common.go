package common

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"os"

	"github.com/docker/docker/client"
)

type ModuleArgs struct {
	Name string
}

type Response struct {
	Msg     string                 `json:"msg"`
	Changed bool                   `json:"changed"`
	Failed  bool                   `json:"failed"`
	Info    map[string]interface{} `json:"info,omitempty"`
}

func NewResponse() Response {
	r := Response{}
	r.Info = make(map[string]interface{})
	return r
}

type DockerClientOpts struct {
	Host        string            `json:"host,omitempty"`
	Version     string            `json:"version,omitempty"`
	HttpClient  *http.Client      `json:"http_client,omitempty"`
	HttpHeaders map[string]string `json:"http_headers,omitempty"`
}

func NewDockerClientOpts() *DockerClientOpts {
	dco := new(DockerClientOpts)
	return dco
}

func ExitJson(responseBody Response) {
	returnResponse(responseBody)
}

func FailJson(responseBody Response) {
	responseBody.Failed = true
	returnResponse(responseBody)
}

func returnResponse(responseBody Response) {
	var response []byte
	var err error
	response, err = json.Marshal(responseBody)
	if err != nil {
		response, _ = json.Marshal(Response{Msg: "Invalid response object", Failed: true})
	}
	os.Stdout.Write(response)
	os.Stdout.Write([]byte("\n"))
	if responseBody.Failed {
		os.Exit(1)
	} else {
		os.Exit(0)
	}
}

func DecorateArgumentStruct(obj interface{}, response Response) {
	if len(os.Args) != 2 {
		response.Msg = "No argument file provided"
		FailJson(response)
	}

	argsFile := os.Args[1]

	text, err := ioutil.ReadFile(argsFile)
	if err != nil {
		response.Msg = "Could not read configuration file: " + argsFile
		FailJson(response)
	}

	err = json.Unmarshal(text, &obj)
	if err != nil {
		response.Msg = "Configuration file not valid JSON: " + argsFile
		FailJson(response)
	}
}

func GetDockerClient(opts *DockerClientOpts) (*client.Client, error) {
	clientOptions := []func(*client.Client) error{}

	if opts.Host != "" {
		clientOptions = append(clientOptions, client.WithHost(opts.Host))
	}

	if opts.Version != "" {
		clientOptions = append(clientOptions, client.WithVersion(opts.Version))
	}

	if opts.HttpClient != nil {
		clientOptions = append(clientOptions, client.WithHTTPClient(opts.HttpClient))
	}

	if opts.HttpHeaders != nil {
		clientOptions = append(clientOptions, client.WithHTTPHeaders(opts.HttpHeaders))
	}

	cli, err := client.NewClientWithOpts(clientOptions...)

	if err != nil {
		return nil, err
	}

	return cli, nil
}
