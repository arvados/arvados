#!/usr/bin/env ruby

dispatch_argv = []
ARGV.reject! do |arg|
  dispatch_argv.push(arg) if /^--/ =~ arg
end

ENV["RAILS_ENV"] = ARGV[0] || ENV["RAILS_ENV"] || "development"
require File.dirname(__FILE__) + '/../config/boot'
require File.dirname(__FILE__) + '/../config/environment'
require './lib/crunch_dispatch.rb'

CrunchDispatch.new.run dispatch_argv
