class RepositoriesController < ApplicationController
  before_filter :set_share_links, if: -> { defined? @object }

  def index_pane_list
    %w(recent help)
  end

  def show_pane_list
    if @user_is_manager
      panes = super | %w(Sharing)
      panes.insert(panes.length-1, panes.delete_at(panes.index('Advanced'))) if panes.index('Advanced')
      panes
    else
      panes = super
    end
    panes.delete('Attributes') if !current_user.is_admin
    panes
  end

  def show_tree
    @commit = params[:commit]
    @path = params[:path] || ''
    @subtree = @object.ls_subtree @commit, @path.chomp('/')
  end

  def show_blob
    @commit = params[:commit]
    @path = params[:path]
    @blobdata = @object.cat_file @commit, @path
  end

  def show_commit
    @commit = params[:commit]
  end
end
