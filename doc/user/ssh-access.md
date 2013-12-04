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

> Maybe mention that the "Add a new authorized key" button will be at the bottom of the page


Go to the `Keys` page in Arvados Workbench (under the `Access` tab) and click the

<p style="margin-left: 4em"><span class="btn btn-primary disabled">Add a new authorized key</span></p>

button. Then click on 'none' in the public_key column, and copy and paste your public key:

![Screen shot of the ssh public key box]({{ site.baseurl }}/images/ssh-adding-public-key.png)

Click on the checkmark button to save your public key.

### Set up your ssh client (Unix-like systems only)

{% include notebox-begin.html %}

If you are using an Arvados site other than {{ site.arvados_api_host }}, replace "{{ site.arvados_api_host }}" with the appropriate Arvados API hostname in these examples.

{% include notebox-end.html %}

On your workstation, add the following section to your `~/.ssh/config`
file:

    Host *.arvados
      ProxyCommand ssh -p2222 turnout@switchyard.{{ site.arvados_api_host }} -x -a $SSH_PROXY_FLAGS %h

> This needs to be explained that it is adding an alias to make it easier to log into an
> arvados server on port 2222.  This is not actually necessary if the user doesn't mind some typing.
> Actually, it might make sense to show the regular command line first, and then mention later that
> it can be shortened using ~/.ssh/config. 

If you have access to an account `foo` on a VM called `blurfl` then
you can log in like this:

    ssh foo@blurfl.arvados

Some other convenient configuration options are `User` and
`ForwardAgent`:

    Host *.a
      ProxyCommand ssh -p2222 turnout@switchyard.{{ site.arvados_api_host }} -x -a $SSH_PROXY_FLAGS %h
      User foo
	  ForwardAgent yes

> This shortened *.arvados to *.a
> This should be consistent

Adding `User foo` will log you in to the VM as user `foo` by default,
so you can just `ssh blurfl.a`. The `ForwardAgent yes` option turns on
the `ssh -A` option to forward your SSH credentials (if you are
using ssh-agent), which becomes important if you use git to
synchronize files between your workstation and the VM.

Then you can log in to the `blurfl` VM as `foo` like this:

    ssh blurfl.a

Arvados Workbench will show you a list of VMs you have access to and
what your account name is for each one: click "VMs" in the "Access"
menu.

### Windows: Setup instructions for PuTTY

PuTTY is a free (MIT-licensed) Win32 Telnet and SSH client. PuTTy includes all the tools a windows user needs to set up Private Keys and to set up and use SSH connections to your virtual machines in the Arvados Cloud. 

You can use PuTTY to create public/private keys, which are how you’ll ensure that that access to Arvados cloud is secure. You can also use PuTTY as an SSH client to access your virtual machine in an Arvados cloud and work with the Arvados Command Line Interface (CLI) client. 

PuTTY is an open source project and you download it [here](http://www.putty.org/).

Arvados uses Public-key encryption to secure access to your virtual machines in the Arvados cloud. this is a very standard approach. It’s secure, and easy to use. 

(Make sure to download the .zip file containing all the binaries, not each one individually)

__Step 1 - Adding PuTTY to the PATH__

1. After downloading PuTTY and unzipping it, you should have a PuTTY folder in C:\Program Files (x86)\ . If the folder is somewhere else, you can change the PATH in step X or move the folder to that directory.

2. In the Start menu, right click Windows and select Properties

3. Select Advanced System Settings, and choose Environment Variables

4. Under system variables, find and edit Path.

5. Add the following to the end of Path (make sure to include semi colon and quotation marks): 

	;\"C:\Program Files (x86)\PuTTY\"

6. Click through the OKs to close all the dialogs you’ve opened

__Step 2 - Creating a Public Key__

1. Open PuTTYgen from the Start Menu

2. At the bottom of the window, make sure the ‘Number of bits in a generated key’ field is set to 4096

3. Click Generate and follow the instructions to generate a key

4. Click to save the Public Key 

5. Click to save the Private Key (we recommend using a strong passphrase) 

6. Select the Public Key text in the box and copy (for next step) 

Now your key is successfully generated. 

__Step 3 - Load Your Public Key in to your Arvados Account through Workbench__

1. Open Workbench on the cloud where you have an arvados account

2. Go to Access > Keys in the menu 

3. Click to create a new key 

4. In the last column “public key” click on the text that says “none” and paste the public key from PuTTYgen into the box. 

Your public key is now registered with the Arvados cluster. 

__Step 4 - Set up Pageant__

1. Start Pageant from the PuTTY folder in the start menu 

2. Pageant will now be running in the system tray. Click the icon to configure. 

3. Choose Add Key and add the private which corresponds with the public key you loaded in your Arvados account through work bench. 

Pageant is now configured. It will run in the background as a system service. 

Note: Pageant is a PuTTY utility that manages private keys which makes repeatedly logging in through SSH less of a hassle. 

__Step 5 - Set up PuTTY__

1. Open PuTTY from the Start Menu

2. On the Session screen set the Host Name (or IP address) to “shell” 

3. On the Session screen set the Port to “22”
 
4. On the Connection > Data screen set the Auto-login username to your VM’s Login,. You can find your login name in Workbench under Access > VMs last column on the table. 

5. On the Connection > Proxy screen set the Proxy Type to “Local” 

6. On the Connection > Proxy screen in the “Telnet command, or local proxy command” box enter “plink -P 2222 turnout@switchyard.qr1hi.arvadosapi.com %host”. Make sure you remove the “\n” from the end of the line.

7. Return to the Session screen. In the Saved Sessions box, enter a name for this configuration and hit Save. 


__Step 6 - Launch an SSH Session__

1. Open PuTTY 

2. Click on the Saved Session name you created in Step 5

3. Click Load to load those saved session settings

4. Click Open and that will open the SSH window at the command prompt. You will now be logged in to your virtual machine. 

_Note: We recommend you do not delete the “Default” Saved Session._

