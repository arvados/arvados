# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

require 'test_helper'

class CredentialsApiTest < ActionDispatch::IntegrationTest
  fixtures :all

  def credential_create_helper
    post "/arvados/v1/credentials",
         params: {:format => :json,
                  credential: {
                    name: "test credential",
                    description: "the credential for test",
                    credential_class: "basic_auth",
                    external_id: "my_username",
                    secret: "my_password",
                    expires_at: Time.now+2.weeks
                  }
                 },
         headers: auth(:active),
         as: :json
    assert_response :success
    json_response
  end

  test "credential create and query" do
    jr = credential_create_helper

    # fields other than secret is are returned by the API
    assert_equal "test credential", jr["name"]
    assert_equal "the credential for test", jr["description"]
    assert_equal "basic_auth", jr["credential_class"]
    assert_equal "my_username", jr["external_id"]
    assert_nil jr["secret"]

    # secret is not returned by the API
    get "/arvados/v1/credentials/#{jr['uuid']}", headers: auth(:active)
    assert_response :success
    jr = json_response
    assert_equal "test credential", jr["name"]
    assert_equal "the credential for test", jr["description"]
    assert_equal "basic_auth", jr["credential_class"]
    assert_equal "my_username", jr["external_id"]
    assert_nil jr["secret"]

    # can get credential from the database and it has the password
    assert_equal "my_password", Credential.find_by_uuid(jr["uuid"]).secret

    # secret cannot appear in queries
    get "/arvados/v1/credentials",
        params: {:format => :json,
                 :filters => [["secret", "=", "my_password"]].to_json,
                },
        headers: auth(:active)
    assert_response 403
    assert_match(/Cannot filter on 'secret'/, json_response["errors"][0])

    get "/arvados/v1/credentials",
        params: {:format => :json,
                 :where => {secret: "my_password"}.to_json
                },
        headers: auth(:active)
    assert_response 403
    assert_match(/Cannot use 'secret' in where clause/, json_response["errors"][0])

    get "/arvados/v1/credentials",
        params: {:format => :json,
                 :order => ["secret"].to_json
                },
        headers: auth(:active)
    assert_response 403
    assert_match(/Cannot order by 'secret'/, json_response["errors"][0])

    get "/arvados/v1/credentials",
        params: {:format => :json,
                 :where => {any: "my_password"}.to_json
                },
        headers: auth(:active)
    assert_response 200
    assert_equal [], json_response["items"]

    get "/arvados/v1/credentials",
        params: {:format => :json,
                 :filters => [["any", "=", "my_password"]].to_json
                },
        headers: auth(:active)
    assert_response 200
    assert_equal [], json_response["items"]

    get "/arvados/v1/credentials",
        params: {:format => :json,
                 :filters => [["any", "ilike", "my_pass%"]].to_json
                },
        headers: auth(:active)
    assert_response 200
    assert_equal [], json_response["items"]

  end

  test "credential fetch by container" do
    jr = credential_create_helper

    # cannot fetch secret using a regular token
    get "/arvados/v1/credentials/#{jr['uuid']}/secret", headers: auth(:active)
    assert_response 403

    get "/arvados/v1/credentials/#{jr['uuid']}/secret",
        headers: {'HTTP_AUTHORIZATION' => "Bearer #{api_client_authorizations(:running_container_auth).token}/#{containers(:running).uuid}"}
    assert_response :success
    assert_equal "my_password", json_response["secret"]

    lg = Log.where(object_uuid: jr['uuid'], event_type: "secret_access").first
    assert_equal jr["name"], lg["properties"]["name"]
    assert_equal jr["credential_class"], lg["properties"]["credential_class"]
    assert_equal jr["external_id"], lg["properties"]["external_id"]
  end

  test "credential owned by admin" do
    post "/arvados/v1/credentials",
         params: {:format => :json,
                  credential: {
                    name: "test credential",
                    description: "the credential for test",
                    credential_class: "basic_auth",
                    external_id: "my_username",
                    secret: "my_password",
                    expires_at: Time.now+2.weeks
                  }
                 },
         headers: auth(:admin),
         as: :json
    assert_response :success
    jr = json_response

    # cannot fetch secret using a regular token, even by admin
    get "/arvados/v1/credentials/#{jr['uuid']}/secret", headers: auth(:admin)
    assert_response 403

    # user 'active' can't see it
    get "/arvados/v1/credentials/#{jr['uuid']}", headers: auth(:active)
    assert_response 404

    # not readable by container run by 'active' user returns a 404
    # here like the previous check because the credential itself isn't
    # considered visible to the user
    get "/arvados/v1/credentials/#{jr['uuid']}/secret",
        headers: {'HTTP_AUTHORIZATION' => "Bearer #{api_client_authorizations(:running_container_auth).token}/#{containers(:running).uuid}"}
    assert_response 404
  end

  test "credential sharing" do
    post "/arvados/v1/credentials",
         params: {:format => :json,
                  credential: {
                    name: "test credential",
                    description: "the credential for test",
                    credential_class: "basic_auth",
                    external_id: "my_username",
                    secret: "my_password",
                    expires_at: Time.now+2.weeks
                  }
                 },
         headers: auth(:admin),
         as: :json
    assert_response :success
    jr = json_response

    # user 'active' can't see it
    get "/arvados/v1/credentials/#{jr['uuid']}", headers: auth(:active)
    assert_response 404

    # not readable by container run by 'active' user returns a 404
    # here like the previous check because the credential itself isn't
    # considered visible to the user
    get "/arvados/v1/credentials/#{jr['uuid']}/secret",
        headers: {'HTTP_AUTHORIZATION' => "Bearer #{api_client_authorizations(:running_container_auth).token}/#{containers(:running).uuid}"}
    assert_response 404

    # active user can't share
    post "/arvados/v1/links",
      params: {
        :format => :json,
        :link => {
          tail_uuid: users(:active).uuid,
          link_class: 'permission',
          name: 'can_read',
          head_uuid: jr["uuid"],
          properties: {}
        }
      },
      headers: auth(:active)
    assert_response 422

    # admin can share
    post "/arvados/v1/links",
      params: {
        :format => :json,
        :link => {
          tail_uuid: users(:active).uuid,
          link_class: 'permission',
          name: 'can_read',
          head_uuid: jr["uuid"],
          properties: {}
        }
      },
      headers: auth(:admin)
    assert_response :success

    # now the 'active' user can read it
    get "/arvados/v1/credentials/#{jr['uuid']}/secret",
        headers: {'HTTP_AUTHORIZATION' => "Bearer #{api_client_authorizations(:running_container_auth).token}/#{containers(:running).uuid}"}
    assert_response :success
  end

  test "credential expiration" do
    post "/arvados/v1/credentials",
         params: {:format => :json,
                  credential: {
                    name: "test credential",
                    description: "the credential for test",
                    credential_class: "basic_auth",
                    external_id: "my_username",
                    secret: "my_password",
                    expires_at: Time.now+5.seconds
                  }
                 },
         headers: auth(:active),
         as: :json
    assert_response :success
    jr = json_response

    get "/arvados/v1/credentials/#{jr['uuid']}/secret",
        headers: {'HTTP_AUTHORIZATION' => "Bearer #{api_client_authorizations(:running_container_auth).token}/#{containers(:running).uuid}"}
    assert_response :success
    assert_equal "my_username", json_response["external_id"]
    assert_equal "my_password", json_response["secret"]

    assert_equal "my_password", Credential.find_by_uuid(jr["uuid"]).secret

    Credential.where(uuid: jr["uuid"]).update_all(expires_at: Time.now)

    get "/arvados/v1/credentials/#{jr['uuid']}/secret",
        headers: {'HTTP_AUTHORIZATION' => "Bearer #{api_client_authorizations(:running_container_auth).token}/#{containers(:running).uuid}"}
    assert_response 403
    assert_match(/Credential has expired/, json_response["errors"][0])

    post "/sys/trash_sweep",
      headers: auth(:admin)
    assert_response :success

    assert_equal "", Credential.find_by_uuid(jr["uuid"]).secret
  end

  test "credential names are unique" do
    post "/arvados/v1/credentials",
         params: {:format => :json,
                  credential: {
                    name: "test credential",
                    description: "the credential for test",
                    credential_class: "basic_auth",
                    external_id: "my_username",
                    secret: "my_password",
                    expires_at: Time.now+2.weeks
                  }
                 },
         headers: auth(:active),
         as: :json
    assert_response :success

    post "/arvados/v1/credentials",
         params: {:format => :json,
                  credential: {
                    name: "test credential",
                    description: "the credential for test",
                    credential_class: "basic_auth",
                    external_id: "my_username",
                    secret: "my_password",
                    expires_at: Time.now+2.weeks
                  }
                 },
         headers: auth(:active),
         as: :json
    assert_response 422
    assert_match(/RecordNotUnique/, json_response["errors"][0])
  end

  test "credential expires_at must be set" do
    post "/arvados/v1/credentials",
         params: {:format => :json,
                  credential: {
                    name: "test credential",
                    description: "the credential for test",
                    credential_class: "basic_auth",
                    external_id: "my_username",
                    secret: "my_password"
                  }
                 },
         headers: auth(:active),
         as: :json
    assert_response 422
    assert_match(/NotNullViolation/, json_response["errors"][0])
  end
end
