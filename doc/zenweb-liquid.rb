require 'zenweb'

module ZenwebLiquid
  VERSION = '0.0.1'
end

module Zenweb

  class Page

    def render_liquid page, content
      liquid self.body, content, page, binding
    end
    
    ##
    # Render a page's liquid and return the intermediate result
    def liquid template, content, page, binding = TOPLEVEL_BINDING
      require 'liquid'
      Liquid::Template.file_system = Liquid::LocalFileSystem.new(File.join(File.dirname(Rake.application().rakefile), "_includes"))
      unless defined? @liquid_template
        @liquid_template = Liquid::Template.parse(template)
      end
      
      vars = {}
      vars["content"] = content

      vars["site"] = site.config.h.clone
      pages = {}
      site.pages.each do |f, p|
        pages[f] = p.config.h.clone
        pages[f]["url"] = p.url
      end
      vars["site"]["pages"] = pages

      vars["page"] = page.config.h.clone
      vars["page"]["url"] = page.url
      
      @liquid_template.render(vars)
    end
  end
end
