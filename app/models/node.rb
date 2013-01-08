class Node < ActiveRecord::Base
  include AssignUuid
  serialize :info, Hash
  before_validation :ensure_ping_secret

  MAX_SLOTS = 64

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
      self.hostname = "compute#{self.slot_number}"
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
end
