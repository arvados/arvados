class PipelineInstance < ArvadosBase
  attr_accessor :pipeline_template

  def self.goes_in_projects?
    true
  end

  def friendly_link_name
    pipeline_name = self.name
    if pipeline_name.nil? or pipeline_name.empty?
      return PipelineTemplate.where(uuid: self.pipeline_template_uuid).first.name
    else
      return pipeline_name
    end
  end

  def content_summary
    begin
      PipelineTemplate.find(pipeline_template_uuid).name
    rescue
      super
    end
  end

  def update_job_parameters(new_params)
    self.components[:steps].each_with_index do |step, i|
      step[:params].each do |param|
        if new_params.has_key?(new_param_name = "#{i}/#{param[:name]}") or
            new_params.has_key?(new_param_name = "#{step[:name]}/#{param[:name]}") or
            new_params.has_key?(new_param_name = param[:name])
          param_type = :value
          %w(hash data_locator).collect(&:to_sym).each do |ptype|
            param_type = ptype if param.has_key? ptype
          end
          param[param_type] = new_params[new_param_name]
        end
      end
    end
  end

  def attribute_editable? attr, *args
    super && (attr.to_sym == :name ||
              (attr.to_sym == :components and
               (self.state == 'New' || self.state == 'Ready')))
  end

  def attributes_for_display
    super.reject { |k,v| k == 'components' }
  end

  def self.creatable?
    false
  end

  def component_input_title(component_name, input_name)
    component = components[component_name]
    return nil if component.nil?
    param_info = component[:script_parameters].andand[input_name.to_sym]
    if param_info.is_a?(Hash) and param_info[:title]
      param_info[:title]
    else
      "\"#{input_name.to_s}\" parameter for #{component[:script]} script in #{component_name} component"
    end
  end
end
