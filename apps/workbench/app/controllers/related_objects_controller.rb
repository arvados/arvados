class RelatedObjectsController < ApplicationController

  def model_class
    Collection
  end

  def breadcrumb_controller_name
    'search'
  end

  def index
    @objects = []
    unless params[:q].andand.length.andand > 0
      return super
    end
    @links = Link.where(any: ['contains', params[:q]])
    [Collection, Group, Human, Specimen, Trait].each do |klass|
      @objects += klass.where(any: ['contains', params[:q]]).to_a
      @objects += klass.where(uuid: (@links.collect(&:head_uuid) | @links.collect(&:tail_uuid)))
    end
    @objects = @objects.uniq_by { |x| x.uuid }.sort_by { |x| x.modified_at }.reverse
    @detail_for = {}
    @objects.each do |object|
      uuid = object.uuid
      @detail_for[uuid] ||= []
      if object.respond_to? :files
        object.files[0..2].each do |file|
          @detail_for[uuid] += ["#{file[0]}/#{file[1]}"]
        end
      end
    end
    @links.each do |link|
      [link.head_uuid, link.tail_uuid].each do |uuid|
        if @detail_for[uuid]
          @detail_for[uuid] += ["#{link.link_class}: #{link.name}"]
        end
      end
    end
    super
  end

  def attributes_for_display
    unless @attributes_for_display
      ret = []
      @objects.each do |object|
        ret |= object.attributes_for_display.collect { |k,v| k }
      end
      @attributes_for_display = ret
    end
    @attributes_for_display
  end
end
