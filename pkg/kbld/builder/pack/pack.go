// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package image

import (
	"bytes"
	"fmt"
	"io"
	"os/exec"
	"regexp"

	ctlbdk "github.com/vmware-tanzu/carvel-kbld/pkg/kbld/builder/docker"
	ctllog "github.com/vmware-tanzu/carvel-kbld/pkg/kbld/logger"
)

var (
	// Example output that includes final digest:
	// [exporter] *** Images:
	// [exporter]       index.docker.io/library/myapp:latest - succeeded
	// [exporter] *** Image ID: 2be602fbc1ecffdf9cc1c8ccb1f1cd6fb1d0a2e76dccbfcc34898bf35c836beb
	// Image ID is printed here: https://github.com/buildpack/lifecycle/blob/4e449525af56096f7cf8a521900bf6216467f0d7/save.go#L39
	packImageID = regexp.MustCompile("Image ID: (sha256:)?([0-9a-z]+)")
)

type Pack struct {
	docker ctlbdk.Docker
	logger ctllog.Logger
}

type PackBuildOpts struct {
	Builder    *string
	Buildpacks *[]string
	ClearCache *bool
	RawOptions *[]string // pack build -h
}

func NewPack(docker ctlbdk.Docker, logger ctllog.Logger) Pack {
	return Pack{docker, logger}
}

func (d Pack) Build(image, directory string, opts PackBuildOpts) (ctlbdk.DockerTmpRef, error) {
	prefixedLogger := d.logger.NewPrefixedWriter(image + " | ")

	prefixedLogger.Write([]byte(fmt.Sprintf("starting build (using pack): %s\n", directory)))
	defer prefixedLogger.Write([]byte("finished build (using pack)\n"))

	var imageID string

	{
		var stdoutBuf, stderrBuf bytes.Buffer

		// --verbose is necessary for Image ID to be displayed
		cmdArgs := []string{"build", "--verbose", image, "--path", "."}

		if opts.Builder == nil {
			return ctlbdk.DockerTmpRef{}, fmt.Errorf("Expected builder to be specified, but was not")
		}
		cmdArgs = append(cmdArgs, "--builder", *opts.Builder)

		if opts.Buildpacks != nil {
			for _, b := range *opts.Buildpacks {
				cmdArgs = append(cmdArgs, []string{"--buildpack", b}...)
			}
		}
		if opts.ClearCache != nil && *opts.ClearCache {
			cmdArgs = append(cmdArgs, "--clear-cache")
		}
		if opts.RawOptions != nil {
			cmdArgs = append(cmdArgs, *opts.RawOptions...)
		}

		cmd := exec.Command("pack", cmdArgs...)
		cmd.Dir = directory
		cmd.Stdout = io.MultiWriter(&stdoutBuf, prefixedLogger)
		cmd.Stderr = io.MultiWriter(&stderrBuf, prefixedLogger)

		err := cmd.Run()
		if err != nil {
			prefixedLogger.Write([]byte(fmt.Sprintf("error: %s\n", err)))
			return ctlbdk.DockerTmpRef{}, err
		}

		matches := packImageID.FindStringSubmatch(stdoutBuf.String())
		if len(matches) != 3 {
			return ctlbdk.DockerTmpRef{}, fmt.Errorf("Expected to find image ID in pack output but did not")
		}

		imageID = "sha256:" + matches[2]
	}

	return d.docker.RetagStable(ctlbdk.NewDockerTmpRef(imageID), image, imageID, prefixedLogger)
}

func (d Pack) Push(tmpRef ctlbdk.DockerTmpRef, imageDst string) (ctlbdk.DockerImageDigest, error) {
	return d.docker.Push(tmpRef, imageDst)
}
