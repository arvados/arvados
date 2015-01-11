module VersionHelper
  # api_version returns the git commit hash for the API server's
  # current version.  It is extracted from api_version_text, which
  # returns the source_version provided by the discovery document and
  # may have the word "-modified" appended to it (if the API server is
  # running from a locally modified repository).

  def api_version
    api_version_text.sub(/[^[:xdigit:]].*/, '')
  end

  def api_version_text
    arvados_api_client.discovery[:source_version]
  end

  # wb_version and wb_version_text provide the same strings for the
  # code version that this Workbench is currently running.

  def wb_version
    Rails.configuration.source_version
  end

  def wb_version_text
    wb_version + (Rails.configuration.local_modified or '')
  end

  def version_link_target version
    "https://arvados.org/projects/arvados/repository/changes?rev=#{version}"
  end
end
