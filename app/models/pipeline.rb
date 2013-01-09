class Pipeline < ActiveRecord::Base
  include AssignUuid
  serialize :components, Hash
end
