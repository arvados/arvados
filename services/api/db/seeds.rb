# This file seeds the database with initial/default values.
#
# It is invoked by `rake db:seed` and `rake db:setup`.

# These two methods would create the system user and group objects on
# demand later anyway, but it's better form to create them up front.
include CurrentApiClient
system_user
system_group
