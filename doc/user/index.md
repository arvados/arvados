---
layout: default
navsection: userguide
title: Getting started
navorder: 0
---

> I'd like to see the user guide consist of the following sections:
> 1. background (general architecture/theory of operation from the user's perspective)
> 2. getting started / tutorials
> 3. how to (in depth topics)
> 4. tools reference (command line, workbench, etc)
> Currently the user guide is mostly just 2.

# Getting started

As a new user, you should take a quick tour of Arvados environment.


### Log in to Workbench

Open a browser and point it to the Workbench URL for your site. It
will look something like this:

`https://workbench.{{ site.arvados_api_host }}/`

Depending on site policy, a site administrator might have to activate
your account before you see any more good stuff.

> "Good stuff" is vague.

### Browse shared data angd pipelines

On the Workbench home page, you should see some datasets, programs,
jobs, and pipelines that you can explore.

> This would be a great place for a screenshot or at least a little
> more guidance on where to look (these things are all accessed
> through the menu bar)

### Install the command line SDK on your workstation

(Optional)

> Is this really optional?  All the tutorials are about how to use
> the command line SDK

Most of the functionality in Arvados is exposed by the REST API. This
means (depending on site policy and firewall) that you can do a lot of
stuff with the command line client and other SDKs running on your own
computer.

> "A lot of stuff" is vague.

Technically you can make all API calls using a generic web client like
[curl](http://curl.haxx.se/docs/) but you will have a more enjoyable
experience with the Arvados CLI client.

> I would mention this somewhere else, a new user isn't going to be using
> curl.

See [command line SDK](sdk-cli.html) for installation instructions.

### Request a virtual machine

> The purpose of this whole section is confusing, because after explaning that you
> can access arvados from your workstation with the client SDK, it then
> implies that you actually need to go and log into an arvados VM instance
> instead.

It's more fun to do stuff with a virtual machine, especially if you
know about [screen](http://www.gnu.org/software/screen/).

> Screen is cool, but not relevant here.

In order to get access to an Arvados VM, you need to:

1. Upload an SSH public key ([learn how](ssh-access.html))
1. Request a new VM (or access to a shared VM)

> Needs some kind of discussion on how to request a new VM or discover
> the hostname of the shared VM

