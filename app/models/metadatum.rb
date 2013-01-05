class Metadatum < ActiveRecord::Base
  serialize :info, Hash

  before_validation :populate_native_target
  include AssignUuid

  def info
    @info ||= Hash.new
    super
  end

  protected

  def populate_native_target
    begin
      class_name = target_kind.
        sub(/^orvos#/,'').
        classify
      self.native_target_type = class_name
      self.native_target_id = class_name.
        constantize.
        where('uuid = ?', target_uuid).
        first.
        id
    rescue
      self.native_target_type = nil
      self.native_target_id = nil
    end
  end
end
