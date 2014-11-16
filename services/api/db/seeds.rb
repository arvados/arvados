# This file seeds the database with initial/default values.
#
# It is invoked by `rake db:seed` and `rake db:setup`.

# These two methods would create these objects on demand
# later anyway, but it's better form to create them up front.
include CurrentApiClient
system_user
system_group
all_users_group
anonymous_group
anonymous_user
empty_collection
