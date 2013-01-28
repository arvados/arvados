class PipelineInvocation < OrvosModel
  include AssignUuid
  include KindAndEtag
  include CommonApiTemplate
  serialize :components, Hash
  belongs_to :pipeline, :foreign_key => :pipeline_uuid, :primary_key => :uuid
  attr_accessor :pipeline

  before_validation :bootstrap_components
  before_validation :update_success

  api_accessible :superuser, :extend => :common do |t|
    t.add :pipeline_uuid
    t.add :pipeline, :if => :pipeline
    t.add :name
    t.add :components
    t.add :success
    t.add :active
    t.add :dependencies
  end

  def dependencies
    dependency_search(self.components).keys
  end

  def progress_table
    begin
      # v0 pipeline format
      nrow = -1
      components['steps'].collect do |step|
        nrow += 1
        row = [nrow, step['name']]
        if step['output_data_locator']
          row << 1.0
        else
          row << 0.0
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

  protected
  def bootstrap_components
    if pipeline and (!components or components.empty?)
      self.components = pipeline.components
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
