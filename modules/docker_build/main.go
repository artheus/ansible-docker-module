package main

import (

	//"context"
	//"encoding/json"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strings"
	"sync"

	common "github.com/artheus/ansible-docker-module"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/pkg/archive"
	"github.com/docker/docker/pkg/jsonmessage"
	"gopkg.in/h2non/filetype.v1"
)

/*
examples:

- name: Build docker image
  docker_build:
    tags:
      - "imagename:1.0"
      - "imagename:latest"
      - "imagename:v1.0"
    nocache: yes
    dockerfile: "Dockerfile"
    buildargs:
      arg1: "first argument"
      something_else: "here it is"
      foo: "bar"
*/

type CpuArgument struct {
	SetCPUs string `json:"set_cpus,omitempty"`
	SetMems string `json:"set_mems,omitempty"`
	Shares  int64  `json:"shares,omitempty"`
	Quota   int64  `json:"quota,omitempty"`
	Period  int64  `json:"period,omitempty"`
}

type BuildArguments struct {
	*common.DockerClientOpts

	Tags          []string            `json:"tags,omitempty"`
	RemoteContext string              `json:"remote_context,omitempty"`
	NoCache       bool                `json:"no_cache,omitempty"`
	Remove        bool                `json:"remove,omitempty"`
	ForceRemove   bool                `json:"force_remove,omitempty"`
	PullParent    bool                `json:"pull_parent,omitempty"`
	Isolation     container.Isolation `json:"isolation,omitempty"`
	CPU           CpuArgument         `json:"cpu,omitempty"`
	Memory        int64               `json:"memory,omitempty"`
	MemorySwap    int64               `json:"memory_swap,omitempty"`
	CgroupParent  string              `json:"cgroup_parent,omitempty"`
	NetworkMode   string              `json:"network_mode,omitempty"`
	ShmSize       int64               `json:"shm_size,omitempty"`
	Dockerfile    string              `json:"dockerfile,omitempty"`
	//Ulimits       []*units.Ulimit             `json:"ulimits,omitempty"`
	BuildArgs   map[string]*string          `json:"build_args,omitempty"`
	AuthConfigs map[string]types.AuthConfig `json:"auth_configs,omitempty"`
	Src         string                      `json:"src"`
	Labels      map[string]string           `json:"labels,omitempty"`
	Squash      bool                        `json:"squash,omitempty"`
	CacheFrom   []string                    `json:"cache_from,omitempty"`
	SecurityOpt []string                    `json:"security_opt,omitempty"`
	ExtraHosts  []string                    `json:"extra_hosts,omitempty"`
	Target      string                      `json:"target,omitempty"`
	SessionID   string                      `json:"session_id,omitempty"`
	Platform    string                      `json:"platform,omitempty"`
}

func (s *BuildArguments) compile() (types.ImageBuildOptions, io.ReadCloser, error) {
	var bo types.ImageBuildOptions
	var buildCtx io.ReadCloser

	bo = types.ImageBuildOptions{}

	bo.SuppressOutput = false
	bo.Tags = s.Tags
	bo.RemoteContext = s.RemoteContext
	bo.NoCache = s.NoCache
	bo.Remove = s.Remove
	bo.ForceRemove = s.ForceRemove
	bo.PullParent = s.PullParent
	bo.Isolation = s.Isolation
	bo.CPUSetCPUs = s.CPU.SetCPUs
	bo.CPUSetMems = s.CPU.SetMems
	bo.CPUShares = s.CPU.Shares
	bo.CPUQuota = s.CPU.Quota
	bo.CPUPeriod = s.CPU.Period
	bo.Memory = s.Memory
	bo.MemorySwap = s.MemorySwap
	bo.CgroupParent = s.CgroupParent
	bo.NetworkMode = s.NetworkMode
	bo.ShmSize = s.ShmSize
	bo.Dockerfile = s.Dockerfile
	//bo.Ulimits = s.Ulimits
	bo.BuildArgs = s.BuildArgs
	bo.AuthConfigs = s.AuthConfigs
	bo.Labels = s.Labels
	bo.Squash = s.Squash
	bo.CacheFrom = s.CacheFrom
	bo.SecurityOpt = s.SecurityOpt
	bo.ExtraHosts = s.ExtraHosts
	bo.Target = s.Target
	bo.SessionID = s.SessionID
	bo.Platform = s.Platform

	fi, err := os.Stat(s.Src)
	if err != nil {
		return bo, buildCtx, err
	}

	switch mode := fi.Mode(); {
	case mode.IsDir():
		// Create tar.gz stream of directory
		r, err := archive.Tar(s.Src, archive.Gzip)
		if err != nil {
			return bo, buildCtx, err
		}
		buildCtx = r
	case mode.IsRegular():
		// Set context to file reader
		kind, err := filetype.MatchFile(s.Src)
		if err != nil {
			return bo, buildCtx, err
		}

		if kind.MIME.Value == "application/gzip" {
			file, err := os.Open(s.Src)
			if err != nil {
				return bo, buildCtx, err
			}
			buildCtx = file
		}
	}

	return bo, buildCtx, nil
}

func newBuildArguments() *BuildArguments {
	ba := new(BuildArguments)
	ba.DockerClientOpts = common.NewDockerClientOpts()
	return ba
}

func main() {
	response := common.NewResponse()
	buildArgs := newBuildArguments()

	common.DecorateArgumentStruct(buildArgs, response)

	if buildArgs.Src == "" {
		response.Msg = "src field cannot be empty, specify a directory or a tar.gz file"
		common.FailJson(response)
	}

	docker, err := common.GetDockerClient(buildArgs.DockerClientOpts)
	if err != nil {
		response.Msg = err.Error()
		common.FailJson(response)
	}

	buildOpts, buildContext, err := buildArgs.compile()
	if err != nil {
		response.Msg = err.Error()
		common.FailJson(response)
	}

	buildResponse, err := docker.ImageBuild(context.Background(), buildContext, buildOpts)
	if err != nil {
		response.Msg = err.Error()
		common.FailJson(response)
	}
	defer buildResponse.Body.Close()

	imageID := ""
	aux := func(msg jsonmessage.JSONMessage) {
		var result types.BuildResult
		err := json.Unmarshal(*msg.Aux, &result)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to parse aux message: %s", err)
		} else {
			imageID = result.ID
		}
	}

	r, w := io.Pipe()
	var wg sync.WaitGroup
	wg.Add(1)

	mw := io.MultiWriter(os.Stderr, w)
	go func() {
		defer wg.Done()
		defer w.Close()
		err := jsonmessage.DisplayJSONMessagesStream(buildResponse.Body, mw, os.Stdout.Fd(), true, aux)
		if err != nil {
			response.Msg = err.Error()
			common.FailJson(response)
		}
	}()

	dockerOutput, err := ioutil.ReadAll(r)
	if err != nil {
		response.Msg = err.Error()
		common.FailJson(response)
	}

	r.Close()
	wg.Wait()

	response.Info["image_id"] = imageID
	response.Info["stdout"] = string(dockerOutput)
	response.Info["stdout_lines"] = strings.Split(string(dockerOutput), "\n")
	if err != nil {
		response.Msg = err.Error()
		common.FailJson(response)
	}
	common.ExitJson(response)
}
