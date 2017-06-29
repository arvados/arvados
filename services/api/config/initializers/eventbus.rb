# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

if ENV['ARVADOS_WEBSOCKETS']
  Server::Application.configure do
    Rails.logger.error "Built-in websocket server is disabled. See note (2017-03-23, e8cc0d7) at https://dev.arvados.org/projects/arvados/wiki/Upgrading_to_master"

    class EventBusRemoved
      def overloaded?
        false
      end
      def on_connect ws
        ws.on :open do |e|
          EM::Timer.new 1 do
            ws.send(SafeJSON.dump({status: 501, message: "Server misconfigured? see http://doc.arvados.org/install/install-ws.html"}))
          end
          EM::Timer.new 3 do
            ws.close
          end
        end
      end
    end

    config.middleware.insert_after(ArvadosApiToken, RackSocket, {
                                     handler: EventBusRemoved,
                                     mount: "/websocket",
                                     websocket_only: (ENV['ARVADOS_WEBSOCKETS'] == "ws-only")
                                   })
  end
end
