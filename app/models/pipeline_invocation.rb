class PipelineInvocation < ActiveRecord::Base
  include AssignUuid
  include KindAndEtag
  include CommonApiTemplate
  serialize :components, Hash
  belongs_to :pipeline, :foreign_key => :pipeline_uuid, :primary_key => :uuid

  before_validation :bootstrap_components
  before_validation :update_success

  api_accessible :superuser, :extend => :common do |t|
    t.add :pipeline_uuid
    t.add :name
    t.add :components
    t.add :success
    t.add :active
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
end
