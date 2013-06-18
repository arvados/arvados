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

and put it in `config/initializers/secret_token.rb`

    Server::Application.config.secret_token = 'your-new-secret-here'

Adjust the following fields in your `environments/production.rb` file.

* `config.site_name` can be the URL of your workbench install.
* `config.arvados_login_base` and `config.arvados_v1_base` should point to
your API server. Use the example values as a guide.
* If you choose not to use https, make sure to also set
`config.force_ssl = false` in the API server's `production.rb` file.

## Apache/Passenger

Set up Apache and Passenger. Point them to the apps/workbench directory in the source tree.

## "Trusted client" setting

Log in to Workbench once (this ensures that the Arvados API server has
a record of the Workbench client).

In the API server project root, start the rails console.

    RAILS_ENV=production bundle exec rails c

Locate the ApiClient record for your Workbench installation.

    ApiClient.where('url_prefix like ?', '%workbench%')

Set the `is_trusted` flag for the appropriate client record.

    ApiClient.find(1234).update_attributes is_trusted: true

