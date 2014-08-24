require 'test_helper'

class UserSessionsApiTest < ActionDispatch::IntegrationTest
  def client_url
    'https://wb.example.com'
  end

  def mock_auth_with_email email
    mock = {
      'provider' => 'josh_id',
      'uid' => 'https://edward.example.com',
      'info' => {
        'identity_url' => 'https://edward.example.com',
        'name' => 'Edward Example',
        'first_name' => 'Edward',
        'last_name' => 'Example',
        'email' => email,
      },
    }
    post('/auth/josh_id/callback',
         {return_to: client_url},
         {'omniauth.auth' => mock})
    assert_response :redirect, 'Did not redirect to client with token'
  end

  test 'create new user during omniauth callback' do
    mock_auth_with_email 'edward@example.com'
    assert_equal(0, @response.redirect_url.index(client_url),
                 'Redirected to wrong address after succesful login: was ' +
                 @response.redirect_url + ', expected ' + client_url + '[...]')
    assert_not_nil(@response.redirect_url.index('api_token='),
                   'Expected api_token in query string of redirect url ' +
                   @response.redirect_url)
  end

  # Test various combinations of auto_setup configuration and email
  # address provided during a new user's first session setup.
  [{result: :nope, email: nil, cfg: {auto: true, repo: true, vm: true}},
   {result: :yup, email: nil, cfg: {auto: true}},
   {result: :nope, email: '@example.com', cfg: {auto: true, repo: true, vm: true}},
   {result: :yup, email: '@example.com', cfg: {auto: true}},
   {result: :nope, email: 'root@', cfg: {auto: true, repo: true, vm: true}},
   {result: :nope, email: 'root@', cfg: {auto: true, repo: true}},
   {result: :nope, email: 'root@', cfg: {auto: true, vm: true}},
   {result: :yup, email: 'root@', cfg: {auto: true}},
   {result: :nope, email: 'gitolite@', cfg: {auto: true, repo: true}},
   {result: :nope, email: '*_*@', cfg: {auto: true, vm: true}},
   {result: :yup, email: 'toor@', cfg: {auto: true, vm: true, repo: true}},
   {result: :yup, email: 'foo@', cfg: {auto: true, vm: true},
     uniqprefix: 'foo'},
   {result: :yup, email: 'foo@', cfg: {auto: true, repo: true},
     uniqprefix: 'foo'},
   {result: :yup, email: 'auto_setup_vm_login@', cfg: {auto: true, repo: true},
     uniqprefix: 'auto_setup_vm_login'},
   ].each do |testcase|
    test "user auto-activate #{testcase.inspect}" do
      # Configure auto_setup behavior according to testcase[:cfg]
      Rails.configuration.auto_setup_new_users = testcase[:cfg][:auto]
      Rails.configuration.auto_setup_new_users_with_vm_uuid =
        (testcase[:cfg][:vm] ? virtual_machines(:testvm).uuid : false)
      Rails.configuration.auto_setup_new_users_with_repository =
        testcase[:cfg][:repo]

      mock_auth_with_email testcase[:email]
      u = assigns(:user)
      vm_links = Link.where('link_class=? and tail_uuid=? and head_uuid like ?',
                            'permission', u.uuid,
                            '%-' + VirtualMachine.uuid_prefix + '-%')
      repo_links = Link.where('link_class=? and tail_uuid=? and head_uuid like ?',
                              'permission', u.uuid,
                              '%-' + Repository.uuid_prefix + '-%')
      repos = Repository.where('uuid in (?)', repo_links.collect(&:head_uuid))
      case u[:result]
      when :nope
        assert_equal false, u.is_invited, "should not have been set up"
        assert_empty vm_links, "should not have VM login permission"
        assert_empty repo_links, "should not have repo permission"
      when :yup
        assert_equal true, u.is_invited
        if testcase[:cfg][:vm]
          assert_equal 1, vm_links.count, "wrong number of VM perm links"
        else
          assert_empty vm_links, "should not have VM login permission"
        end
        if testcase[:cfg][:repo]
          assert_equal 1, repo_links.count, "wrong number of repo perm links"
          assert_equal 1, repos.count, "wrong number of repos"
          assert_equal 'can_manage', repo_links.first.name, "wrong perm type"
        else
          assert_empty repo_links, "should not have repo permission"
        end
      end
      if (prefix = testcase[:uniqprefix])
        # This email address conflicts with a test fixture. Make sure
        # every VM login and repository name got digits added to make
        # it unique.
        (repos.collect(&:name) +
         vm_links.collect { |link| link.properties['username'] }
         ).each do |name|
          r = name.match /^(.{#{prefix.length}})(\d+)$/
          assert_not_nil r, "#{name.inspect} does not match {prefix}\\d+"
          assert_equal(prefix, r[1],
                       "#{name.inspect} was not {#{prefix.inspect} plus digits}")
        end
      end
    end
  end
end
