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
    raise "must have :ip and :ping_secret" unless o[:ip] and o[:ping_secret]

    if o[:ping_secret] != self.ping_secret
      logger.info "Ping: secret mismatch: received \"#{o[:ping_secret]}\" != \"#{self.info[:ping_secret]}\""
      return nil
    end
    self.last_ping_at = Time.now

    @bypass_arvados_authorization = true

    save!
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
