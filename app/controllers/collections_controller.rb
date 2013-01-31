class CollectionsController < ApplicationController
  def index
    @links = Link.where(head_kind: 'orvos#collection') |
      Link.where(tail_kind: 'orvos#collection')
  end
end
