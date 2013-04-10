module KindAndEtag

  def self.included(base)
    base.extend(ClassMethods)
  end

  module ClassMethods
  end

  def kind
    'orvos#' + self.class.to_s.underscore
  end

  def etag
    Digest::MD5.hexdigest(self.inspect).to_i(16).to_s(36)
  end
end
