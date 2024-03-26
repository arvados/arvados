# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

class Workflow < ArvadosModel
  include HasUuid
  include KindAndEtag
  include CommonApiTemplate

  validate :validate_definition
  before_save :set_name_and_description

  api_accessible :user, extend: :common do |t|
    t.add :name
    t.add :description
    t.add :definition
  end

  def validate_definition
    begin
      @definition_yaml = YAML.safe_load self.definition if !definition.nil?
    rescue => e
      errors.add :definition, "is not valid yaml: #{e.message}"
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
    super - ["definition"]
  end

  def self.limit_index_columns_read
    ["definition"]
  end
end
