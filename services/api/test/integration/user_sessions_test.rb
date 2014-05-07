require 'test_helper'

class UserSessionsApiTest < ActionDispatch::IntegrationTest
  test 'create new user during omniauth callback' do
    mock = {
      'provider' => 'josh_id',
      'uid' => 'https://edward.example.com',
      'info' => {
        'identity_url' => 'https://edward.example.com',
        'name' => 'Edward Example',
        'first_name' => 'Edward',
        'last_name' => 'Example',
        'email' => 'edward@example.com',
      },
    }
    client_url = 'https://wb.example.com'
    post('/auth/josh_id/callback',
         {return_to: client_url},
         {'omniauth.auth' => mock})
    assert_response :redirect, 'Did not redirect to client with token'
    assert_equal(0, @response.redirect_url.index(client_url),
                 'Redirected to wrong address after succesful login: was ' +
                 @response.redirect_url + ', expected ' + client_url + '[...]')
    assert_not_nil(@response.redirect_url.index('api_token='),
                   'Expected api_token in query string of redirect url ' +
                   @response.redirect_url)
  end
end
