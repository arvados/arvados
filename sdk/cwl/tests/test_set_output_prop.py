# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

import arvados
import subprocess

api = arvados.api()

def test_execute():
    group = api.groups().create(body={"group": {"name": "test-17004-project", "group_class": "project"}}, ensure_unique_name=True).execute()
    try:
        contents = api.groups().contents(uuid=group["uuid"]).execute()
        if len(contents["items"]) != 0:
            raise Exception("Expected 0 items")

        cmd = ["arvados-cwl-runner", "--project-uuid", group["uuid"], "17004-output-props.cwl", "--inp", "scripts/download_all_data.sh"]
        print(" ".join(cmd))
        subprocess.check_output(cmd)

        contents = api.groups().contents(uuid=group["uuid"]).execute()

        found = False
        for c in contents["items"]:
            if (c["kind"] == "arvados#collection" and
                c["properties"].get("type") == "output" and
                c["properties"].get("foo") == "bar" and
                c["properties"].get("baz") == "download_all_data.sh"):
                found = True
        if not found:
            raise Exception("Didn't find collection with properties")

    finally:
        api.groups().delete(uuid=group["uuid"]).execute()

if __name__ == '__main__':
    test_execute()
