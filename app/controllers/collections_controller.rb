class CollectionsController < ApplicationController
  before_filter :ensure_current_user_is_admin

  def graph
    index
  end

  def index
    @links = Link.eager.where(head_kind: 'orvos#collection') |
      Link.eager.where(tail_kind: 'orvos#collection')
    @collections = {}
    @links.each do |l|
      if l.head_kind == 'orvos#collection'
        c = (@collections[l.head_uuid] ||= {uuid: l.head_uuid})
        if l.link_class == 'resources' and l.name == 'wants'
          if l.head.respond_to? :created_at
            c[:created_at] = l.head.created_at
          end
          c[:wanted] = true
          if l.owner == current_user.uuid
            c[:wanted_by_me] = true
          end
        end
      end
      if l.tail_kind == 'orvos#collection'
        c = (@collections[l.tail_uuid] ||= {uuid: l.tail_uuid})
        if l.link_class == 'group' and l.name == 'member_of'
          c[:projects] ||= {}
          c[:projects][l.tail_uuid] = true
        end
        if l.link_class == 'data_origin'
          c[:origin] = l
        end
      end
    end
  end
end
