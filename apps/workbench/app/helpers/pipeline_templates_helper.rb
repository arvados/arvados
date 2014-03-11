require 'tsort'

class Hash
  include TSort
  def tsort_each_node(&block)
    keys.sort.each(&block)
  end

  def tsort_each_child(node)
    if self[node]
      self[node][:script_parameters].sort.map do |k, v|
        if v.is_a? Hash and v[:output_of]
          yield v[:output_of].to_sym
        end
      end
    end
  end
end

module PipelineTemplatesHelper
  def self.sort_components(components)
    components.tsort
  end
end
