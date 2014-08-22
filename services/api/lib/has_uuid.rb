module HasUuid

  UUID_REGEX = /^[0-9a-z]{5}-([0-9a-z]{5})-[0-9a-z]{15}$/

  def self.included(base)
    base.extend(ClassMethods)
    base.validate :validate_uuid
    base.before_create :assign_uuid
    base.before_destroy :destroy_permission_links
    base.has_many :links_via_head, class_name: 'Link', foreign_key: :head_uuid, primary_key: :uuid, conditions: "not (link_class = 'permission')", dependent: :restrict
    base.has_many :links_via_tail, class_name: 'Link', foreign_key: :tail_uuid, primary_key: :uuid, conditions: "not (link_class = 'permission')", dependent: :restrict
  end

  module ClassMethods
    def uuid_prefix
      Digest::MD5.hexdigest(self.to_s).to_i(16).to_s(36)[-5..-1]
    end
    def generate_uuid
      [Server::Application.config.uuid_prefix,
       self.uuid_prefix,
       rand(2**256).to_s(36)[-15..-1]].
        join '-'
    end
  end

  protected

  def respond_to_uuid?
    self.respond_to? :uuid
  end

  def validate_uuid
    if self.respond_to_uuid? and self.uuid_changed?
      if current_user.andand.is_admin and self.uuid.is_a?(String)
        if (re = self.uuid.match HasUuid::UUID_REGEX)
          if re[1] == self.class.uuid_prefix
            return true
          else
            self.errors.add(:uuid, "type field is '#{re[1]}', expected '#{self.class.uuid_prefix}'")
            return false
          end
        else
          self.errors.add(:uuid, "not a valid Arvados uuid '#{self.uuid}'")
          return false
        end
      else
        if self.new_record?
          self.errors.add(:uuid, "assignment not permittid")
        else
          self.errors.add(:uuid, "change not permitted")
        end
        return false
      end
    else
      return true
    end
  end

  def assign_uuid
    if self.respond_to_uuid? and self.uuid.nil? or self.uuid.empty?
      self.uuid = self.class.generate_uuid
    end
    true
  end

  def destroy_permission_links
    if uuid
      Link.destroy_all(['link_class=? and (head_uuid=? or tail_uuid=?)',
                        'permission', uuid, uuid])
    end
  end
end
