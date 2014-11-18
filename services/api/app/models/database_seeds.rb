class DatabaseSeeds
  include CurrentApiClient
  def self.install
    system_user
    system_group
    anonymous_group
    anonymous_user
    empty_collection
  end
end

