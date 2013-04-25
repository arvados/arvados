class KeepDisk < ArvadosModel
  include AssignUuid
  include KindAndEtag
  include CommonApiTemplate
  before_validation :ensure_ping_secret

  api_accessible :superuser, :extend => :common do |t|
    t.add :node_uuid
    t.add :filesystem_uuid
    t.add :ping_secret
    t.add :bytes_total
    t.add :bytes_free
    t.add :is_readable
    t.add :is_writable
    t.add :last_read_at
    t.add :last_write_at
    t.add :last_ping_at
  end

  def ping(o)
    raise "must have :service_host and :ping_secret" unless o[:service_host] and o[:ping_secret]

    if o[:ping_secret] != self.ping_secret
      logger.info "Ping: secret mismatch: received \"#{o[:ping_secret]}\" != \"#{self.ping_secret}\""
      return nil
    end

    @bypass_arvados_authorization = true
    self.update_attributes(o.select { |k,v|
                             [:service_host,
                              :service_port,
                              :service_ssl_flag,
                              :bytes_total,
                              :bytes_free,
                              :is_readable,
                              :is_writable,
                              :last_read_at,
                              :last_write_at
                             ].collect(&:to_s).index k
                           }.merge(last_ping_at: Time.now))
  end

  protected

  def ensure_ping_secret
    self.ping_secret ||= rand(2**256).to_s(36)
  end

  def permission_to_update
    @bypass_arvados_authorization or super
  end

  def permission_to_create
    current_user and current_user.is_admin
  end
end
