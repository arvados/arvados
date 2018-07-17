# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

require 'integration_helper'
require 'webrick'

class LinkAccountTest < ActionDispatch::IntegrationTest
  setup do
    need_javascript
  end

  def start_sso_stub token
    port = available_port('sso_stub')

    s = WEBrick::HTTPServer.new(
      :Port => port,
      :BindAddress => 'localhost',
      :Logger => WEBrick::Log.new('/dev/null', WEBrick::BasicLog::DEBUG),
      :AccessLog => [nil,nil]
    )

    s.mount_proc("/login"){|req, res|
      res.set_redirect(WEBrick::HTTPStatus::TemporaryRedirect, req.query["return_to"] + "&api_token=#{token}")
      s.shutdown
    }

    s.mount_proc("/logout"){|req, res|
      res.set_redirect(WEBrick::HTTPStatus::TemporaryRedirect, req.query["return_to"])
    }

    Thread.new do
      s.start
    end

    "http://localhost:#{port}/"
  end

  test "Add another login to this account" do
    visit page_with_token('active_trustedclient')
    stub = start_sso_stub(api_fixture('api_client_authorizations')['project_viewer_trustedclient']['api_token'])
    Rails.configuration.arvados_login_base = stub + "login"

    find("#notifications-menu").click
    assert_text "active-user@arvados.local"

    find("a", text: "Link account").click
    find("button", text: "Add another login to this account").click

    find("#notifications-menu").click
    assert_text "project-viewer@arvados.local"

    find("button", text: "Link accounts").click

    find("#notifications-menu").click
    assert_text "active-user@arvados.local"
  end

  test "Use this login to access another account" do
    visit page_with_token('project_viewer_trustedclient')
    stub = start_sso_stub(api_fixture('api_client_authorizations')['active_trustedclient']['api_token'])
    Rails.configuration.arvados_login_base = stub + "login"

    find("#notifications-menu").click
    assert_text "project-viewer@arvados.local"

    find("a", text: "Link account").click
    find("button", text: "Use this login to access another account").click

    find("#notifications-menu").click
    assert_text "active-user@arvados.local"

    find("button", text: "Link accounts").click

    find("#notifications-menu").click
    assert_text "active-user@arvados.local"
  end

  test "Link login of inactive user to this account" do
    visit page_with_token('active_trustedclient')
    stub = start_sso_stub(api_fixture('api_client_authorizations')['inactive_uninvited_trustedclient']['api_token'])
    Rails.configuration.arvados_login_base = stub + "login"

    find("#notifications-menu").click
    assert_text "active-user@arvados.local"

    find("a", text: "Link account").click
    find("button", text: "Add another login to this account").click

    find("#notifications-menu").click
    assert_text "inactive-uninvited-user@arvados.local"

    find("button", text: "Link accounts").click

    find("#notifications-menu").click
    assert_text "active-user@arvados.local"
  end

  test "Cannot link to inactive user" do
    visit page_with_token('active_trustedclient')
    stub = start_sso_stub(api_fixture('api_client_authorizations')['inactive_uninvited_trustedclient']['api_token'])
    Rails.configuration.arvados_login_base = stub + "login"

    find("#notifications-menu").click
    assert_text "active-user@arvados.local"

    find("a", text: "Link account").click
    find("button", text: "Use this login to access another account").click

    find("#notifications-menu").click
    assert_text "inactive-uninvited-user@arvados.local"

    assert_text "Cannot link active-user@arvados.local"

    assert find("#link-account-submit")['disabled']

    find("button", text: "Cancel").click

    find("#notifications-menu").click
    assert_text "active-user@arvados.local"
  end

  test "Inactive user can link to active account" do
    visit page_with_token('inactive_uninvited_trustedclient')
    stub = start_sso_stub(api_fixture('api_client_authorizations')['active_trustedclient']['api_token'])
    Rails.configuration.arvados_login_base = stub + "login"

    find("#notifications-menu").click
    assert_text "inactive-uninvited-user@arvados.local"

    assert_text "Already have an account with a different login?"

    find("a", text: "Link this login to your existing account").click

    assert_no_text "Add another login to this account"

    find("button", text: "Use this login to access another account").click

    find("#notifications-menu").click
    assert_text "active-user@arvados.local"

    find("button", text: "Link accounts").click

    find("#notifications-menu").click
    assert_text "active-user@arvados.local"
  end

  test "Admin cannot link to non-admin" do
    visit page_with_token('admin_trustedclient')
    stub = start_sso_stub(api_fixture('api_client_authorizations')['active_trustedclient']['api_token'])
    Rails.configuration.arvados_login_base = stub + "login"

    find("#notifications-menu").click
    assert_text "admin@arvados.local"

    find("a", text: "Link account").click
    find("button", text: "Use this login to access another account").click

    find("#notifications-menu").click
    assert_text "active-user@arvados.local"

    assert_text "Cannot link admin account admin@arvados.local"

    assert find("#link-account-submit")['disabled']

    find("button", text: "Cancel").click

    find("#notifications-menu").click
    assert_text "admin@arvados.local"
  end

end
