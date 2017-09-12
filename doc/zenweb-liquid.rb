# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: CC-BY-SA-3.0

require 'zenweb'
require 'liquid'

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

  class LiquidCode < Liquid::Include
    Syntax = /(#{Liquid::QuotedFragment}+)(\s+(?:as)\s+(#{Liquid::QuotedFragment}+))?/o

    def initialize(tag_name, markup, tokens)
      Liquid::Tag.instance_method(:initialize).bind(self).call(tag_name, markup, tokens)

      if markup =~ Syntax
        @template_name = $1
        @language = $3
        @attributes    = {}
      else
        raise SyntaxError.new("Error in tag 'code' - Valid syntax: include '[code_file]' as '[language]'")
      end
    end

    def render(context)
      require 'coderay'

      partial = load_cached_partial(context)
      html = ''

      context.stack do
        html = CodeRay.scan(partial.root.nodelist.join, @language).div
      end

      html
    end

    Liquid::Template.register_tag('code', LiquidCode)
  end

  class LiquidCodeBlock < Liquid::Block
    Syntax = /((?:as)\s+(#{Liquid::QuotedFragment}+))?/o

    def initialize(tag_name, markup, tokens)
      Liquid::Tag.instance_method(:initialize).bind(self).call(tag_name, markup, tokens)

      if markup =~ Syntax
        @language = $2
        @attributes = {}
      else
        raise SyntaxError.new("Error in tag 'code' - Valid syntax: codeblock as '[language]'")
      end
    end

    def render(context)
      require 'coderay'

      partial = super
      html = ''

      if partial[0] == '\n'
        partial = partial[1..-1]
      end

      context.stack do
        html = CodeRay.scan(partial, @language).div
      end

      "<notextile>#{html}</notextile>"
    end

    Liquid::Template.register_tag('codeblock', LiquidCodeBlock)
  end
end
