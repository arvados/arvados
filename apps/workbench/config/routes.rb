ArvadosWorkbench::Application.routes.draw do
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
  end
  resources :logs
  resources :factory_jobs
  resources :uploaded_datasets
  resources :groups
  resources :specimens
  resources :pipeline_templates
  resources :pipeline_instances
  resources :links
  match '/collections/graph' => 'collections#graph'
  resources :collections
  root :to => 'users#welcome'

  # Send unroutable requests to an arbitrary controller
  # (ends up at ApplicationController#render_not_found)
  match '*a', :to => 'links#render_not_found'
end
