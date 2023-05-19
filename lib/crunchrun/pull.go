// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0
package crunchrun

import (
	"context"
	"fmt"
	"io"
	"os"

	"git.arvados.org/arvados.git/sdk/go/arvados"
)

func (runner *ContainerRunner) runBuiltinCommand() error {
	err := runner.UpdateContainerRunning("")
	if err != nil {
		return err
	}
	exitCode := 1
	runner.ExitCode = &exitCode
	runner.finalState = string(arvados.ContainerStateComplete)
	if len(runner.Container.Command) == 3 && runner.Container.Command[0] == "docker" && runner.Container.Command[1] == "pull" {
		repotag := runner.Container.Command[2]
		outcoll, err := pullImageAndSaveCollection(runner.Container.UUID, runner.executor, repotag, runner.containerClient, runner.ContainerKeepClient)
		if err != nil {
			return err
		}
		runner.OutputPDH = &outcoll.PortableDataHash
		exitCode = 0
		return nil
	}
	return fmt.Errorf("unsupported builtin command %v", runner.Container.Command)
}

func pullImageAndSaveCollection(ctrUUID string, executor containerExecutor, repotag string, arvClient *arvados.Client, keepClient IKeepClient) (outcoll arvados.Collection, err error) {
	outfs, err := outcoll.FileSystem(arvClient, keepClient)
	if err != nil {
		return outcoll, fmt.Errorf("error creating filesystem: %w", err)
	}

	imagedata, imagehash, err := executor.PullImage(context.TODO(), repotag)
	if err != nil {
		return outcoll, fmt.Errorf("error pulling image: %w", err)
	}
	tarfile, err := outfs.OpenFile(imagehash+".tar", os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0777)
	if err != nil {
		return outcoll, fmt.Errorf("error opening file to save image: %w", err)
	}
	defer tarfile.Close()

	_, err = io.Copy(tarfile, imagedata)
	if err != nil {
		return outcoll, fmt.Errorf("error saving image data: %w", err)
	}
	err = imagedata.Close()
	if err != nil {
		return outcoll, fmt.Errorf("error closing image data reader: %w", err)
	}
	err = tarfile.Close()
	if err != nil {
		return outcoll, fmt.Errorf("error closing image file: %w", err)
	}
	outcoll.ManifestText, err = outfs.MarshalManifest(".")
	if err != nil {
		return outcoll, fmt.Errorf("error saving image collection manifest: %w", err)
	}
	err = arvClient.RequestAndDecode(&outcoll, "POST", "arvados/v1/collections", nil, map[string]interface{}{
		"collection": map[string]interface{}{
			"manifest_text": outcoll.ManifestText,
			"is_trashed":    true,
		}})
	if err != nil {
		return outcoll, fmt.Errorf("error saving image collection: %w", err)
	}
	// Now we update the container properties with the repo:tag
	// and hash. RailsAPI will use these values when creating the
	// container request output collection during finalize.
	//
	// Additionally, RailsAPI has "arvados/builtin"-specific code
	// to create a tag link with these values, pointing to the
	// CR's output collection.
	err = arvClient.RequestAndDecode(nil, "PATCH", "arvados/v1/containers/"+ctrUUID, nil, map[string]interface{}{
		"container": map[string]interface{}{
			"output_properties": map[string]interface{}{
				"docker-image-repo-tag": repotag,
				"docker-image-hash":     imagehash,
			}}})
	if err != nil {
		return outcoll, err
	}
	return outcoll, nil
}
