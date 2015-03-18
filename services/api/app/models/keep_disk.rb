class KeepDisk < ArvadosModel
  include HasUuid
  include KindAndEtag
  include CommonApiTemplate
  before_validation :ensure_ping_secret

  api_accessible :user, extend: :common do |t|
    t.add :node_uuid
    t.add :filesystem_uuid
    t.add :bytes_total
    t.add :bytes_free
    t.add :is_readable
    t.add :is_writable
    t.add :last_read_at
    t.add :last_write_at
    t.add :last_ping_at
    t.add :service_host
    t.add :service_port
    t.add :service_ssl_flag
    t.add :keep_service_uuid
  end
  api_accessible :superuser, :extend => :user do |t|
    t.add :ping_secret
  end

  def foreign_key_attributes
    super.reject { |a| a == "filesystem_uuid" }
  end

  def ping(o)
    raise "must have :service_host and :ping_secret" unless o[:service_host] and o[:ping_secret]

    if o[:ping_secret] != self.ping_secret
      logger.info "Ping: secret mismatch: received \"#{o[:ping_secret]}\" != \"#{self.ping_secret}\""
      return nil
    end

    @bypass_arvados_authorization = true
    self.update_attributes!(o.select { |k,v|
                             [:bytes_total,
                              :bytes_free,
                              :is_readable,
                              :is_writable,
                              :last_read_at,
                              :last_write_at
                             ].collect(&:to_s).index k
                           }.merge(last_ping_at: db_current_time))
  end

  def service_host
    KeepService.find_by_uuid(self.keep_service_uuid).andand.service_host
  end

  def service_port
    KeepService.find_by_uuid(self.keep_service_uuid).andand.service_port
  end

  def service_ssl_flag
    KeepService.find_by_uuid(self.keep_service_uuid).andand.service_ssl_flag
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
