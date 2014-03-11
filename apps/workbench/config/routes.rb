ArvadosWorkbench::Application.routes.draw do
  themes_for_rails

  resources :keep_disks
  resources :user_agreements do
    put 'sign', on: :collection
    get 'signatures', on: :collection
  end
  get '/user_agreements/signatures' => 'user_agreements#signatures'
  resources :nodes
  resources :humans
  resources :traits
  resources :api_client_authorizations
  resources :repositories
  resources :virtual_machines
  resources :authorized_keys
  resources :job_tasks
  resources :jobs
  match '/logout' => 'sessions#destroy'
  match '/logged_out' => 'sessions#index'
  resources :users do
    get 'home', :on => :member
    get 'welcome', :on => :collection
    post 'sudo', :on => :member
  end
  resources :logs
  resources :factory_jobs
  resources :uploaded_datasets
  resources :groups
  resources :specimens
  resources :pipeline_templates
  resources :pipeline_instances do
    get 'compare', on: :collection
  end
  resources :links
  match '/collections/graph' => 'collections#graph'
  resources :collections
  get '/collections/:uuid/*file' => 'collections#show_file', :format => false

  post 'actions' => 'actions#post'

  root :to => 'users#welcome'

  # Send unroutable requests to an arbitrary controller
  # (ends up at ApplicationController#render_not_found)
  match '*a', :to => 'links#render_not_found'
end
