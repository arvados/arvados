ArvadosWorkbench::Application.routes.draw do
  themes_for_rails

  resources :keep_disks
  resources :keep_services
  resources :user_agreements do
    post 'sign', on: :collection
    get 'signatures', on: :collection
  end
  get '/user_agreements/signatures' => 'user_agreements#signatures'
  get "users/setup_popup" => 'users#setup_popup', :as => :setup_user_popup
  get "users/setup" => 'users#setup', :as => :setup_user
  resources :nodes
  resources :humans
  resources :traits
  resources :api_client_authorizations
  resources :repositories
  resources :virtual_machines
  resources :authorized_keys
  resources :job_tasks
  resources :jobs do
    post 'cancel', :on => :member
  end
  match '/logout' => 'sessions#destroy', via: [:get, :post]
  get '/logged_out' => 'sessions#index'
  resources :users do
    get 'choose', :on => :collection
    get 'home', :on => :member
    get 'welcome', :on => :collection
    get 'inactive', :on => :collection
    get 'activity', :on => :collection
    get 'storage', :on => :collection
    post 'sudo', :on => :member
    post 'unsetup', :on => :member
    get 'setup_popup', :on => :member
    get 'profile', :on => :member
  end
  get '/manage_account' => 'users#manage_account'
  get "/add_ssh_key_popup" => 'users#add_ssh_key_popup', :as => :add_ssh_key_popup
  get "/add_ssh_key" => 'users#add_ssh_key', :as => :add_ssh_key
  resources :logs
  resources :factory_jobs
  resources :uploaded_datasets
  resources :groups do
    get 'choose', on: :collection
  end
  resources :specimens
  resources :pipeline_templates do
    get 'choose', on: :collection
  end
  resources :pipeline_instances do
    get 'compare', on: :collection
    post 'copy', on: :member
  end
  resources :links
  get '/collections/graph' => 'collections#graph'
  resources :collections do
    post 'set_persistent', on: :member
    get 'sharing_popup', :on => :member
    post 'share', :on => :member
    post 'unshare', :on => :member
    get 'choose', on: :collection
  end
  get('/collections/download/:uuid/:reader_token/*file' => 'collections#show_file',
      format: false)
  get '/collections/download/:uuid/:reader_token' => 'collections#show_file_links'
  get '/collections/:uuid/*file' => 'collections#show_file', :format => false
  resources :projects do
    match 'remove/:item_uuid', on: :member, via: :delete, action: :remove_item
    match 'remove_items', on: :member, via: :delete, action: :remove_items
    get 'move_items', on: :member, action: :move_items
    get 'choose', on: :collection
    post 'share_with', on: :member
  end
  resources :search do
    get 'choose', :on => :collection
  end

  post 'actions' => 'actions#post'
  get 'actions' => 'actions#show'
  get 'websockets' => 'websocket#index'

  root :to => 'projects#index'

  # Send unroutable requests to an arbitrary controller
  # (ends up at ApplicationController#render_not_found)
  match '*a', to: 'links#render_not_found', via: [:get, :post]
end
