# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

module Psych
  module Visitors
    class YAMLTree < Psych::Visitors::Visitor
      def visit_ActiveSupport_Duration o
        seconds = o.to_i
        outstr = ""
        if seconds / 3600 > 0
          outstr += "#{seconds / 3600}h"
          seconds = seconds % 3600
        end
        if seconds / 60 > 0
          outstr += "#{seconds / 60}m"
          seconds = seconds % 60
        end
        if seconds > 0
          outstr += "#{seconds}s"
        end
        if outstr == ""
          outstr = "0s"
        end
        @emitter.scalar outstr, nil, nil, true, false, Nodes::Scalar::ANY
      end
    end
  end
end

def set_cfg cfg, k, v
  # "foo.bar = baz" --> { cfg["foo"]["bar"] = baz }
  ks = k.split '.'
  k = ks.pop
  ks.each do |kk|
    cfg = cfg[kk]
    if cfg.nil?
      break
    end
  end
  if !cfg.nil?
    cfg[k] = v
  end
end

$config_migrate_map = {}
$config_types = {}
def declare_config(assign_to, configtype, migrate_from=nil, migrate_fn=nil)
  if migrate_from
    $config_migrate_map[migrate_from] = migrate_fn || ->(cfg, k, v) {
      set_cfg cfg, assign_to, v
    }
  end
  $config_types[assign_to] = configtype
end

module Boolean; end
class TrueClass; include Boolean; end
class FalseClass; include Boolean; end

class NonemptyString < String
end

def parse_duration durstr
  duration_re = /(\d+(\.\d+)?)(s|m|h)/
  dursec = 0
  while durstr != ""
    mt = duration_re.match durstr
    if !mt
      raise "#{cfgkey} not a valid duration: '#{cfg[k]}', accepted suffixes are s, m, h"
    end
    multiplier = {s: 1, m: 60, h: 3600}
    dursec += (Float(mt[1]) * multiplier[mt[3].to_sym])
    durstr = durstr[mt[0].length..-1]
  end
  return dursec.seconds
end

def migrate_config from_config, to_config
  remainders = {}
  from_config.each do |k, v|
    if $config_migrate_map[k.to_sym]
      $config_migrate_map[k.to_sym].call to_config, k, v
    else
      remainders[k] = v
    end
  end
  remainders
end

def coercion_and_check check_cfg
  $config_types.each do |cfgkey, cfgtype|
    cfg = check_cfg
    k = cfgkey
    ks = k.split '.'
    k = ks.pop
    ks.each do |kk|
      cfg = cfg[kk]
      if cfg.nil?
        break
      end
    end

    if cfg.nil?
      raise "missing #{cfgkey}"
    end

    if cfgtype == String and !cfg[k]
      cfg[k] = ""
    end

    if cfgtype == String and cfg[k].is_a? Symbol
      cfg[k] = cfg[k].to_s
    end

    if cfgtype == Pathname and cfg[k].is_a? String

      if cfg[k] == ""
        cfg[k] = Pathname.new("")
      else
        cfg[k] = Pathname.new(cfg[k])
        if !cfg[k].exist?
          raise "#{cfgkey} path #{cfg[k]} does not exist"
        end
      end
    end

    if cfgtype == NonemptyString
      if (!cfg[k] || cfg[k] == "")
        raise "#{cfgkey} cannot be empty"
      end
      if cfg[k].is_a? String
        next
      end
    end

    if cfgtype == ActiveSupport::Duration
      if cfg[k].is_a? Integer
        cfg[k] = cfg[k].seconds
      elsif cfg[k].is_a? String
        cfg[k] = parse_duration cfg[k]
      end
    end

    if cfgtype == URI
      cfg[k] = URI(cfg[k])
    end

    if !cfg[k].is_a? cfgtype
      raise "#{cfgkey} expected #{cfgtype} but was #{cfg[k].class}"
    end
  end

end

def copy_into_config src, dst
  src.each do |k, v|
    dst.send "#{k}=", Marshal.load(Marshal.dump v)
  end
end
