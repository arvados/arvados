---
layout: default
navsection: userguide
title: Setting up SSH keys
navorder: 2
---

# Setting up SSH keys

Arvados uses SSH public key authentication for two things:

* logging in to a VM, and
* pushing code to a git repository.

If you don't already have an SSH key pair (or you don't know whether
you do), there are lots of tutorials out there to help you get
started:

* [github SSH key
tutorial](https://www.google.com/search?q=github+ssh+key+help)

Once you have a public/private key pair, copy and paste the public key
into Arvados Workbench's "add SSH key" box.

* A screen shot would be nice here.

On your workstation, add the following section to your `~/.ssh/config`
file:

    Host *.arvados
      ProxyCommand ssh turnout@9ujm1.arvados.org %h %p %u
      Port 2222

If you have access to an account `foo` on a VM called `blurfl` then
you can log in like this:

    ssh foo@blurfl.arvados

### Another option for impatient and lazy people

If you want to shorten this and you always/usually have access to the
`foo` account on VMs, you can add a section like this to
`~/.ssh/config`:

    Host *.a
      ProxyCommand ssh turnout@9ujm1.arvados.org %hrvados %p %u
      Port 2222
      User foo

Then you can log in to the `blurfl` VM as `foo` like this:

    ssh blurfl.a

(Arvados Workbench will show you a list of VMs you have access to and
what your account name is for each one.)

### Setup instructions for PuTTY

If you use Microsoft Windows, you should download the PuTTY software.

* Details about configuring PuTTY would be nice here.
