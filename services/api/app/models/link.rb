class Link < ArvadosModel
  include HasUuid
  include KindAndEtag
  include CommonApiTemplate
  serialize :properties, Hash
  before_create :permission_to_attach_to_objects
  before_update :permission_to_attach_to_objects
  before_create :assign_generic_name_if_none_given
  after_update :maybe_invalidate_permissions_cache
  after_create :maybe_invalidate_permissions_cache
  after_destroy :maybe_invalidate_permissions_cache
  attr_accessor :head_kind, :tail_kind
  validate :name_link_has_valid_name

  api_accessible :user, extend: :common do |t|
    t.add :tail_uuid
    t.add :link_class
    t.add :name
    t.add :head_uuid
    t.add :head_kind
    t.add :tail_kind
    t.add :properties
  end

  def properties
    @properties ||= Hash.new
    super
  end

  def head_kind
    if k = ArvadosModel::resource_class_for_uuid(head_uuid)
      k.kind
    end
  end

  def tail_kind
    if k = ArvadosModel::resource_class_for_uuid(tail_uuid)
      k.kind
    end
  end

  protected

  def assign_generic_name_if_none_given
    if new_record? and link_class == 'name' and (!name or name.empty?)
      # Creating a name link with no name means "invent a generic
      # name, like New Foo (1)"

      head_class = ArvadosModel::resource_class_for_uuid(head_uuid)
      if !head_class
        errors.add :name, 'cannot be automatically assigned for this uuid'
        return
      end
      name_base = "New " + head_class.to_s.underscore.humanize.downcase
      if not Link.where(link_class: link_class,
                        tail_uuid: tail_uuid,
                        name: name_base).any?
        self.name = name_base
      else
        # Find how many digits the largest N has among "New model (N)" names
        maxlen = ActiveRecord::Base.connection.
          execute("SELECT max(length(name)) maxlen FROM links "\
                  "WHERE link_class='name' "\
                  "AND tail_uuid=#{Link.sanitize(tail_uuid)} "\
                  "AND name~#{Link.sanitize "#{name_base} \\([0-9]+\\)"}")[0]
        if maxlen and maxlen['maxlen']
          # Find the largest N by sorting alphanumerically
          maxname = ActiveRecord::Base.connection.
            execute("SELECT max(name) maxname FROM links "\
                    "WHERE link_class='name' "\
                    "AND tail_uuid=#{Link.sanitize(tail_uuid)} "\
                    "AND length(name)=#{maxlen['maxlen']} "\
                    "AND name~#{Link.sanitize "#{name_base} \\([0-9]+\\)"}"
                    )[0]['maxname']
          n = maxname.match(/\(([0-9]+)\)$/)[1].to_i
          n += 1
        else
          # "New foo" is taken, but "New foo (1)" isn't.
          n = 1
        end
        self.name = name_base + " (#{n})"
      end
    end
  end

  def permission_to_attach_to_objects
    # Anonymous users cannot write links
    return false if !current_user

    # All users can write links that don't affect permissions
    return true if self.link_class != 'permission'

    # Administrators can grant permissions
    return true if current_user.is_admin

    # All users can grant permissions on objects they own
    head_obj = self.class.
      resource_class_for_uuid(self.head_uuid).
      where('uuid=?',head_uuid).
      first
    if head_obj
      return true if head_obj.owner_uuid == current_user.uuid
    end

    # Users with "can_grant" permission on an object can grant
    # permissions on that object
    has_grant_permission = self.class.
      where('link_class=? AND name=? AND tail_uuid=? AND head_uuid=?',
            'permission', 'can_grant', current_user.uuid, self.head_uuid).
      count > 0
    return true if has_grant_permission

    # Default = deny.
    false
  end

  def maybe_invalidate_permissions_cache
    if self.link_class == 'permission'
      # Clearing the entire permissions cache can generate many
      # unnecessary queries if many active users are not affected by
      # this change. In such cases it would be better to search cached
      # permissions for head_uuid and tail_uuid, and invalidate the
      # cache for only those users. (This would require a browseable
      # cache.)
      User.invalidate_permissions_cache
    end
  end

  def name_link_has_valid_name
    if link_class == 'name'
      if new_record? and (!name or name.empty?)
        # Unique name will be assigned in before_create filter
        true
      else
        unless name.is_a? String and !name.empty?
          errors.add('name', 'must be a non-empty string')
        end
      end
    else
      true
    end
  end
end
