class PipelineInstance < ArvadosModel
  include AssignUuid
  include KindAndEtag
  include CommonApiTemplate
  serialize :components, Hash
  serialize :properties, Hash
  serialize :components_summary, Hash
  belongs_to :pipeline_template, :foreign_key => :pipeline_template_uuid, :primary_key => :uuid

  before_validation :bootstrap_components
  before_validation :update_success
  before_create :set_state_for_new_pipeline
  before_save :set_state_for_new_pipeline

  api_accessible :user, extend: :common do |t|
    t.add :pipeline_template_uuid
    t.add :pipeline_template, :if => :pipeline_template
    t.add :name
    t.add :components
    t.add :success
    t.add :active
    t.add :state
    t.add :dependencies
    t.add :properties
    t.add :components_summary
  end

  # Supported states for a pipeline instance
  New = 'New'
  Ready = 'Ready'
  RunningOnServer = 'RunningOnServer'
  RunningOnClient = 'RunningOnClient'
  Paused = 'Paused'
  Failed = 'Failed'
  Complete = 'Complete'

  def dependencies
    dependency_search(self.components).keys
  end

  def active
    self.state == RunningOnServer || self.state == RunningOnClient      
  end

  def success
    self.state == Complete      
  end

  def set_state state
    self.state = state
  end

  def set_state_for_new_pipeline
    if !self.state || self.state == New
      if PipelineInstance.is_ready self.components
        self.state = Ready
      else
        self.state = New
      end
    end
  end

  # if a legacy client tries to update active or success attributes, convert to state
  def update_attribute name, value
    if name == 'success'
      if value == true
        self.state = Complete
      else
        self.state = Failed
      end

      name = 'state'
      value = self.state
    elsif name == 'active'
      if value == true
        self.state = RunningOnServer
      else
        self.state = New
      end

      name = 'state'
      value = self.state
    end

    super
  end

  # if all components have input, the pipeline is Ready
  def self.is_ready components
    if !components || components.empty?  # is this correct?
      return true
    end

    all_components_have_input = true
    components.each do |name, component|
      if !component.andand['script_parameters'].andand['input'] || 
          component.andand['script_parameters'].andand['input'].empty?
        all_components_have_input = false
        break
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
    self.where("state = 'Ready' and state != 'RunningOnClient'")
  end

  protected
  def bootstrap_components
    if pipeline_template and (!components or components.empty?)
      self.components = pipeline_template.components.deep_dup
    end
  end

  def update_success
    if components and progress_ratio == 1.0
      self.success = true
    end
  end

  def dependency_search(haystack)
    if haystack.is_a? String
      if (re = haystack.match /^([0-9a-f]{32}(\+[^,]+)*)+/)
        {re[1] => true}
      else
        {}
      end
    elsif haystack.is_a? Array
      deps = {}
      haystack.each do |value|
        deps.merge! dependency_search(value)
      end
      deps
    elsif haystack.respond_to? :keys
      deps = {}
      haystack.each do |key, value|
        deps.merge! dependency_search(value)
      end
      deps
    else
      {}
    end
  end
end
