#!/usr/bin/env python

import os
import sys
import subprocess
import tempfile
import shutil
import argparse

import arvados
import arvados.commands.keepdocker as keepdocker
import arvados.commands.arv_copy as arv_copy
import arvados_cwl

def registerDocker((name, prefix, branch), dirname, names):
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

def registerCWL(prefix, dirname, names):
    cwlfile = None
    for c in ("CWLFile", "Dockstore.cwl"):
        if c in names:
            cwlfile = c
            break
    if cwlfile is None:
        return
    arvados_cwl.main(["--create-workflow", cwlfile], sys.stdout, sys.stderr)

def gitclone(api, repo):
    (src_git_url, src_git_config) = arv_copy.select_git_url(api, repo, 3, True, "--insecure-http")

    tempdir = tempfile.mkdtemp()
    arvados.util.run_command(
        ["git"] + src_git_config + ["clone", src_git_url, tempdir],
        cwd=os.path.dirname(tempdir),
        env={"HOME": os.environ["HOME"],
             "ARVADOS_API_TOKEN": arvados.config.get("ARVADOS_API_TOKEN"),
             "GIT_ASKPASS": "/bin/false"})
    return tempdir

def handle(name, ab, branch):
    subprocess.check_call(["git", "checkout", branch])
    os.path.walk(ab, registerDocker, (name, ab, branch))
    os.path.walk(ab, registerCWL, (name, ab, branch))

def main(argv):
    api = arvados.api("v1")

    parser = argparse.ArgumentParser()
    parser.add_argument('repo', nargs="?", default=None)
    parser.add_argument('branch', nargs="?", default=None)
    args = parser.parse_args(argv)

    if args.repo is None:
        items = api.repositories().list().execute()["items"]
    else:
        items = api.repositories().list(filters=[["name", "=", args.repo]]).execute()["items"]

    for i in items:
        name = i["name"]
        if '/' not in name:
            continue
        ab = gitclone(api, name)
        os.chdir(ab)
        try:
            if args.branch is None:
                for l in subprocess.check_output(["git", "branch", "--list"]).split("\n"):
                    l = l[2:]
                    if l:
                        handle(name, ab, l)
            else:
                handle(name, ab, args.branch)

        finally:
            shutil.rmtree(ab)
            os.chdir('/')

if __name__ == '__main__':
    main(sys.argv[1:])
