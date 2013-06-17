---
layout: default
navsection: userguide
title: Setting up SSH access
navorder: 2
---

# Setting up SSH access

Arvados uses SSH public key authentication for two things:

* logging in to a VM, and
* pushing code to a git repository.

### Generate a public/private SSH key pair

If you don't already have an SSH key pair (or you don't know whether
you do), there are lots of tutorials out there to help you get
started:

* [github SSH key
tutorial](https://www.google.com/search?q=github+ssh+key+help)

### Associate your SSH public key with your Arvados Workbench account

Go to the `Keys` tab in Arvados Workbench and click the

  `Add a new authorized key`

button. Then click on 'none' in the public_key column, and copy and paste your public key:

  ![Screen shot of the ssh public key box]({{ site.baseurl }}/images/ssh-adding-public-key.png)

Click on the checkmark button to save your public key.

### Unix-like systems: set up an ~/.ssh/config snippet for quick ssh access

On your workstation, add the following section to your `~/.ssh/config`
file:

    Host *.arvados
      ProxyCommand ssh turnout@9ujm1.arvados.org %h %p %u
      Port 2222

If you have access to an account `foo` on a VM called `blurfl` then
you can log in like this:

    ssh foo@blurfl.arvados

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

### Windows: Setup instructions for PuTTY

{% include alert-stub.html %}

If you use Microsoft Windows, you should download the PuTTY software.

* Details about configuring PuTTY would be nice here.
