module VersionHelper
  include ArvadosApiClientHelper

  def api_version()
    arvados_api_client.discovery[:source_version]
  end

  def wb_version()
    Rails.configuration.source_version
  end

  def wb_version_text()
    wbv = wb_version
    wbv += Rails.configuration.local_modified if Rails.configuration.local_modified
    wbv
  end
end
