module CommonApiTemplate
  def self.included(base)
    base.extend(ClassMethods)
    base.acts_as_api
    base.api_accessible :common do |t|
      t.add :href
      t.add :kind
      t.add :etag
      t.add :uuid
      t.add :owner_uuid
      t.add :created_at
      t.add :modified_by_client_uuid
      t.add :modified_by_user_uuid
      t.add :modified_at
      t.add :updated_at
    end
  end

  module ClassMethods
  end
end
