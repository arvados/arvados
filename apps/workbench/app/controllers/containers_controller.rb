class ContainersController < ApplicationController
  def show_pane_list
    %w(Status Log Advanced)
  end
end
