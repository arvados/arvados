require 'can_be_an_owner'

class Group < ArvadosModel
  include AssignUuid
  include KindAndEtag
  include CommonApiTemplate
  include CanBeAnOwner

  api_accessible :user, extend: :common do |t|
    t.add :name
    t.add :group_class
    t.add :description
    t.add :writable_by
  end
end
