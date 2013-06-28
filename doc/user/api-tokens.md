---
layout: default
navsection: userguide
title: Getting an API token
navorder: 1
---

# API authentication

Open a browser and point it to the Workbench URL for your site. It
will look something like this:

`https://workbench.{{ site.arvados_api_host }}/`

Log in, if you haven't done that already.

Click the "API tokens" link.

Copy an API token and set environment variables in your terminal
session like this.

    export ARVADOS_API_TOKEN=unvz7ktg5p5k2q4wb9hpfl9fkge96rvv1j1gjpiq
    export ARVADOS_API_HOST={{ site.arvados_api_host }}

If you are using a local development server with a self-signed
certificate, you might need to bypass certificate verification. Don't
do this if you are using a production service.

    export ARVADOS_API_HOST_INSECURE=yes
