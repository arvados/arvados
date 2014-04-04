# The client object must be instantiated _after_ zza_load_config.rb
# runs, because it relies on configuration settings.
#
if not $application_config
  raise "Fatal: Config must be loaded before instantiating ArvadosApiClient."
end

$arvados_api_client = ArvadosApiClient.new
