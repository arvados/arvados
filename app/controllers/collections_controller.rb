class CollectionsController < ApplicationController
  before_filter :ensure_current_user_is_admin

  def graph
    index
  end

  def index
    @links = Link.where(head_kind: 'orvos#collection') |
      Link.where(tail_kind: 'orvos#collection')
  end
end
