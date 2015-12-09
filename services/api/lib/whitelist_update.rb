module WhitelistUpdate
  def check_update_whitelist permitted_fields
    attribute_names.each do |field|
      if not permitted_fields.include? field.to_sym and self.send((field.to_s + "_changed?").to_sym)
        errors.add field, "illegal update of field"
      end
    end
  end

  def validate_state_change
    if self.state_changed?
      unless state_transitions[self.state_was].andand.include? self.state
        errors.add :state, "invalid state change from #{self.state_was} to #{self.state}"
        return false
      end
    end
  end
end
