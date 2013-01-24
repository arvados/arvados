class OrvosModel < ActiveRecord::Base
  self.abstract_class = true

  include CurrentApiClient      # current_user, current_api_client, etc.

  attr_protected :created_by_user
  attr_protected :created_by_client
  attr_protected :created_at
  attr_protected :modified_by_user
  attr_protected :modified_by_client
  attr_protected :modified_at
  before_create :initialize_created_by_fields
  before_update :update_modified_by_fields

  def self.kind_class(kind)
    kind.match(/^orvos\#(.+?)(_list|List)?$/)[1].pluralize.classify.constantize rescue nil
  end

  def eager_load_associations
    self.class.columns.each do |col|
      re = col.name.match /^(.*)_kind$/
      if (re and
          self.respond_to? re[1].to_sym and
          (auuid = self.send(re[1].to_sym)) and
          (aclass = self.class.kind_class(self.send(col.name.to_sym))) and
          (aobject = aclass.where('uuid=?', auuid).first))
        self.send((re[1]+'=').to_sym, aobject)
      end
    end
  end

  protected

  def update_modified_by_fields
    if self.changed?
      self.modified_at = Time.now
      self.modified_by_user = current_user.uuid
      self.modified_by_client = current_api_client.uuid
    end
  end

  def initialize_created_by_fields
    self.created_at = Time.now
    self.created_by_user = current_user.uuid
    self.created_by_client = current_api_client.uuid
    self.modified_at = Time.now
    self.modified_by_user = current_user.uuid
    self.modified_by_client = current_api_client.uuid
  end
end
