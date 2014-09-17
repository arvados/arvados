class Node < ArvadosModel
  include HasUuid
  include KindAndEtag
  include CommonApiTemplate
  serialize :info, Hash
  before_validation :ensure_ping_secret
  after_update :dnsmasq_update

  MAX_SLOTS = 64

  @@confdir = Rails.configuration.dnsmasq_conf_dir
  @@domain = Rails.configuration.compute_node_domain rescue `hostname --domain`.strip
  @@nameservers = Rails.configuration.compute_node_nameservers

  api_accessible :user, :extend => :common do |t|
    t.add :hostname
    t.add :domain
    t.add :ip_address
    t.add :last_ping_at
    t.add :slot_number
    t.add :status
    t.add :crunch_worker_state
    t.add :info
  end
  api_accessible :superuser, :extend => :user do |t|
    t.add :first_ping_at
    t.add lambda { |x| @@nameservers }, :as => :nameservers
  end

  def info
    if current_user.andand.current_user.is_admin
      super
    else
      super.select { |k| not k.to_s.include? "secret" }
    end
  end

  def domain
    super || @@domain
  end

  def crunch_worker_state
    case self.info.andand['slurm_state']
    when 'alloc', 'comp'
      'busy'
    when 'idle'
      'idle'
    else
      'down'
    end
  end

  def status
    if !self.last_ping_at
      if Time.now - self.created_at > 5.minutes
        'startup-fail'
      else
        'pending'
      end
    elsif Time.now - self.last_ping_at > 1.hours
      'missing'
    else
      'running'
    end
  end

  def ping(o)
    raise "must have :ip and :ping_secret" unless o[:ip] and o[:ping_secret]

    if o[:ping_secret] != self.info['ping_secret']
      logger.info "Ping: secret mismatch: received \"#{o[:ping_secret]}\" != \"#{self.info['ping_secret']}\""
      raise ArvadosModel::UnauthorizedError.new("Incorrect ping_secret")
    end
    self.last_ping_at = Time.now

    @bypass_arvados_authorization = true

    # Record IP address
    if self.ip_address.nil?
      logger.info "#{self.uuid} ip_address= #{o[:ip]}"
      self.ip_address = o[:ip]
      self.first_ping_at = Time.now
    end

    # Record instance ID if not already known
    if o[:ec2_instance_id]
      if !self.info['ec2_instance_id']
        self.info['ec2_instance_id'] = o[:ec2_instance_id]
        if (Rails.configuration.compute_node_ec2_tag_enable rescue true)
          tag_cmd = ("ec2-create-tags #{o[:ec2_instance_id]} " +
                     "--tag 'Name=#{self.uuid}'")
          `#{tag_cmd}`
        end
      elsif self.info['ec2_instance_id'] != o[:ec2_instance_id]
        logger.debug "Multiple nodes have credentials for #{self.uuid}"
        raise "#{self.uuid} is already running at #{self.info['ec2_instance_id']} so rejecting ping from #{o[:ec2_instance_id]}"
      end
    end

    # Assign hostname
    if self.slot_number.nil?
      try_slot = 0
      begin
        self.slot_number = try_slot
        begin
          self.save!
          break
        rescue ActiveRecord::RecordNotUnique
          try_slot += 1
        end
        raise "No available node slots" if try_slot == MAX_SLOTS
      end while true
      self.hostname = self.class.hostname_for_slot(self.slot_number)
      if info['ec2_instance_id']
        if (Rails.configuration.compute_node_ec2_tag_enable rescue true)
          `ec2-create-tags #{self.info['ec2_instance_id']} --tag 'hostname=#{self.hostname}'`
        end
      end
    end

    # Record other basic stats
    ['total_cpu_cores', 'total_ram_mb', 'total_scratch_mb'].each do |key|
      if value = (o[key] or o[key.to_sym])
        self.info[key] = value
      else
        self.info.delete(key)
      end
    end

    save!
  end

  def start!(ping_url_method)
    ensure_permission_to_save
    ping_url = ping_url_method.call({ id: self.uuid, ping_secret: self.info['ping_secret'] })
    if (Rails.configuration.compute_node_ec2run_args and
        Rails.configuration.compute_node_ami)
      ec2_args = ["--user-data '#{ping_url}'",
                  "-t c1.xlarge -n 1",
                  Rails.configuration.compute_node_ec2run_args,
                  Rails.configuration.compute_node_ami
                 ]
      ec2run_cmd = ["ec2-run-instances",
                    "--client-token", self.uuid,
                    ec2_args].flatten.join(' ')
      ec2spot_cmd = ["ec2-request-spot-instances",
                     "-p #{Rails.configuration.compute_node_spot_bid} --type one-time",
                     ec2_args].flatten.join(' ')
    else
      ec2run_cmd = ''
      ec2spot_cmd = ''
    end
    self.info['ec2_run_command'] = ec2run_cmd
    self.info['ec2_spot_command'] = ec2spot_cmd
    self.info['ec2_start_command'] = ec2spot_cmd
    logger.info "#{self.uuid} ec2_start_command= #{ec2spot_cmd.inspect}"
    result = `#{ec2spot_cmd} 2>&1`
    self.info['ec2_start_result'] = result
    logger.info "#{self.uuid} ec2_start_result= #{result.inspect}"
    result.match(/INSTANCE\s*(i-[0-9a-f]+)/) do |m|
      instance_id = m[1]
      self.info['ec2_instance_id'] = instance_id
      if (Rails.configuration.compute_node_ec2_tag_enable rescue true)
        `ec2-create-tags #{instance_id} --tag 'Name=#{self.uuid}'`
      end
    end
    result.match(/SPOTINSTANCEREQUEST\s*(sir-[0-9a-f]+)/) do |m|
      sir_id = m[1]
      self.info['ec2_sir_id'] = sir_id
      if (Rails.configuration.compute_node_ec2_tag_enable rescue true)
        `ec2-create-tags #{sir_id} --tag 'Name=#{self.uuid}'`
      end
    end
    self.save!
  end

  protected

  def ensure_ping_secret
    self.info['ping_secret'] ||= rand(2**256).to_s(36)
  end

  def dnsmasq_update
    if self.hostname_changed? or self.ip_address_changed?
      if self.hostname and self.ip_address
        self.class.dnsmasq_update(self.hostname, self.ip_address)
      end
    end
  end

  def self.dnsmasq_update(hostname, ip_address)
    return unless @@confdir
    ptr_domain = ip_address.
      split('.').reverse.join('.').concat('.in-addr.arpa')
    hostfile = File.join @@confdir, hostname
    File.open hostfile, 'w' do |f|
      f.puts "address=/#{hostname}/#{ip_address}"
      f.puts "address=/#{hostname}.#{@@domain}/#{ip_address}" if @@domain
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

  def permission_to_update
    @bypass_arvados_authorization or super
  end

  def permission_to_create
    current_user and current_user.is_admin
  end
end
