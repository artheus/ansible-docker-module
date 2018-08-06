package main

import (
	common "github.com/artheus/ansible-docker-module"
)

type ConfigArguments struct {
	*common.DockerClientOpts
}

func newConfigArguments() *ConfigArguments {
	ba := new(ConfigArguments)
	ba.DockerClientOpts = common.NewDockerClientOpts()
	return ba
}

func main() {
	response := common.Response{}
	configArgs := newConfigArguments()

	// TODO: Implement this

	println(configArgs, response.Msg)
}
