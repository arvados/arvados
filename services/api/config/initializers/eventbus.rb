require 'eventbus'

# See application.yml for details about configuring the websocket service.

Server::Application.configure do
  # Enables websockets if ARVADOS_WEBSOCKETS is defined with any value.  If
  # ARVADOS_WEBSOCKETS=ws-only, server will only accept websocket connections
  # and return an error response for all other requests.
  if ENV['ARVADOS_WEBSOCKETS']
    config.middleware.insert_after ArvadosApiToken, RackSocket, {
      :handler => EventBus,
      :mount => "/websocket",
      :websocket_only => (ENV['ARVADOS_WEBSOCKETS'] == "ws-only")
    }
    Rails.logger.info "Websockets #{ENV['ARVADOS_WEBSOCKETS']}, running at /websocket"

    Rails.application.config.after_initialize do
      ActiveRecord::Base.connection_pool.disconnect!

      ActiveSupport.on_load(:active_record) do
        config = ActiveRecord::Base.configurations[Rails.env] ||
                 Rails.application.config.database_configuration[Rails.env]
        config['pool'] = Rails.application.config.websocket_db_pool
        ActiveRecord::Base.establish_connection(config)
        Rails.logger.info "Database connection pool size #{Rails.application.config.websocket_db_pool}"
      end
    end

  else
    Rails.logger.info "Websockets disabled"
  end
end
