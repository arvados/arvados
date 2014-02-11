require 'zenweb'

module ZenwebTextile
  VERSION = '0.0.1'
end

module Zenweb
  class Page
    
    ##
    # Render a page's textile and return the resulting html
    def render_textile page, content
      require 'RedCloth'
      RedCloth.new(content ? content : self.body).to_html
    end
  end
end
