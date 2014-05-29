class Group < ArvadosBase
  def self.goes_in_folders?
    true
  end

  def contents params={}
    res = arvados_api_client.api self.class, "/#{self.uuid}/contents", {
      _method: 'GET'
    }.merge(params)
    ret = ArvadosResourceList.new
    ret.results = arvados_api_client.unpack_api_response(res)
    ret
  end

  def class_for_display
    group_class == 'folder' ? 'Folder' : super
  end

  def editable?
    respond_to?(:writable_by) and
      writable_by and
      writable_by.index(current_user.uuid)
  end
end
