---
layout: default
navsection: installguide
title: Install the Arvados workbench application
navorder: 1
---

{% include alert-stub.html %}

# Workbench setup

## Prerequisites

1. A GNU/linux (virtual) machine (can be shared with the API server)
2. A hostname for your workbench application

## Download the source tree

Please follow the instructions on the [Download page](https://arvados.org/projects/arvados/wiki/Download) in the wiki.

## Configure the Workbench application

You need to update config/initializers/secret_token.rb. Generate a new secret with

  rake secret

and put it in config/initializers/secret_token.rb:

  Server::Application.config.secret_token = 'your-new-secret-here'

Then edit your

    environments/production.rb

file. 

You will need to adjust the following fields 

    config.arvados_login_base
    config.arvados_v1_base
    config.site_name

The *config.site_name* can be set to the URL for your workbench install.

The *config.arvados_login_base* and *config.arvados_v1_base* fields should point to
your API server. Use the example values as a guide. If you choose not to use
https, make sure to also set *config.force_ssl* to false in the API server
production.rb.

## Apache/Passenger

Set up Apache and Passenger. Point them to the apps/workbench directory in the source tree.


