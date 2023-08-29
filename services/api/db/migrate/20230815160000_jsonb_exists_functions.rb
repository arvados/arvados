# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

class JsonbExistsFunctions < ActiveRecord::Migration[5.2]
  def up

    # Define functions for the "?" and "?&" operators.  We can't use
    # "?" and "?&" directly in ActiveRecord queries because "?" is
    # used for parameter substitution.
    #
    # We used to use jsonb_exists() and jsonb_exists_all() but
    # apparently Postgres associates indexes with operators but not
    # with functions, so while a query using an operator can use the
    # index, the equivalent clause using the function will always
    # perform a full row scan.
    #
    # See ticket https://dev.arvados.org/issues/20858 for examples.
    #
    # As a workaround, we can define IMMUTABLE functions, which are
    # directly inlined into the query, which then uses the index as
    # intended.
    #
    # Huge shout out to this stack overflow post that explained what
    # is going on and provides the workaround used here.
    #
    # https://dba.stackexchange.com/questions/90002/postgresql-operator-uses-index-but-underlying-function-does-not

    ActiveRecord::Base.connection.execute %{
CREATE OR REPLACE FUNCTION jsonb_exists_inline_op(jsonb, text)
RETURNS bool
LANGUAGE sql
IMMUTABLE
AS $$SELECT $1 ? $2$$
}

    ActiveRecord::Base.connection.execute %{
CREATE OR REPLACE FUNCTION jsonb_exists_all_inline_op(jsonb, text[])
RETURNS bool
LANGUAGE sql
IMMUTABLE
AS 'SELECT $1 ?& $2'
}
  end

  def down
    ActiveRecord::Base.connection.execute "DROP FUNCTION jsonb_exists_inline_op"
    ActiveRecord::Base.connection.execute "DROP FUNCTION jsonb_exists_all_inline_op"
  end
end
