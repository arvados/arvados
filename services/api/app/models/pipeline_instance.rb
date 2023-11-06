# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

class PipelineInstance < ArvadosModel
  include HasUuid
  include KindAndEtag
  include CommonApiTemplate
  serialize :components, Hash
  serialize :properties, Hash
  serialize :components_summary, Hash
  belongs_to :pipeline_template,
             foreign_key: 'pipeline_template_uuid',
             primary_key: 'uuid',
             optional: true

  before_validation :bootstrap_components
  before_validation :update_state
  before_validation :verify_status
  before_validation :update_timestamps_when_state_changes
  before_create :set_state_before_save
  before_save :set_state_before_save
  before_create :create_disabled
  before_update :update_disabled

  api_accessible :user, extend: :common do |t|
    t.add :pipeline_template_uuid
    t.add :name
    t.add :components
    t.add :properties
    t.add :state
    t.add :components_summary
    t.add :description
    t.add :started_at
    t.add :finished_at
  end

  # Supported states for a pipeline instance
  States =
    [
     (New = 'New'),
     (Ready = 'Ready'),
     (RunningOnServer = 'RunningOnServer'),
     (RunningOnClient = 'RunningOnClient'),
     (Paused = 'Paused'),
     (Failed = 'Failed'),
     (Complete = 'Complete'),
    ]

  def self.limit_index_columns_read
    ["components"]
  end

  # if all components have input, the pipeline is Ready
  def components_look_ready?
    if !self.components || self.components.empty?
      return false
    end

    all_components_have_input = true
    self.components.each do |name, component|
      component['script_parameters'].andand.each do |parametername, parameter|
        parameter = { 'value' => parameter } unless parameter.is_a? Hash
        if parameter['value'].nil? and parameter['required']
          if parameter['output_of']
            next
          end
          all_components_have_input = false
          break
        end
      end
    end
    return all_components_have_input
  end

  def progress_table
    begin
      # v0 pipeline format
      nrow = -1
      components['steps'].collect do |step|
        nrow += 1
        row = [nrow, step['name']]
        if step['complete'] and step['complete'] != 0
          if step['output_data_locator']
            row << 1.0
          else
            row << 0.0
          end
        else
          row << 0.0
          if step['failed']
            self.state = Failed
          end
        end
        row << (step['warehousejob']['id'] rescue nil)
        row << (step['warehousejob']['revision'] rescue nil)
        row << step['output_data_locator']
        row << (Time.parse(step['warehousejob']['finishtime']) rescue nil)
        row
      end
    rescue
      []
    end
  end

  def progress_ratio
    t = progress_table
    return 0 if t.size < 1
    t.collect { |r| r[2] }.inject(0.0) { |sum,a| sum += a } / t.size
  end

  def self.queue
    self.where("state = 'RunningOnServer'")
  end

  def cancel(cascade: false, need_transaction: true)
    raise "No longer supported"
  end

  protected
  def bootstrap_components
    if pipeline_template and (!components or components.empty?)
      self.components = pipeline_template.components.deep_dup
    end
  end

  def update_state
    if components and progress_ratio == 1.0
      self.state = Complete
    end
  end

  def verify_status
    changed_attributes = self.changed

    if new_record? or 'components'.in? changed_attributes
      self.state ||= New
      if (self.state == New) and self.components_look_ready?
        self.state = Ready
      end
    end

    if !self.state.in?(States)
      errors.add :state, "'#{state.inspect} must be one of: [#{States.join ', '}]"
      throw(:abort)
    end
  end

  def set_state_before_save
    if self.components_look_ready? && (!self.state || self.state == New)
      self.state = Ready
    end
  end

  def update_timestamps_when_state_changes
    return if not (state_changed? or new_record?)

    case state
    when RunningOnServer, RunningOnClient
      self.started_at ||= db_current_time
    when Failed, Complete
      current_time = db_current_time
      self.started_at ||= current_time
      self.finished_at ||= current_time
    end
  end


  def create_disabled
    raise "Disabled"
  end

  def update_disabled
    raise "Disabled"
  end
end
