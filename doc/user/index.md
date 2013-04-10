---
layout: default
navsection: userguide
title: Getting started
navorder: 0
---

# Getting started

As a new user, you should take a quick tour of Arvados environment.


### Log in to Workbench

Open a browser and point it to the Workbench URL for your site. It
will look something like this:

`https://workbench.a123z.arvados.org/`

Depending on site policy, a site administrator might have to activate
your account before you see any more good stuff.

### Browse shared data and pipelines

On the Workbench home page, you should see some datasets, programs,
jobs, and pipelines that you can explore.

### Install the command line SDK on your workstation

(Optional)

Most of the functionality in Arvados is exposed by the REST API. This
means (depending on site policy and firewall) that you can do a lot of
stuff with the command line client and other SDKs running on your own
computer.

Technically you can make all API calls using a generic web client like
[curl](http://curl.haxx.se/docs/) but you will have a more enjoyable
experience with the Arvados CLI client.

See [command line SDK](sdk-cli.html) for installation instructions.

### Request a virtual machine

It's more fun to do stuff with a virtual machine, especially if you
know about [screen](http://www.gnu.org/software/screen/).

In order to get access to an Arvados VM, you need to:

1. Upload an SSH public key ([learn how](ssh-access.html))
1. Request a new VM (or access to a shared VM)

