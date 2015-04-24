require "arvados/keep"

class PipelineInstance < ArvadosBase
  attr_accessor :pipeline_template

  def self.goes_in_projects?
    true
  end

  def friendly_link_name lookup=nil
    pipeline_name = self.name
    if pipeline_name.nil? or pipeline_name.empty?
      template = if lookup and lookup[self.pipeline_template_uuid]
                   lookup[self.pipeline_template_uuid]
                 else
                   PipelineTemplate.where(uuid: self.pipeline_template_uuid).first
                 end
      if template
        template.name
      else
        self.uuid
      end
    else
      pipeline_name
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

  def editable_attributes
    %w(name description components)
  end

  def attribute_editable?(name, ever=nil)
    if name.to_s == "components"
      (ever or %w(New Ready).include?(state)) and super
    else
      super
    end
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

  def textile_attributes
    [ 'description' ]
  end

  def job_uuids
    components_map { |cspec| cspec[:job][:uuid] rescue nil }
  end

  def job_log_ids
    components_map { |cspec| cspec[:job][:log] rescue nil }
  end

  def stderr_log_object_uuids
    result = job_uuids.values.compact
    result << uuid
  end

  def stderr_log_query(limit=nil)
    query = Log.
      where(event_type: "stderr",
            object_uuid: stderr_log_object_uuids).
      order("id DESC")
    unless limit.nil?
      query = query.limit(limit)
    end
    query
  end

  def stderr_log_lines(limit=2000)
    stderr_log_query(limit).results.reverse.
      flat_map { |log| log.properties[:text].split("\n") rescue [] }
  end

  def has_readable_logs?
    log_pdhs, log_uuids = job_log_ids.values.compact.partition do |loc_s|
      Keep::Locator.parse(loc_s)
    end
    if log_pdhs.any? and
        Collection.where(portable_data_hash: log_pdhs).limit(1).results.any?
      true
    elsif log_uuids.any? and
        Collection.where(uuid: log_uuids).limit(1).results.any?
      true
    else
      stderr_log_query(1).results.any?
    end
  end

  private

  def components_map
    Hash[components.map { |cname, cspec| [cname, yield(cspec)] }]
  end
end
