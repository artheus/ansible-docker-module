package main

import (
	"context"
	"encoding/json"

	common "github.com/artheus/ansible-docker-module"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/swarm"
)

type ConfigArguments struct {
	*common.DockerClientOpts

	State  string            `json:"state,omitempty"` // preset, absent, inspect
	Name   string            `json:"name"`
	Labels map[string]string `json:"labels"`
	Data   []byte            `json:"data,omitempty"`
	Src    string            `json:"src,omitempty"`
}

func newConfigArguments() *ConfigArguments {
	ca := new(ConfigArguments)
	ca.DockerClientOpts = common.NewDockerClientOpts()
	return ca
}

func (ca *ConfigArguments) toConfigSpec() swarm.ConfigSpec {
	cs := swarm.ConfigSpec{}
	cs.Name = ca.Name
	cs.Labels = ca.Labels
	cs.Data = ca.Data
	return cs
}

func main() {
	response := common.NewResponse()
	configArgs := newConfigArguments()

	common.DecorateArgumentStruct(configArgs, response)

	if configArgs.Name == "" {
		response.Msg = "name field cannot be empty, specify a docker config name"
		common.FailJson(response)
	}

	docker, err := common.GetDockerClient(configArgs.DockerClientOpts)
	if err != nil {
		response.Msg = err.Error()
		common.FailJson(response)
	}

	ctx := context.Background()

	options := types.ConfigListOptions{}
	options.Filters = filters.NewArgs(filters.Arg("name", "testconfig"))
	configs, err := docker.ConfigList(ctx, options)

	if len(configs) == 1 {
		switch configArgs.State {
		case "present":
			docker.ConfigUpdate(ctx, configs[0].ID, swarm.Version{Index: configs[0].Version.Index + 1}, configArgs.toConfigSpec())
		case "absent":
			err := docker.ConfigRemove(ctx, configs[0].ID)
			if err != nil {
				response.Msg = "Unable to remove config: " + err.Error()
				common.FailJson(response)
			}
			response.Msg = "Config removed successfully"
			common.ExitJson(response)
		case "inspect":
			inspect, data, err := docker.ConfigInspectWithRaw(ctx, configs[0].ID)
			if err != nil {
				response.Msg = "Unable to inspect config: " + err.Error()
				common.FailJson(response)
			}
			response.Info["id"] = inspect.ID
			response.Info["created_at"] = inspect.CreatedAt
			response.Info["updated_at"] = inspect.UpdatedAt
			response.Info["meta"] = inspect.Meta
			response.Info["spec"] = inspect.Spec
			response.Info["version"] = inspect.Version
			response.Info["contents"] = data
			common.ExitJson(response)
		}
	} else {
		if configArgs.State == "present" {
			docker.ConfigCreate(ctx, configArgs.toConfigSpec())
		}

		if configArgs.State == "inspect" {
			response.Msg = "Unable to inspect non-existant config"
			common.FailJson(response)
		}

		if configArgs.State == "absent" {
			response.Msg = "Config already removed"
			common.ExitJson(response)
		}
	}

	jsonBytes, err := json.Marshal(configs[0])
	println(string(jsonBytes))

	common.ExitJson(response)
}
