module CommonApiTemplate
  def self.included(base)
    base.extend(ClassMethods)
    base.acts_as_api
    base.api_accessible :common do |t|
      t.add :kind
      t.add :etag
      t.add :uuid
      t.add :created_by_client
      t.add :created_by_user
      t.add :created_at
      t.add :modified_by_client
      t.add :modified_by_user
      t.add :modified_at
      t.add :updated_at
    end
  end

  module ClassMethods
  end
end
