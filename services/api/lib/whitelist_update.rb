module WhitelistUpdate
  def check_update_whitelist permitted_fields
    attribute_names.each do |field|
      if not permitted_fields.include? field.to_sym and self.send((field.to_s + "_changed?").to_sym)
        errors.add field, "Illegal update of field #{field}"
      end
    end
  end
end
