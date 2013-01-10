class Node < ActiveRecord::Base
  include AssignUuid
  serialize :info, Hash
  before_validation :ensure_ping_secret

  MAX_SLOTS = 64

  @@confdir = begin
                Rails.configuration.dnsmasq_conf_dir or
                  ('/etc/dnsmasq.d' if File.exists? '/etc/dnsmasq.d/.')
              end

  def info
    @info ||= Hash.new
    super
  end

  def ping(o)
    raise "must have :ip and :ping_secret" unless o[:ip] and o[:ping_secret]

    if o[:ping_secret] != self.info[:ping_secret]
      logger.info "Ping: secret mismatch: received \"#{o[:ping_secret]}\" != \"#{self.info[:ping_secret]}\""
      return nil
    end
    self.last_ping_at = Time.now

    # Record IP address
    if self.ip_address.nil?
      logger.info "#{self.uuid} ip_address= #{o[:ip]}"
      self.ip_address = o[:ip]
      self.first_ping_at = Time.now
    end

    # Record instance ID if not already known
    self.info[:ec2_instance_id] ||= o[:ec2_instance_id]

    # Assign hostname
    if self.slot_number.nil?
      try_slot = 0
      begin
        self.slot_number = try_slot
        try_slot += 1
        break if self.save rescue nil
        raise "No available node slots" if try_slot == MAX_SLOTS
      end while true
      self.hostname = self.class.hostname_for_slot(self.slot_number)
      self.class.dnsmasq_update(self.hostname, self.ip_address)
    end

    save
  end

  def start!(ping_url_method)
    ping_url = ping_url_method.call({ uuid: self.uuid, ping_secret: self.info[:ping_secret] })
    cmd = ["ec2-run-instances",
           "--user-data '#{ping_url}'",
           "-t c1.xlarge -n 1 -g orvos-compute",
           "ami-68ca6901"
          ].join(' ')
    self.info[:ec2_start_command] = cmd
    logger.info "#{self.uuid} ec2_start_command= #{cmd.inspect}"
    result = `#{cmd} 2>&1`
    self.info[:ec2_start_result] = result
    logger.info "#{self.uuid} ec2_start_result= #{result.inspect}"
    result.match(/INSTANCE\s*(i-[0-9a-f]+)/) do |m|
      self.info[:ec2_instance_id] = m[1]
    end
    self.save!
  end

  protected

  def ensure_ping_secret
    self.info[:ping_secret] ||= rand(2**256).to_s(36)
  end

  def self.dnsmasq_update(hostname, ip_address)
    return unless @@confdir
    ptr_domain = ip_address.
      split('.').reverse.join('.').concat('.in-addr.arpa')
    hostfile = File.join @@confdir, hostname
    File.open hostfile, 'w' do |f|
      f.puts "address=/#{hostname}/#{ip_address}"
      f.puts "ptr-record=#{ptr_domain},#{hostname}"
    end
    File.open(File.join(@@confdir, 'restart.txt'), 'w') do |f|
      # this should trigger a dnsmasq restart
    end
  end

  def self.hostname_for_slot(slot_number)
    "compute#{slot_number}"
  end

  # At startup, make sure all DNS entries exist.  Otherwise, slurmctld
  # will refuse to start.
  if @@confdir and
      !File.exists? (File.join(@@confdir, hostname_for_slot(MAX_SLOTS-1)))
    (0..MAX_SLOTS-1).each do |slot_number|
      hostname = hostname_for_slot(slot_number)
      hostfile = File.join @@confdir, hostname
      if !File.exists? hostfile
        dnsmasq_update(hostname, '127.40.4.0')
      end
    end
  end
end
