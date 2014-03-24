require 'test_helper'

class ValidLinksTest < ActionDispatch::IntegrationTest
  fixtures :all

  test "tail must exist on update" do
    admin_auth = {'HTTP_AUTHORIZATION' => "OAuth2 #{api_client_authorizations(:admin).api_token}"}

    post "/arvados/v1/links", {
      :format => :json,
      :link => {
        link_class: 'test',
        name: 'stuff',
        head_uuid: users(:active).uuid,
        tail_uuid: virtual_machines(:testvm).uuid
      }
    }, admin_auth
    assert_response :success
    u = jresponse['uuid']

    put "/arvados/v1/links/#{u}", {
      :format => :json,
      :link => {
        tail_uuid: virtual_machines(:testvm2).uuid
      }
    }, admin_auth
    assert_response :success
    #puts @response.body
    #puts jresponse['tail_uuid']
    #puts virtual_machines(:testvm2)
    assert_equal virtual_machines(:testvm2).uuid, (ActiveSupport::JSON.decode @response.body)['tail_uuid']

    put "/arvados/v1/links/#{u}", {
      :format => :json,
      :link => {
        tail_uuid: 'zzzzz-tpzed-xyzxyzxerrrorxx'
      }
    }, admin_auth
    assert_response 422
  end

end
