class KeepDisksController < ApplicationController
  def create
    defaults = { is_readable: true, is_writable: true }
    @object = KeepDisk.new defaults.merge(params[:keep_disk] || {})
    super
  end
end
