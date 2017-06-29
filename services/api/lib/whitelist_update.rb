# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

module WhitelistUpdate
  def check_update_whitelist permitted_fields
    attribute_names.each do |field|
      if !permitted_fields.include?(field.to_sym) && really_changed(field)
        errors.add field, "cannot be modified in this state (#{send(field+"_was").inspect}, #{send(field).inspect})"
      end
    end
  end

  def really_changed(attr)
    return false if !send(attr+"_changed?")
    old = send(attr+"_was")
    new = send(attr)
    if (old.nil? || old == [] || old == {}) && (new.nil? || new == [] || new == {})
      false
    else
      old != new
    end
  end

  def validate_state_change
    if self.state_changed?
      unless state_transitions[self.state_was].andand.include? self.state
        errors.add :state, "cannot change from #{self.state_was} to #{self.state}"
        return false
      end
    end
  end
end
