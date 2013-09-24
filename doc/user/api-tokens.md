---
layout: default
navsection: userguide
title: Getting an API token
navorder: 1
---

# Getting an API token

Open a browser and point it to the Workbench URL for your site. It
will look something like this:

`https://workbench.{{ site.arvados_api_host }}/`

Log in, if you haven't done that already.

Click the "API tokens" link.

At the top of the "API tokens" page, you will see a few lines like this.

    ### Pasting the following lines at a shell prompt will allow Arvados SDKs
    ### to authenticate to your account, youraddress@example.com
    
    read ARVADOS_API_TOKEN <<EOF
    2jv9kd1o39t0pcfu7aueem7a1zjxhak73w90tzq3gx0es7j1ld
    EOF
    export ARVADOS_API_TOKEN ARVADOS_API_HOST=qr1hi.arvadosapi.com

Paste those lines into your terminal window to set up your
terminal. This effectively copies your credentials from your browser
to your terminal session.

Now, `arv -h user current` will display your account info in JSON
format.

Optionally, copy those lines to your .bashrc file so you don't have to
repeat this process each time you log in.

### SSL + development mode

If you are using a local development server with a self-signed
certificate, you might need to bypass certificate verification. Don't
do this if you are using a production service.

    export ARVADOS_API_HOST_INSECURE=yes
