# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

require 'tempfile'

class Node < ArvadosModel
  include HasUuid
  include KindAndEtag
  include CommonApiTemplate

  # Posgresql JSONB columns should NOT be declared as serialized, Rails 5
  # already know how to properly treat them.
  attribute :properties, :jsonbHash, default: {}
  attribute :info, :jsonbHash, default: {}

  before_validation :ensure_ping_secret
  after_update :dns_server_update

  # Only a controller can figure out whether or not the current API tokens
  # have access to the associated Job.  They're expected to set
  # job_readable=true if the Job UUID can be included in the API response.
  belongs_to(:job, foreign_key: :job_uuid, primary_key: :uuid)
  attr_accessor :job_readable

  UNUSED_NODE_IP = '127.40.4.0'
  MAX_VMS = 3

  api_accessible :user, :extend => :common do |t|
    t.add :hostname
    t.add :domain
    t.add :ip_address
    t.add :last_ping_at
    t.add :slot_number
    t.add :status
    t.add :api_job_uuid, as: :job_uuid
    t.add :crunch_worker_state
    t.add :properties
  end
  api_accessible :superuser, :extend => :user do |t|
    t.add :first_ping_at
    t.add :info
    t.add lambda { |x| Rails.configuration.Containers.SLURM.Managed.ComputeNodeNameservers.keys }, :as => :nameservers
  end

  after_initialize do
    @bypass_arvados_authorization = false
  end

  def domain
    super || Rails.configuration.Containers.SLURM.Managed.ComputeNodeDomain
  end

  def api_job_uuid
    job_readable ? job_uuid : nil
  end

  def crunch_worker_state
    return 'down' if slot_number.nil?
    case self.info.andand['slurm_state']
    when 'alloc', 'comp', 'mix', 'drng'
      'busy'
    when 'idle'
      'idle'
    else
      'down'
    end
  end

  def status
    if !self.last_ping_at
      if db_current_time - self.created_at > 5.minutes
        'startup-fail'
      else
        'pending'
      end
    elsif db_current_time - self.last_ping_at > 1.hours
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

    current_time = db_current_time
    self.last_ping_at = current_time

    @bypass_arvados_authorization = true

    # Record IP address
    if self.ip_address.nil?
      logger.info "#{self.uuid} ip_address= #{o[:ip]}"
      self.ip_address = o[:ip]
      self.first_ping_at = current_time
    end

    # Record instance ID if not already known
    if o[:ec2_instance_id]
      if !self.info['ec2_instance_id']
        self.info['ec2_instance_id'] = o[:ec2_instance_id]
      elsif self.info['ec2_instance_id'] != o[:ec2_instance_id]
        logger.debug "Multiple nodes have credentials for #{self.uuid}"
        raise "#{self.uuid} is already running at #{self.info['ec2_instance_id']} so rejecting ping from #{o[:ec2_instance_id]}"
      end
    end

    assign_slot

    # Record other basic stats
    ['total_cpu_cores', 'total_ram_mb', 'total_scratch_mb'].each do |key|
      if value = (o[key] or o[key.to_sym])
        self.properties[key] = value.to_i
      else
        self.properties.delete(key)
      end
    end

    save!
  end

  def assign_slot
    return if self.slot_number.andand > 0
    while true
      self.slot_number = self.class.available_slot_number
      if self.slot_number.nil?
        raise "No available node slots"
      end
      begin
        save!
        return assign_hostname
      rescue ActiveRecord::RecordNotUnique
        # try again
      end
    end
  end

  protected

  def assign_hostname
    if self.hostname.nil? and Rails.configuration.Containers.SLURM.Managed.AssignNodeHostname
      self.hostname = self.class.hostname_for_slot(self.slot_number)
    end
  end

  def self.available_slot_number
    # Join the sequence 1..max with the nodes table. Return the first
    # (i.e., smallest) value that doesn't match the slot_number of any
    # existing node.
    connection.exec_query('SELECT n FROM generate_series(1, $1) AS slot(n)
                          LEFT JOIN nodes ON n=slot_number
                          WHERE slot_number IS NULL
                          LIMIT 1',
                          # query label:
                          'Node.available_slot_number',
                          # [col_id, val] for $1 vars:
                          [[nil, MAX_VMS]],
                         ).rows.first.andand.first
  end

  def ensure_ping_secret
    self.info['ping_secret'] ||= rand(2**256).to_s(36)
  end

  def dns_server_update
    if saved_change_to_ip_address? && ip_address
      Node.where('id != ? and ip_address = ?',
                 id, ip_address).each do |stale_node|
        # One or more(!) stale node records have the same IP address
        # as the new node. Clear the ip_address field on the stale
        # nodes. Otherwise, we (via SLURM) might inadvertently connect
        # to the new node using the old node's hostname.
        stale_node.update_attributes!(ip_address: nil)
      end
    end
    if hostname_before_last_save && saved_change_to_hostname?
      self.class.dns_server_update(hostname_before_last_save, UNUSED_NODE_IP)
    end
    if hostname && (saved_change_to_hostname? || saved_change_to_ip_address?)
      self.class.dns_server_update(hostname, ip_address || UNUSED_NODE_IP)
    end
  end

  def self.dns_server_update hostname, ip_address
    ok = true

    ptr_domain = ip_address.
      split('.').reverse.join('.').concat('.in-addr.arpa')

    template_vars = {
      hostname: hostname,
      uuid_prefix: Rails.configuration.ClusterID,
      ip_address: ip_address,
      ptr_domain: ptr_domain,
    }

    if (!Rails.configuration.Containers.SLURM.Managed.DNSServerConfDir.to_s.empty? and
        !Rails.configuration.Containers.SLURM.Managed.DNSServerConfTemplate.to_s.empty?)
      tmpfile = nil
      begin
        begin
          template = IO.read(Rails.configuration.Containers.SLURM.Managed.DNSServerConfTemplate)
        rescue IOError, SystemCallError => e
          logger.error "Reading #{Rails.configuration.Containers.SLURM.Managed.DNSServerConfTemplate}: #{e.message}"
          raise
        end

        hostfile = File.join Rails.configuration.Containers.SLURM.Managed.DNSServerConfDir, "#{hostname}.conf"
        Tempfile.open(["#{hostname}-", ".conf.tmp"],
                                 Rails.configuration.Containers.SLURM.Managed.DNSServerConfDir) do |f|
          tmpfile = f.path
          f.puts template % template_vars
        end
        File.rename tmpfile, hostfile
      rescue IOError, SystemCallError => e
        logger.error "Writing #{hostfile}: #{e.message}"
        ok = false
      ensure
        if tmpfile and File.file? tmpfile
          # Cleanup remaining temporary file.
          File.unlink tmpfile
        end
      end
    end

    if !Rails.configuration.Containers.SLURM.Managed.DNSServerUpdateCommand.empty?
      cmd = Rails.configuration.Containers.SLURM.Managed.DNSServerUpdateCommand % template_vars
      if not system cmd
        logger.error "dns_server_update_command #{cmd.inspect} failed: #{$?}"
        ok = false
      end
    end

    if (!Rails.configuration.Containers.SLURM.Managed.DNSServerConfDir.to_s.empty? and
        !Rails.configuration.Containers.SLURM.Managed.DNSServerReloadCommand.to_s.empty?)
      restartfile = File.join(Rails.configuration.Containers.SLURM.Managed.DNSServerConfDir, 'restart.txt')
      begin
        File.open(restartfile, 'w') do |f|
          # Typically, this is used to trigger a dns server restart
          f.puts Rails.configuration.Containers.SLURM.Managed.DNSServerReloadCommand
        end
      rescue IOError, SystemCallError => e
        logger.error "Unable to write #{restartfile}: #{e.message}"
        ok = false
      end
    end

    ok
  end

  def self.hostname_for_slot(slot_number)
    config = Rails.configuration.Containers.SLURM.Managed.AssignNodeHostname

    return nil if !config

    sprintf(config, {:slot_number => slot_number})
  end

  # At startup, make sure all DNS entries exist.  Otherwise, slurmctld
  # will refuse to start.
  if (!Rails.configuration.Containers.SLURM.Managed.DNSServerConfDir.to_s.empty? and
      !Rails.configuration.Containers.SLURM.Managed.DNSServerConfTemplate.to_s.empty? and
      !Rails.configuration.Containers.SLURM.Managed.AssignNodeHostname.empty?)

    (0..MAX_VMS-1).each do |slot_number|
      hostname = hostname_for_slot(slot_number)
      hostfile = File.join Rails.configuration.Containers.SLURM.Managed.DNSServerConfDir, "#{hostname}.conf"
      if !File.exist? hostfile
        n = Node.where(:slot_number => slot_number).first
        if n.nil? or n.ip_address.nil?
          dns_server_update(hostname, UNUSED_NODE_IP)
        else
          dns_server_update(hostname, n.ip_address)
        end
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
