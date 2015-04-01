class Repository < ArvadosModel
  include HasUuid
  include KindAndEtag
  include CommonApiTemplate

  # Order is important here.  We must validate the owner before we can
  # validate the name.
  validate :valid_owner
  validate :name_format, :if => Proc.new { |r| r.errors[:owner_uuid].empty? }
  validates(:name, uniqueness: true, allow_nil: false)

  api_accessible :user, extend: :common do |t|
    t.add :name
    t.add :fetch_url
    t.add :push_url
  end

  def self.attributes_required_columns
    super.merge({"push_url" => ["name"], "fetch_url" => ["name"]})
  end

  def push_url
    "git@git.%s.arvadosapi.com:%s.git" % [Rails.configuration.uuid_prefix, name]
  end

  def fetch_url
    push_url
  end

  def server_path
    # Find where the repository is stored on the API server's filesystem,
    # and return that path, or nil if not found.
    # This method is only for the API server's internal use, and should not
    # be exposed through the public API.  Following our current gitolite
    # setup, it searches for repositories stored by UUID, then name; and it
    # prefers bare repositories over checkouts.
    [["%s.git"], ["%s", ".git"]].each do |repo_base, *join_args|
      [:uuid, :name].each do |path_attr|
        git_dir = File.join(Rails.configuration.git_repositories_dir,
                            repo_base % send(path_attr), *join_args)
        return git_dir if File.exist?(git_dir)
      end
    end
    nil
  end

  protected

  def permission_to_update
    if not super
      false
    elsif current_user.is_admin
      true
    elsif name_changed?
      current_user.uuid == owner_uuid
    else
      true
    end
  end

  def owner
    User.find_by_uuid(owner_uuid)
  end

  def valid_owner
    if owner.nil? or (owner.username.nil? and (owner.uuid != system_user_uuid))
      errors.add(:owner_uuid, "must refer to a user with a username")
      false
    end
  end

  def name_format
    if owner.uuid == system_user_uuid
      prefix_match = ""
      errmsg_start = "must be"
    else
      prefix_match = Regexp.escape(owner.username + "/")
      errmsg_start = "must be the owner's username, then '/', then"
    end
    if not /^#{prefix_match}[A-Za-z][A-Za-z0-9]*$/.match(name)
      errors.add(:name,
                 "#{errmsg_start} a letter followed by alphanumerics")
      false
    end
  end
end
