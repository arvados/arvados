---
layout: default
navsection: userguide
title: "Intro: git"
navorder: 4
---

# Intro: git

Git is a revision control system. There are lots of resources for
learning about git (try [Try Git](http://try.git.io) or [Top 10 Git
Tutorials for
Beginners](http://sixrevisions.com/resources/git-tutorials-beginners/)). Here
we just cover the specifics of using git in the Arvados environment.

### Find your repository

Go to Workbench &rarr; Access &rarr; Repositories.

[https://workbench.{{ site.arvados_api_host }}/repositories](https://workbench.{{ site.arvados_api_host }}/repositories)

The "push_url" column will contain a string like `git@git.{{ site.arvados_api_host }}:example.git`.

This url can be used to pull and push commits between your Arvados
hosted repository and your VM/workstation.

### Make sure your SSH credentials are available

Git requires you to authenticate with your SSH private key. The best
way to make this happen from a VM is to use SSH agent forwarding.

When you log in to your VM, use the `-A` flag in your `ssh` command,
like this:

    ssh -A shell.q

At the shell prompt in the VM, type `ssh-add -l` to display a list of
keys that can be used. You should see something like this:

    2048 a7:f0:fb:ad:ba:66:fd:c2:8e:58:49:3b:6b:2a:1f:c3 example@host (RSA)

### Clone your repository

This step copies your Arvados-hosted repository to a new directory on
your VM.

Log in to your VM (using `ssh -A`!) and type:

    git clone git@git.{{ site.arvados_api_host }}:EXAMPLE.git

(Replace "EXAMPLE" with your own repository's name, or just copy the
usage example shown on the Repositories page.)

### Commit to your repository

This part works just like any other git tree.

    # (edit foo.txt)
    git add foo.txt
    git commit -m "Added section explaining what foo isn't"

### Push your commits to Arvados

From within your source tree, type:

    git push

