class ContainerRequestsController < ApplicationController
  def show_pane_list
    %w(Status Log Advanced)
  end

  def cancel
    @object.update_attributes! priority: 0
  end
end
