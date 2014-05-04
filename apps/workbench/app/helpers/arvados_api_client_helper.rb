module ArvadosApiClientHelper
  def arvados_api_client
    ArvadosApiClient.new_or_current
  end
end
