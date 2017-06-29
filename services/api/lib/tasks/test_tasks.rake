# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

namespace :test do
  new_task = Rake::TestTask.new(tasks: "test:prepare") do |t|
    t.libs << "test"
    t.pattern = "test/tasks/**/*_test.rb"
  end
end
