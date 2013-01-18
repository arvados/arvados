class OrvosModel < ActiveRecord::Base
  self.abstract_class = true

  def self.kind_class(kind)
    kind.match(/^orvos\#(.+?)(_list|List)?$/)[1].pluralize.classify.constantize rescue nil
  end

  def eager_load_associations
    self.class.columns.each do |col|
      re = col.name.match /^(.*)_kind$/
      if (re and
          self.respond_to? re[1].to_sym and
          (auuid = self.send(re[1].to_sym)) and
          (aclass = self.class.kind_class(self.send(col.name.to_sym))) and
          (aobject = aclass.where('uuid=?', auuid).first))
        self.send((re[1]+'=').to_sym, aobject)
      end
    end
  end
end
