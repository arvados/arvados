# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

import arvados
import subprocess

api = arvados.api()

def check_contents(group, wf_uuid):
    contents = api.groups().contents(uuid=group["uuid"]).execute()
    if len(contents["items"]) != 4:
        raise Exception("Expected 4 items in "+group["uuid"]+" was "+str(len(contents["items"])))

    found = False
    for c in contents["items"]:
        if c["kind"] == "arvados#workflow" and c["uuid"] == wf_uuid:
            found = True
    if not found:
        raise Exception("Couldn't find workflow in "+group["uuid"])

    found = False
    for c in contents["items"]:
        if c["kind"] == "arvados#collection" and c["portable_data_hash"] == "d7514270f356df848477718d58308cc4+94":
            found = True
    if not found:
        raise Exception("Couldn't find collection dependency")

    found = False
    for c in contents["items"]:
        if c["kind"] == "arvados#collection" and c["name"].startswith("Docker image arvados jobs"):
            found = True
    if not found:
        raise Exception("Couldn't find jobs image dependency")

    found = False
    for c in contents["items"]:
        if c["kind"] == "arvados#collection" and c["portable_data_hash"] == "13d3901489516f9986c9685867043d39+61":
            found = True
    if not found:
        raise Exception("Couldn't find collection containing workflow")


def test_create():
    group = api.groups().create(body={"group": {"name": "test-19070-project-1", "group_class": "project"}}, ensure_unique_name=True).execute()
    try:
        contents = api.groups().contents(uuid=group["uuid"]).execute()
        if len(contents["items"]) != 0:
            raise Exception("Expected 0 items")

        # Create workflow, by default should also copy dependencies
        cmd = ["arvados-cwl-runner", "--create-workflow", "--project-uuid", group["uuid"], "19070-copy-deps.cwl"]
        print(" ".join(cmd))
        wf_uuid = subprocess.check_output(cmd)
        wf_uuid = wf_uuid.decode("utf-8").strip()
        check_contents(group, wf_uuid)
    finally:
        api.groups().delete(uuid=group["uuid"]).execute()


def test_update():
    group = api.groups().create(body={"group": {"name": "test-19070-project-2", "group_class": "project"}}, ensure_unique_name=True).execute()
    try:
        contents = api.groups().contents(uuid=group["uuid"]).execute()
        if len(contents["items"]) != 0:
            raise Exception("Expected 0 items")

        # Create workflow, but with --no-copy-deps it shouldn't copy anything
        cmd = ["arvados-cwl-runner", "--no-copy-deps", "--create-workflow", "--project-uuid", group["uuid"], "19070-copy-deps.cwl"]
        print(" ".join(cmd))
        wf_uuid = subprocess.check_output(cmd)
        wf_uuid = wf_uuid.decode("utf-8").strip()

        contents = api.groups().contents(uuid=group["uuid"]).execute()
        if len(contents["items"]) != 2:
            raise Exception("Expected 2 items")

        found = False
        for c in contents["items"]:
            if c["kind"] == "arvados#workflow" and c["uuid"] == wf_uuid:
                found = True
        if not found:
            raise Exception("Couldn't find workflow")

        found = False
        for c in contents["items"]:
            if c["kind"] == "arvados#collection" and c["portable_data_hash"] == "13d3901489516f9986c9685867043d39+61":
                found = True
        if not found:
            raise Exception("Couldn't find collection containing workflow")

        # Updating by default will copy missing items
        cmd = ["arvados-cwl-runner", "--update-workflow", wf_uuid, "19070-copy-deps.cwl"]
        print(" ".join(cmd))
        wf_uuid = subprocess.check_output(cmd)
        wf_uuid = wf_uuid.decode("utf-8").strip()
        check_contents(group, wf_uuid)

    finally:
        api.groups().delete(uuid=group["uuid"]).execute()


def test_execute():
    group = api.groups().create(body={"group": {"name": "test-19070-project-3", "group_class": "project"}}, ensure_unique_name=True).execute()
    try:
        contents = api.groups().contents(uuid=group["uuid"]).execute()
        if len(contents["items"]) != 0:
            raise Exception("Expected 0 items")

        # Execute workflow, shouldn't copy anything.
        cmd = ["arvados-cwl-runner", "--project-uuid", group["uuid"], "19070-copy-deps.cwl"]
        print(" ".join(cmd))
        wf_uuid = subprocess.check_output(cmd)
        wf_uuid = wf_uuid.decode("utf-8").strip()

        contents = api.groups().contents(uuid=group["uuid"]).execute()
        # container request
        # final output collection
        # container log
        # step output collection
        # container request log
        if len(contents["items"]) != 5:
            raise Exception("Expected 5 items")

        found = False
        for c in contents["items"]:
            if c["kind"] == "arvados#collection" and c["portable_data_hash"] == "d7514270f356df848477718d58308cc4+94":
                found = True
        if found:
            raise Exception("Didn't expect to find collection dependency")

        found = False
        for c in contents["items"]:
            if c["kind"] == "arvados#collection" and c["name"].startswith("Docker image arvados jobs"):
                found = True
        if found:
            raise Exception("Didn't expect to find jobs image dependency")

        # Execute workflow with --copy-deps
        cmd = ["arvados-cwl-runner", "--project-uuid", group["uuid"], "--copy-deps", "19070-copy-deps.cwl"]
        print(" ".join(cmd))
        wf_uuid = subprocess.check_output(cmd)
        wf_uuid = wf_uuid.decode("utf-8").strip()

        contents = api.groups().contents(uuid=group["uuid"]).execute()
        found = False
        for c in contents["items"]:
            if c["kind"] == "arvados#collection" and c["portable_data_hash"] == "d7514270f356df848477718d58308cc4+94":
                found = True
        if not found:
            raise Exception("Couldn't find collection dependency")

        found = False
        for c in contents["items"]:
            if c["kind"] == "arvados#collection" and c["name"].startswith("Docker image arvados jobs"):
                found = True
        if not found:
            raise Exception("Couldn't find jobs image dependency")

    finally:
        api.groups().delete(uuid=group["uuid"]).execute()

if __name__ == '__main__':
    test_create()
    test_update()
    test_execute()
