#!/usr/bin/env ruby

# Keep node.info[:running_job_uuid] and node.info[:slurm_state] up to date.
#
# use:     script/update_node_attributes.rb [rails_env] [update_interval]
# example: script/update_node_attributes.rb production 10

ENV["RAILS_ENV"] = ARGV[0] || "development"
@update_interval = ARGV[1] ? ARGV[1].to_i : 5

require File.dirname(__FILE__) + '/../config/boot'
require File.dirname(__FILE__) + '/../config/environment'

include ApplicationHelper
act_as_system_user

@slurm_state = {}
@running_job_uuid = {}

while true
  IO.popen('sinfo --noheader --Node || true').readlines.each do |line|
    tokens = line.strip.split
    nodestate = tokens.last.downcase

    nodenames = []
    if (re = tokens.first.match /^([^\[]*)\[([-\d,]+)\]$/)
      nodeprefix = re[1]
      re[2].split(',').each do |number_range|
        if number_range.index('-')
          range = number_range.split('-').collect(&:to_i)
          (range[0]..range[1]).each do |n|
            nodenames << "#{nodeprefix}#{n}"
          end
        else
          nodenames << "#{nodeprefix}#{number_range}"
        end
      end
    else
      nodenames << tokens.first
    end

    nodenames.each do |nodename|
      if @slurm_state[nodename] != nodestate
        has_no_job = ! ['alloc','comp'].index(nodestate)
        node = Node.
          where('slot_number=? and hostname=?',
                nodename.match(/(\d+)$/)[1].to_i,
                nodename).
          first
        raise "Fatal: Node does not exist: #{nodename}" if !node

        puts "Node #{node.uuid} slot #{node.slot_number} name #{node.hostname} state #{nodestate}#{' (has_no_job)' if has_no_job}"
        node_info_was = node.info.dup
        node.info[:slurm_state] = nodestate
        node.info[:running_job_uuid] = nil if has_no_job
        if node_info_was != node.info and not node.save
          raise "Fail: update node #{node.uuid} state #{nodestate}"
        end
        @slurm_state[nodename] = nodestate
      end
    end
  end

  IO.popen('squeue --noheader --format="%j %t %N" || true').readlines.each do |line|
    tokens = line.strip.split
    running_job_uuid = tokens.first

    nodenames = []
    if (re = tokens.last.match /^([^\[]*)\[([-\d,]+)\]$/)
      nodeprefix = re[1]
      re[2].split(',').each do |number_range|
        if number_range.index('-')
          range = number_range.split('-').collect(&:to_i)
          (range[0]..range[1]).each do |n|
            nodenames << "#{nodeprefix}#{n}"
          end
        else
          nodenames << "#{nodeprefix}#{number_range}"
        end
      end
    else
      nodenames << tokens.first
    end

    nodenames.each do |nodename|
      if @running_job_uuid[nodename] != running_job_uuid
        node = Node.
          where('slot_number=? and hostname=?',
                nodename.match(/(\d+)$/)[1].to_i,
                nodename).
          first
        raise "Fatal: Node does not exist: #{nodename}" if !node
        puts "Node #{node.uuid} slot #{node.slot_number} name #{node.hostname} running_job_uuid #{running_job_uuid}"
        if node.info[:running_job_uuid] != running_job_uuid
          node.info[:running_job_uuid] = running_job_uuid
          if not node.save
            raise "Fail: update node #{node.uuid} running_job_uuid #{running_job_uuid}"
          end
        end
        @running_job_uuid[nodename] = running_job_uuid
      end
    end
  end

  sleep @update_interval
end
