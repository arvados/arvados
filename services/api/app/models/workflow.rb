# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

class Workflow < ArvadosModel
  include HasUuid
  include KindAndEtag
  include CommonApiTemplate

  validate :validate_definition
  validate :validate_collection_uuid
  before_save :set_name_and_description
  before_save :link_with_collection

  api_accessible :user, extend: :common do |t|
    t.add :name
    t.add :description
    t.add :definition
    t.add :collection_uuid
  end

  def validate_definition
    begin
      @definition_yaml = YAML.safe_load self.definition if !definition.nil?
    rescue => e
      errors.add :definition, "is not valid yaml: #{e.message}"
    end
  end

  def validate_collection_uuid
    return if !collection_uuid_changed?

    c = Collection.
          readable_by(current_user).
          find_by_uuid(collection_uuid)
    if !c
      errors.add :collection_uuid, "does not exist or do not have permission to read."
    end

    if c.properties["type"] != "workflow"
      errors.add :collection_uuid, "properties does not have type: workflow"
    end
  end

  def set_name_and_description
    old_wf = {}
    begin
      old_wf = YAML.safe_load self.definition_was if !self.definition_was.nil?
    rescue => e
      logger.warn "set_name_and_description error: #{e.message}"
      return
    end

    ['name', 'description'].each do |a|
      if !self.changes.include?(a)
        v = self.read_attribute(a)
        if !v.present? or v == old_wf[a]
          val = @definition_yaml[a] if self.definition and @definition_yaml
          self[a] = val
        end
      end
    end
  end

  def self.full_text_searchable_columns
    super - ["definition", "collection_uuid"]
  end

  def link_with_collection
    return if collection_uuid.nil? || !collection_uuid_changed?
    Collection.find_by_uuid(collection_uuid).update_linked_workflows([self], false)
  end

  def self.readable_by(*users_list)
    return super if users_list.select { |u| u.is_a?(User) && u.is_admin }.any?
    super.where(collection_uuid: nil).or(where(Collection.readable_by(*users_list).where("collections.uuid = workflows.collection_uuid").arel.exists))
  end

end
