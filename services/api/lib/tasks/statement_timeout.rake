# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

Rake::Task.tasks.each do |task|
  if task.name =~ /^(db:migrate(:.*)?|db:rollback)$/
    task.enhance(["db:disable_timeout"])
  end
end

namespace :db do
  desc 'disable postgresql statement_timeout and lock_timeout before running migrations'
  task disable_timeout: :environment do
    ActiveRecord::ConnectionAdapters::AbstractAdapter.set_callback :checkout, :before, ->(conn) do
      # override the default timeouts set by
      # config/initializers/db_timeout.rb
      conn.execute "SET statement_timeout = 0"
      conn.execute "SET lock_timeout = 0"
    end
  end
end
