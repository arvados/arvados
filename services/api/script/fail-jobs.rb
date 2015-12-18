#!/usr/bin/env ruby

require 'trollop'

opts = Trollop::options do
  banner 'Fail jobs that have state=="Running".'
  banner 'Options:'
  opt(:before,
      'fail only jobs that started before the given time (or "reboot")',
      type: :string)
end

ENV["RAILS_ENV"] = ARGV[0] || ENV["RAILS_ENV"] || "development"
require File.dirname(__FILE__) + '/../config/boot'
require File.dirname(__FILE__) + '/../config/environment'
require Rails.root.join('lib/crunch_dispatch.rb')

CrunchDispatch.new.fail_jobs before: opts[:before]
