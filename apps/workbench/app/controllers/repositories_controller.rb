class RepositoriesController < ApplicationController
  before_filter :set_share_links, if: -> { defined? @object }

  def index_pane_list
    %w(recent help)
  end

  def show_pane_list
    if @user_is_manager
      super | %w(Sharing)
    else
      super
    end
  end
end
