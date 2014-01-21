#!/usr/bin/env ruby

# usage: list-inactive-users.rb [n-days-old-to-ignore]
#
# (default = 7)

abort 'Error: Ruby >= 1.9.3 required.' if RUBY_VERSION < '1.9.3'

threshold = ARGV.shift.to_i rescue 7

require 'arvados'
arv = Arvados.new(api_version: 'v1')

saidheader = false
arv.user.list(where: {is_active: false})[:items].each do |user|
  if Time.now - Time.parse(user[:created_at]) < threshold*86400
    if !saidheader
      saidheader = true
      puts "Inactive users who first logged in <#{threshold} days ago:"
      puts ""
    end
    puts "#{user[:modified_at]} #{user[:uuid]} #{user[:full_name]} <#{user[:email]}>"
  end
end
