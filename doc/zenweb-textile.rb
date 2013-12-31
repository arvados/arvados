require 'zenweb'

module ZenwebTextile
  VERSION = '0.0.1'
end

module Zenweb
  class Page
    
    ##
    # Render a page's textile and return the resulting html
    def render_textile page, content
      textile body
    end

    ##
    # Render textile in +content+ 
    def textile content
      require 'RedCloth'
      @textile = RedCloth.new(content).to_html
    end
  end
end
