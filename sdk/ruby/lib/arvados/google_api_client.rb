require 'google/api_client'
require 'json'
require 'tempfile'

class Google::APIClient
  def discovery_document(api, version)
    api = api.to_s
    discovery_uri = self.discovery_uri(api, version)
    discovery_uri_hash = Digest::MD5.hexdigest(discovery_uri)
    discovery_cache_path =
      File.expand_path("~/.cache/arvados/discovery-#{discovery_uri_hash}.json")
    @discovery_documents[discovery_uri_hash] ||=
      disk_cached_discovery_document(discovery_cache_path) or
      fetched_discovery_document(discovery_uri, discovery_cache_path)
  end

  private

  def disk_cached_discovery_document(cache_path)
    begin
      if (Time.now - File.mtime(cache_path)) < 86400
        open(cache_path) do |cache_file|
          return JSON.load(cache_file)
        end
      end
    rescue IOError, SystemCallError, JSON::JSONError
      # Error reading the cache.  Act like it doesn't exist.
    end
    nil
  end

  def write_cached_discovery_document(cache_path, body)
    cache_dir = File.dirname(cache_path)
    cache_file = nil
    begin
      FileUtils.makedirs(cache_dir)
      cache_file = Tempfile.new("discovery", cache_dir)
      cache_file.write(body)
      cache_file.flush
      File.rename(cache_file.path, cache_path)
    rescue IOError, SystemCallError
      # Failure to write the cache is non-fatal.  Do nothing.
    ensure
      cache_file.close! unless cache_file.nil?
    end
  end

  def fetched_discovery_document(uri, cache_path)
    response = self.execute!(:http_method => :get,
                             :uri => uri,
                             :authenticated => false)
    write_cached_discovery_document(cache_path, response.body)
    JSON.load(response.body)
  end
end
