# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

Rails.application.routes.draw do
  themes_for_rails

  # OPTIONS requests are not allowed at routes that use cookies.
  ['/auth/*a', '/login', '/logout'].each do |nono|
    match nono, to: 'user_sessions#cross_origin_forbidden', via: 'OPTIONS'
  end
  # OPTIONS at discovery and API paths get an empty response with CORS headers.
  match '/discovery/v1/*a', to: 'static#empty', via: 'OPTIONS'
  match '/arvados/v1/*a', to: 'static#empty', via: 'OPTIONS'

  namespace :arvados do
    namespace :v1 do
      resources :api_client_authorizations do
        post 'create_system_auth', on: :collection
        get 'current', on: :collection
      end
      resources :api_clients
      resources :authorized_keys
      resources :collections do
        get 'provenance', on: :member
        get 'used_by', on: :member
        post 'trash', on: :member
        post 'untrash', on: :member
      end
      resources :groups do
        get 'contents', on: :collection
        get 'contents', on: :member
        get 'shared', on: :collection
        post 'trash', on: :member
        post 'untrash', on: :member
      end
      resources :containers do
        get 'auth', on: :member
        post 'lock', on: :member
        post 'unlock', on: :member
        post 'update_priority', on: :member
        get 'secret_mounts', on: :member
        get 'current', on: :collection
      end
      resources :container_requests do
        get 'container_status', on: :member
      end
      resources :keep_services do
        get 'accessible', on: :collection
      end
      resources :links
      resources :logs
      resources :workflows
      resources :user_agreements do
        get 'signatures', on: :collection
        post 'sign', on: :collection
      end
      resources :users do
        get 'current', on: :collection
        get 'system', on: :collection
        post 'activate', on: :member
        post 'setup', on: :collection
        post 'unsetup', on: :member
        post 'merge', on: :collection
        patch 'batch_update', on: :collection
      end
      resources :virtual_machines do
        get 'logins', on: :member
        get 'get_all_logins', on: :collection
      end
      get '/permissions/:uuid', to: 'links#get_permissions'
    end
  end

  post '/sys/trash_sweep', to: 'sys#trash_sweep'

  if Rails.env == 'test'
    post '/database/reset', to: 'database#reset'
  end

  # omniauth
  match '/auth/:provider/callback', to: 'user_sessions#create', via: [:get, :post]
  match '/auth/failure', to: 'user_sessions#failure', via: [:get, :post]
  # not handled by omniauth provider -> 403 with no CORS headers.
  get '/auth/*a', to: 'user_sessions#cross_origin_forbidden'

  # Custom logout
  match '/login', to: 'user_sessions#login', via: [:get, :post]
  match '/logout', to: 'user_sessions#logout', via: [:get, :post]

  match '/discovery/v1/apis/arvados/v1/rest', to: 'arvados/v1/schema#index', via: [:get, :post]

  match '/static/login_failure', to: 'static#login_failure', as: :login_failure, via: [:get, :post]

  match '/_health/:check', to: 'arvados/v1/management#health', via: [:get]
  match '/metrics', to: 'arvados/v1/management#metrics', via: [:get]

  # Send unroutable requests to an arbitrary controller
  # (ends up at ApplicationController#render_not_found)
  match '*a', to: 'static#render_not_found', via: [:get, :post, :put, :patch, :delete, :options]

  root to: 'static#home'
end
