# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

module FakeWebsocketHelper
  def use_fake_websocket_driver
    Capybara.current_driver = :poltergeist_with_fake_websocket
  end

  def fake_websocket_event(logdata)
    stamp = Time.now.utc.in_time_zone.as_json
    defaults = {
      owner_uuid: api_fixture('users')['system_user']['uuid'],
      event_at: stamp,
      created_at: stamp,
      updated_at: stamp,
    }
    event = {data: Oj.dump(defaults.merge(logdata), mode: :compat)}
    script = '$(window).data("arv-websocket").onmessage('+Oj.dump(event, mode: :compat)+');'
    page.evaluate_script(script)
  end
end
