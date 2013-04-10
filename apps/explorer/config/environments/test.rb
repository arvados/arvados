Vcffarm::Application.configure do
  # Settings specified here will take precedence over those in config/application.rb

  # The test environment is used exclusively to run your application's
  # test suite. You never need to work with it otherwise. Remember that
  # your test database is "scratch space" for the test suite and is wiped
  # and recreated between test runs. Don't rely on the data there!
  config.cache_classes = true

  # Configure static asset server for tests with Cache-Control for performance
  config.serve_static_assets = true
  config.static_cache_control = "public, max-age=3600"

  # Log error messages when you accidentally call methods on nil
  config.whiny_nils = true

  # Show full error reports and disable caching
  config.consider_all_requests_local       = true
  config.action_controller.perform_caching = false

  # Raise exceptions instead of rendering exception templates
  config.action_dispatch.show_exceptions = false

  # Disable request forgery protection in test environment
  config.action_controller.allow_forgery_protection    = false

  # Tell Action Mailer not to deliver emails to the real world.
  # The :test delivery method accumulates sent emails in the
  # ActionMailer::Base.deliveries array.
  config.action_mailer.delivery_method = :test

  # Raise exception on mass assignment protection for Active Record models
  config.active_record.mass_assignment_sanitizer = :strict

  # Print deprecation notices to the stderr
  config.active_support.deprecation = :stderr

  config.orvos_login_base = 'http://orvos/login'
  config.orvos_v1_base = 'https://orvos/orvos/v1'

  config.data_import_dir = '/data/vcffarm-upload'
  config.data_export_dir = '/data/vcffarm-download'

  # Authentication stub: hard code pre-approved API tokens.
  # config.accept_api_token = { rand(2**256).to_s(36) => true }
  config.accept_api_token = {}

  config.vcf_pipeline_uuid = '9ujm1-mxsvm-o62u4mdoxvs0ckp'

  config.site_name = 'vcffarm.example.com'
end
