require 'zenweb'

module ZenwebLiquid
  VERSION = '0.0.1'
end

module Zenweb
  class Page
    
    ##
    # Render a page's liquid and return the intermediate result
    def render_liquid page, content, binding = TOPLEVEL_BINDING
      require 'liquid'
      
      unless defined? @liquid_template then
        @liquid_template = Liquid::Template.parse(content).render()
      end
      
      @liquid_template.render(binding)
    end
  end
end
