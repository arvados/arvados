---
layout: default
navsection: installguide
title: Install the API server
navorder: 1
---

{% include alert-stub.html %}

# API server setup

## Prerequisites

1. A GNU/linux (virtual) machine
2. A domain name for your api server

## Download the source tree

Please follow the instructions on the [Download page](https://arvados.org/projects/arvados/wiki/Download) in the wiki.

## Configure the API server

First configure the database:

    cp config/database.yml.sample config/database.yml

Edit database.yml to your liking and make sure the database and db user exist.
Then set up the database:
 
    RAILS_ENV=production rake db:setup

Then set up omniauth:

    cp config/initializers/omniauth.rb.example config/initializers/omniauth.rb

Edit config/initializers/omniauth.rb. Choose an APP_SECRET and APP_ID. Also set
CUSTOM_PROVIDER_URL.

Make sure your Omniauth provider knows about your APP_ID and APP_SECRET
combination.

Finally, edit your

    environments/production.rb

file. Specifically, you want to make sure that 

    config.uuid_prefix

is set to a unique 5-digit hex string. You can replace the 'cfi-aws-0' string
with a string of your choice to make that happen.

The config.uuid_prefix string is a unique identifier for your API server. It
also serves as the first part of the hostname for your API server, for instance

    9ujm1.arvadosapi.com

You should use your own domain instead of arvadosapi.com

## Apache/Passenger

Set up Apache and Passenger. Point them to the services/api directory in the source tree.

## Add an admin user

Point browser to the API endpoint. Log in with a google account.

In the rails console:

    Thread.current[:user] = User.find(1)
    Thread.current[:user].is_admin = true
    User.find(1).update_attributes is_admin: true
    User.find(1).is_admin

This should be

     => true

## Create a token

In rails console

     a = ApiClient.new(owner:1); a.save!
     x = ApiClientAuthorization.new(api_client_id:a.id, user_id:1); x.save; x.api_token

