class OrvosModel < ActiveRecord::Base
  self.abstract_class = true

  include CurrentApiClient      # current_user, current_api_client, etc.

  attr_protected :created_at
  attr_protected :modified_by_user
  attr_protected :modified_by_client
  attr_protected :modified_at
  before_update :permission_to_update
  before_create :update_modified_by_fields
  before_update :update_modified_by_fields

  def self.kind_class(kind)
    kind.match(/^orvos\#(.+?)(_list|List)?$/)[1].pluralize.classify.constantize rescue nil
  end

  def eager_load_associations
    self.class.columns.each do |col|
      re = col.name.match /^(.*)_kind$/
      if (re and
          self.respond_to? re[1].to_sym and
          (auuid = self.send((re[1] + '_uuid').to_sym)) and
          (aclass = self.class.kind_class(self.send(col.name.to_sym))) and
          (aobject = aclass.where('uuid=?', auuid).first))
        self.instance_variable_set('@'+re[1], aobject)
      end
    end
  end

  protected

  def permission_to_update
    return false unless current_user
    return true if current_user.is_admin
    if self.owner_changed? and
        self.owner_was != current_user.uuid and
        0 == Link.where(link_class: 'permission',
                        name: 'can_pillage',
                        tail_uuid: self.owner,
                        head_uuid: current_user.uuid).count
      logger.warn "User #{current_user.uuid} tried to change owner of #{self.class.to_s} #{self.uuid} to #{self.owner}"
      return false
    end
    self.owner == current_user.uuid or
      current_user.is_admin or
      current_user.uuid == self.uuid or
      Link.where(link_class: 'permission',
                 name: 'can_write',
                 tail_uuid: self.owner,
                 head_uuid: current_user.uuid).count > 0
  end

  def update_modified_by_fields
    if self.changed?
      self.created_at ||= Time.now
      self.owner ||= current_user.uuid
      self.modified_at = Time.now
      self.modified_by_user = current_user.uuid
      self.modified_by_client = current_api_client ? current_api_client.uuid : nil
    end
  end
end
