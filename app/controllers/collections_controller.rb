class CollectionsController < ApplicationController
  before_filter :ensure_current_user_is_admin
  skip_before_filter :find_object_by_uuid, :only => [:graph]

  def graph
    index
  end

  def index
    @links = Link.eager.limit(100).where(head_kind: 'orvos#collection') |
      Link.eager.limit(100).where(tail_kind: 'orvos#collection')
    @collections = Collection.limit(100).to_hash
    @collections.merge!(Collection.
                        limit(100).
                        where(uuid: @links.select{|x|x.head_kind=='orvos#collection'}.collect(&:head_uuid) |
                              @links.select{|x|x.tail_kind=='orvos#collection'}.collect(&:tail_uuid)).
                        to_hash)
    @collection_info = {}
    @collections.each do |uuid, c|
      ci = (@collection_info[uuid] ||= {uuid: uuid})
      ci[:created_at] = c.created_at
    end
    @links.each do |l|
      if l.head_kind == 'orvos#collection'
        c = (@collection_info[l.head_uuid] ||= {uuid: l.head_uuid})
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
        c = (@collection_info[l.tail_uuid] ||= {uuid: l.tail_uuid})
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

  def show
    return super if !@object
    @provenance = []
    @output2job = {}
    @output2colorindex = {}
    @sourcedata = {params[:uuid] => {uuid: params[:uuid]}}
    @protected = {}
    whence = `whence #{params[:uuid]}`
    colorindex = -1
    whence.split("\n").each do |line|
      if line.match /^(\#\d+@\S+)$/
        job = Job.where(submit_id: line).first
        @provenance << {job: job, target: line}
      elsif (re = line.match /^ +output *= *(\S+)/)
        if !@provenance.empty?
          @provenance[-1][:output] = re[1]
          @output2job[re[1]] = @provenance[-1][:job]
          if !@output2colorindex[re[1]]
            @output2colorindex[re[1]] = (colorindex += 1) % 10
          end
          @sourcedata.delete re[1]
        end
      elsif (re = line.match /^([0-9a-f]{32}\b)/)
        @sourcedata[re[1]] ||= {uuid: re[1]}
      end
    end
    Link.where(head_uuid: @sourcedata.keys | @output2job.keys).each do |link|
      if link.link_class == 'resources' and link.name == 'wants'
        @protected[link.head_uuid] = true
      end
    end
    Link.where(tail_uuid: @sourcedata.keys).each do |link|
      if link.link_class == 'data_origin'
        @sourcedata[link.tail_uuid][:data_origins] ||= []
        @sourcedata[link.tail_uuid][:data_origins] << [link.name, link.head_kind, link.head_uuid]
      end
    end
    Collection.where(uuid: @sourcedata.keys).each do |collection|
      if @sourcedata[collection.uuid]
        @sourcedata[collection.uuid][:collection] = collection
      end
    end
  end
end
