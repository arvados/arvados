#!/usr/bin/env python

import os
import sys
import subprocess
import tempfile
import shutil
import argparse
import StringIO
import urlparse
import logging

import arvados
import arvados.commands.keepdocker as keepdocker
import arvados.commands.arv_copy as arv_copy
import arvados_cwl

_logger = logging.getLogger("workflowimporter")

defaultStreamHandler = logging.StreamHandler()
_logger.addHandler(defaultStreamHandler)
_logger.setLevel(logging.INFO)


def registerDocker((api, reporecord, prefix, branch), dirname, names):
    sp = urlparse.urlsplit(reporecord["name"])
    name = sp.path
    if name.startswith("/"):
        name = name[1:]
    if name.endswith(".git"):
        name = name[0:-4]
    name = "/".join(name.split("/")[-2:])

    dockerfile = None
    if "Dockerfile" in names:
        dockerfile = "Dockerfile"
    if dockerfile is None:
        return
    tagname = dirname[len(prefix)+1:].replace("/", "-")
    if tagname:
        tagname = name+"-"+tagname
    else:
        tagname = name
    os.chdir(dirname)
    subprocess.check_call(["docker", "build", "--tag=%s:%s" % (tagname, branch), "."])
    try:
        keepdocker.main([tagname, branch])
    except SystemExit:
        pass

def registerCWL((api, reporecord, prefix, branch), dirname, names):
    cwlfile = None
    for c in ("CWLFile", "Dockstore.cwl"):
        if c in names:
            cwlfile = c
            break
    if cwlfile is None:
        return

    items = api.links().list(filters=[["link_class", "=", "workflow-import"],
                                      ["tail_uuid", "=", reporecord["uuid"]],
                                      ["name", "=", branch]]).execute()["items"]

    uuid = items[0]["head_uuid"] if items else None
    stdout = StringIO.StringIO()
    stderr = StringIO.StringIO()
    os.chdir(dirname)
    if uuid:
        rval = arvados_cwl.main(["--update-workflow="+uuid, cwlfile], stdout, stderr)
        if rval != 0:
            raise Exception(stderr.getvalue())
        _logger.info("Updated workflow %s", uuid)
    else:
        rval = arvados_cwl.main(["--create-workflow", cwlfile], stdout, stderr)
        if rval != 0:
            raise Exception(stderr.getvalue())
        wf = api.links().create(body={"link_class": "workflow-import",
                                      "tail_uuid": reporecord["uuid"],
                                      "head_uuid": stdout.getvalue().strip(),
                                      "name": branch}).execute()
        -logger.info("Created workflow %s", stdout.getvalue().strip())

def gitclone(api, repo, insecure_http):
    if not (repo.startswith("http://") or repo.startswith("https://") or repo.startswith("/")):
        (src_git_url, src_git_config) = arv_copy.select_git_url(api, repo, 3, insecure_http, "--insecure-http")
    else:
        src_git_config = []
        src_git_url = repo

    tempdir = tempfile.mkdtemp()
    arvados.util.run_command(
        ["git"] + src_git_config + ["clone", src_git_url, tempdir],
        cwd=os.path.dirname(tempdir),
        env={"HOME": os.environ["HOME"],
             "ARVADOS_API_TOKEN": arvados.config.get("ARVADOS_API_TOKEN"),
             "GIT_ASKPASS": "/bin/false"})
    return tempdir

def handle(api, name, ab, branch):
    subprocess.check_call(["git", "checkout", branch])
    os.path.walk(ab, registerDocker, (api, name, ab, branch))
    os.path.walk(ab, registerCWL, (api, name, ab, branch))

def main(argv):
    api = arvados.api("v1")

    parser = argparse.ArgumentParser()
    parser.add_argument('--insecure-http', default=False, action="store_true")
    parser.add_argument('repo', nargs="?", default=None)
    parser.add_argument('branch', nargs="?", default="master")
    args = parser.parse_args(argv)

    if args.repo is None:
        items = api.repositories().list().execute()["items"]
    else:
        sp = urlparse.urlsplit(args.repo)
        if sp.scheme:
            items = [{"name": args.repo, "uuid": None}]
        elif os.path.isdir(args.repo):
            items = [{"name": os.path.abspath(args.repo), "uuid": None}]
        else:
            items = api.repositories().list(filters=[["name", "=", args.repo]]).execute()["items"]

    for i in items:
        name = i["name"]
        if '/' not in name:
            continue
        ab = gitclone(api, name, args.insecure_http)
        os.chdir(ab)
        try:
            if args.branch is None:
                for l in subprocess.check_output(["git", "branch", "--list"]).split("\n"):
                    l = l[2:]
                    if l:
                        handle(api, i, ab, l)
            else:
                handle(api, i, ab, args.branch)

        finally:
            shutil.rmtree(ab)
            os.chdir('/')

if __name__ == '__main__':
    main(sys.argv[1:])
