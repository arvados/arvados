#!/usr/bin/env ruby

require 'sinatra/base'
require 'digest/md5'
require 'digest/sha1'
require 'arvados'

class Keep < Sinatra::Base
  configure do
    mime_type :binary, 'application/octet-stream'
    enable :logging
    set :port, (ENV['PORT'] || '25107').to_i
    set :bind, (ENV['IP'] || '0.0.0.0')
  end

  def verify_hash(data, hash)
    if hash.length == 32
      Digest::MD5.hexdigest(data) == hash && hash
    elsif hash.length == 40
      Digest::SHA1.hexdigest(data) == hash && hash
    else
      false
    end
  end

  def self.debuglevel
    if ENV['DEBUG'] and ENV['DEBUG'].match /^-?\d+/
      ENV['DEBUG'].to_i
    else
      0
    end
  end

  def self.debuglog(loglevel, msg)
    if debuglevel >= loglevel
      $stderr.puts "[keepd/#{$$} #{Time.now}] #{msg}"
    end
  end
  def debuglog(*args)
    self.class.debuglog *args
  end

  def self.keepdirs
    return @@keepdirs if defined? @@keepdirs
    # Configure backing store directories
    keepdirs = []
    rootdir = (ENV['KEEP_ROOT'] || '/').sub /\/$/, ''
    `mount`.split("\n").each do |mountline|
      dev, on_txt, mountpoint, type_txt, fstype, opts = mountline.split
      if on_txt == 'on' and type_txt == 'type'
        debuglog 2, "dir #{mountpoint} is mounted"
        if mountpoint[0..(rootdir.length)] == rootdir + '/'
          debuglog 2, "dir #{mountpoint} is in #{rootdir}/"
          keepdir = "#{mountpoint.sub /\/$/, ''}/keep"
          if File.exists? "#{keepdir}/."
            keepdirs << {
              :root => "#{keepdir}",
              :readonly => false,
              :device => dev,
              :device_inode => File.stat(dev).ino
            }
            if opts.gsub(/[\(\)]/, '').split(',').index('ro')
              keepdirs[-1][:readonly] = true
            end
            debuglog 2, "keepdir #{keepdirs[-1].inspect}"
          end
        end
      end
    end
    Dir.open('/dev/disk/by-uuid/').each do |fs_uuid|
      next if fs_uuid.match /^\./
      fs_root_inode = File.stat("/dev/disk/by-uuid/#{fs_uuid}").ino
      keepdirs.each do |kd|
        if kd[:device_inode] == fs_root_inode
          kd[:filesystem_uuid] = fs_uuid
          debuglog 0, "keepdir #{kd.inspect}"
        end
      end
    end
    @@keepdirs = keepdirs
  end
  self.keepdirs

  def find_backfile(hash, opts)
    subdir = hash[0..2]
    @@keepdirs.each do |keepdir|
      backfile = "#{keepdir[:root]}/#{subdir}/#{hash}"
      if File.exists? backfile
        data = nil
        File.open("#{keepdir[:root]}/lock", "a+") do |f|
          if f.flock File::LOCK_EX
            data = File.read backfile
          end
        end
        if data and (!opts[:verify_hash] or verify_hash data, hash)
          return [backfile, data]
        end
      end
    end
    nil
  end

  get '/:locator' do |locator|
    regs = locator.match /^([0-9a-f]{32,})/
    if regs
      hash = regs[1]
      backfile, data = find_backfile hash, :verify_hash => false
      if data
        content_type :binary
        body data
      else
        status 404
        body 'not found'
      end
    else
      pass
    end
    self.class.ping_arvados
  end

  put '/:locator' do |locator|
    data = request.body.read
    hash = verify_hash(data, locator)
    if not hash
      status 422
      body "Checksum mismatch"
      return
    end
    backfile, havedata = find_backfile hash, :verify_hash => true
    if havedata
      status 200
      body 'OK'
    else
      wrote = nil
      subdir = hash[0..2]
      @@keepdirs.each do |keepdir|
        next if keepdir[:readonly]
        backdir = "#{keepdir[:root]}/#{subdir}"
        if !File.exists? backdir
          begin
            Dir.mkdir backdir
          rescue
          end
        end
        backfile = "#{keepdir[:root]}/#{subdir}/#{hash}"
        File.open("#{keepdir[:root]}/lock", "a+") do |lf|
          if lf.flock File::LOCK_EX
            File.open(backfile + ".tmp", "a+") do |wf|
              if wf.flock File::LOCK_EX
                wf.seek 0, File::SEEK_SET
                wf.truncate 0
                wrote = wf.write data
              end
              if wrote == data.length
                File.rename backfile+".tmp", backfile
                break
              else
                File.unlink backfile+".tmp"
              end
            end
          end
        end
      end
      if wrote == data.length
        status 200
        body 'OK'
      else
        status 500
        body 'Fail'
      end
    end
    self.class.ping_arvados
  end

  protected

  def self.ping_arvados
    return if defined? @@last_ping_at and @@last_ping_at > Time.now - 300
    @@last_ping_at = Time.now
    begin
      @@arvados ||= Arvados.new
      @@keepdirs.each do |kd|
        found = false
        @@arvados.keep_disk.list(where:{filesystem_uuid: kd[:filesystem_uuid]})[:items].each do |arv_disk|
          ack = @@arvados.keep_disk.
            update(uuid: arv_disk[:uuid],
                   keep_disk: { last_ping_at: Time.now })
          debuglog 0, "device #{kd[:device]} uuid #{ack[:uuid]} last_ping_at #{ack[:last_ping_at]}"
          found = true
          break
        end
        if not found
          @@arvados.keep_disk.create keep_disk: {
            filesystem_uuid: kd[:filesystem_uuid],
            last_ping_at: Time.now
          }
        end
      end
    rescue Exception => e
      debuglog 0, "ping_arvados: #{e.inspect}"
    end
  end
  self.ping_arvados

  if app_file == $0
    run! do |server|
      if ENV['SSL_CERT'] and ENV['SSL_KEY']
        ssl_options = {
          :cert_chain_file => ENV['SSL_CERT'],
          :private_key_file => ENV['SSL_KEY'],
          :verify_peer => false
        }
        server.ssl = true
        server.ssl_options = ssl_options
      end
    end
  end
end
