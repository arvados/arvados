# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

class ContainerRequest < ArvadosBase
  def self.creatable?
    false
  end

  def textile_attributes
    [ 'description' ]
  end

  def self.goes_in_projects?
    true
  end

  def self.copies_to_projects?
    false
  end

  def work_unit(label=nil, child_objects=nil)
    ContainerWorkUnit.new(self, label, self.uuid, child_objects=child_objects)
  end

  def editable_attributes
    super + ["reuse_steps"]
  end

  def reuse_steps
    command.each do |arg|
      if arg == "--enable-reuse"
        return true
      end
    end
    false
  end

  def self.attribute_info
    self.columns
    @attribute_info[:reuse_steps] = {:type => "boolean"}
    @attribute_info
  end

end
