require 'zenweb'

module ZenwebTextile
  VERSION = '0.0.1'
end

module Zenweb
  class Page
    alias_method :old_body, :body
    def body
      # Don't try to parse binary files as text
      if /\.(?:#{Site.binary_files.join("|")})$/ =~ path
        @body ||= File.binread path
      else
        @body ||= begin
                    _, body = Zenweb::Config.split path
                    body.strip
                  end
      end
    end
  end
end
