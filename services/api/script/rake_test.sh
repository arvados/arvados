#! /bin/sh
# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

# This script invokes `rake test' in a fresh Docker instance of the
# API server, e.g.:
#   docker run -t -i arvados/api /usr/src/arvados/services/api/script/rake_test.sh

/etc/init.d/postgresql start

export RAILS_ENV=test
cd /usr/src/arvados/services/api
cp config/environments/test.rb.example config/environments/test.rb
bundle exec rake db:setup
bundle exec rake test
