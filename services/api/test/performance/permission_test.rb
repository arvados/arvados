# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

require 'test_helper'
require 'benchmark'


def create_eight parent
  uuids = []
  values = []
  (0..8).each do
    uuid = Group.generate_uuid
    values.push "('#{uuid}', '#{parent}', now(), now(), '#{uuid}')"
    uuids.push uuid
  end
  ActiveRecord::Base.connection.execute("INSERT INTO groups (uuid, owner_uuid, created_at, updated_at, name) VALUES #{values.join ','}")
  uuids
end

class PermissionPerfTest < ActionDispatch::IntegrationTest
  def test_groups_index
    n = 0
    act_as_system_user do
      puts("Time spent creating records:", Benchmark.measure do
             ActiveRecord::Base.transaction do
               root = Group.create!(owner_uuid: users(:permission_perftest).uuid)
               n += 1
               a = create_eight root.uuid
               n += 8
               a.each do |p1|
                 b = create_eight p1
                 n += 8
                 b.each do |p2|
                   c = create_eight p2
                   n += 8
                   c.each do |p3|
                     d = create_eight p3
                     n += 8
                   end
                 end
               end
             end
           end)
    end
    puts "created #{n}"
    puts "Time spent getting group index:"
    (0..4).each do
      puts(Benchmark.measure do
             get '/arvados/v1/groups', {format: :json, limit: 1000}, auth(:permission_perftest)
             assert json_response['items_available'] >= n
           end)
    end
  end
end
