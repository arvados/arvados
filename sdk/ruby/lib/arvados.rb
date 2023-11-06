# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

require 'rubygems'
require 'active_support/inflector'
require 'json'
require 'fileutils'
require 'andand'
require 'net/http'

require 'arvados/google_api_client'

ActiveSupport::Inflector.inflections do |inflect|
  inflect.irregular 'specimen', 'specimens'
  inflect.irregular 'human', 'humans'
end

class Arvados
  class ArvadosClient < Google::APIClient
    attr_reader :request_id

    def execute(*args)
      @request_id = "req-" + Random.new.rand(2**128).to_s(36)[0..19]
      if args.last.is_a? Hash
        args.last[:headers] ||= {}
        args.last[:headers]['X-Request-Id'] = @request_id
      end
      begin
        super(*args)
      rescue => e
        if !e.message.match(/.*req-[0-9a-zA-Z]{20}.*/)
          raise $!, "#{$!} (Request ID: #{@request_id})", $!.backtrace
        end
        raise e
      end
    end
  end

  class TransactionFailedError < StandardError
  end

  @@debuglevel = 0
  class << self
    attr_accessor :debuglevel
  end

  def initialize(opts={})
    @application_version ||= 0.0
    @application_name ||= File.split($0).last

    @arvados_api_version = opts[:api_version] || 'v1'

    @config = nil
    [[:api_host, 'ARVADOS_API_HOST'],
     [:api_token, 'ARVADOS_API_TOKEN']].each do |op, en|
      if opts[op]
        config[en] = opts[op]
      end
      if !config[en]
        raise "#{$0}: no :#{op} or ENV[#{en}] provided."
      end
    end

    if (opts[:suppress_ssl_warnings] or
        %w(1 true yes).index(config['ARVADOS_API_HOST_INSECURE'].
                             andand.downcase))
      suppress_warnings do
        OpenSSL::SSL.const_set 'VERIFY_PEER', OpenSSL::SSL::VERIFY_NONE
      end
    end

    # Define a class and an Arvados instance method for each Arvados
    # resource. After this, self.job will return Arvados::Job;
    # self.job.new() and self.job.find() will do what you want.
    _arvados = self
    namespace_class = Arvados.const_set "A#{self.object_id}", Class.new
    self.arvados_api.schemas.each do |classname, schema|
      next if classname.match(/List$/)
      klass = Class.new(Arvados::Model) do
        def self.arvados
          @arvados
        end
        def self.api_models_sym
          @api_models_sym
        end
        def self.api_model_sym
          @api_model_sym
        end
      end

      # Define the resource methods (create, get, update, delete, ...)
      self.
        arvados_api.
        send(classname.underscore.split('/').last.pluralize.to_sym).
        discovered_methods.
        each do |method|
        class << klass; self; end.class_eval do
          define_method method.name do |*params|
            self.api_exec method, *params
          end
        end
      end

      # Give the new class access to the API
      klass.instance_eval do
        @arvados = _arvados
        # TODO: Pull these from the discovery document instead.
        @api_models_sym = classname.underscore.split('/').last.pluralize.to_sym
        @api_model_sym = classname.underscore.split('/').last.to_sym
      end

      # Create the new class in namespace_class so it doesn't
      # interfere with classes created by other Arvados objects. The
      # result looks like Arvados::A26949680::Job.
      namespace_class.const_set classname, klass

      self.define_singleton_method classname.underscore do
        klass
      end
    end
  end

  def client
    @client ||= ArvadosClient.
      new(:host => config["ARVADOS_API_HOST"],
          :application_name => @application_name,
          :application_version => @application_version.to_s)
  end

  def arvados_api
    @arvados_api ||= self.client.discovered_api('arvados', @arvados_api_version)
  end

  def self.debuglog(message, verbosity=1)
    $stderr.puts "#{File.split($0).last} #{$$}: #{message}" if @@debuglevel >= verbosity
  end

  def debuglog *args
    self.class.debuglog(*args)
  end

  def config(config_file_path="~/.config/arvados/settings.conf")
    return @config if @config

    # Initialize config settings with environment variables.
    config = {}
    config['ARVADOS_API_HOST']          = ENV['ARVADOS_API_HOST']
    config['ARVADOS_API_TOKEN']         = ENV['ARVADOS_API_TOKEN']
    config['ARVADOS_API_HOST_INSECURE'] = ENV['ARVADOS_API_HOST_INSECURE']

    if config['ARVADOS_API_HOST'] and config['ARVADOS_API_TOKEN']
      # Environment variables take precedence over the config file, so
      # there is no point reading the config file. If the environment
      # specifies a _HOST without asking for _INSECURE, we certainly
      # shouldn't give the config file a chance to create a
      # system-wide _INSECURE state for this user.
      #
      # Note: If we start using additional configuration settings from
      # this file in the future, we might have to read the file anyway
      # instead of returning here.
      return (@config = config)
    end

    begin
      expanded_path = File.expand_path config_file_path
      if File.exist? expanded_path
        # Load settings from the config file.
        lineno = 0
        File.open(expanded_path).each do |line|
          lineno = lineno + 1
          # skip comments and blank lines
          next if line.match('^\s*#') or not line.match('\S')
          var, val = line.chomp.split('=', 2)
          var.strip!
          val.strip!
          # allow environment settings to override config files.
          if !var.empty? and val
            config[var] ||= val
          else
            debuglog "#{expanded_path}: #{lineno}: could not parse `#{line}'", 0
          end
        end
      end
    rescue StandardError => e
      debuglog "Ignoring error reading #{config_file_path}: #{e}", 0
    end

    @config = config
  end

  def cluster_config
    return @cluster_config if @cluster_config

    uri = URI("https://#{config()["ARVADOS_API_HOST"]}/arvados/v1/config")
    cc = JSON.parse(Net::HTTP.get(uri))

    @cluster_config = cc
  end

  class Model
    def self.arvados_api
      arvados.arvados_api
    end
    def self.client
      arvados.client
    end
    def self.debuglog(*args)
      arvados.class.debuglog(*args)
    end
    def debuglog(*args)
      self.class.arvados.class.debuglog(*args)
    end
    def self.api_exec(method, parameters={})
      api_method = arvados_api.send(api_models_sym).send(method.name.to_sym)
      parameters.each do |k,v|
        parameters[k] = v.to_json if v.is_a? Array or v.is_a? Hash
      end
      # Look for objects expected by request.properties.(key).$ref and
      # move them from parameters (query string) to request body.
      body = nil
      method.discovery_document['request'].
        andand['properties'].
        andand.each do |k,v|
        if v.is_a? Hash and v['$ref']
          body ||= {}
          body[k] = parameters.delete k.to_sym
        end
      end
      result = client.
        execute(:api_method => api_method,
                :authenticated => false,
                :parameters => parameters,
                :body_object => body,
                :headers => {
                  :authorization => 'Bearer '+arvados.config['ARVADOS_API_TOKEN']
                })
      resp = JSON.parse result.body, :symbolize_names => true
      if resp[:errors]
        if !resp[:errors][0].match(/.*req-[0-9a-zA-Z]{20}.*/)
          resp[:errors][0] += " (#{result.headers['X-Request-Id'] or client.request_id})"
        end
        raise Arvados::TransactionFailedError.new(resp[:errors])
      elsif resp[:uuid] and resp[:etag]
        self.new(resp)
      elsif resp[:items].is_a? Array
        resp.merge(:items => resp[:items].collect do |i|
                     self.new(i)
                   end)
      else
        resp
      end
    end

    def []=(x,y)
      @attributes_to_update[x] = y
      @attributes[x] = y
    end
    def [](x)
      if @attributes[x].is_a? Hash or @attributes[x].is_a? Array
        # We won't be notified via []= if these change, so we'll just
        # assume they are going to get changed, and submit them if
        # save() is called.
        @attributes_to_update[x] = @attributes[x]
      end
      @attributes[x]
    end
    def save
      @attributes_to_update.keys.each do |k|
        @attributes_to_update[k] = @attributes[k]
      end
      j = self.class.api_exec :update, {
        :uuid => @attributes[:uuid],
        self.class.api_model_sym => @attributes_to_update.to_json
      }
      unless j.respond_to? :[] and j[:uuid]
        debuglog "Failed to save #{self.to_s}: #{j[:errors] rescue nil}", 0
        nil
      else
        @attributes_to_update = {}
        @attributes = j
      end
    end

    protected

    def initialize(j)
      @attributes_to_update = {}
      @attributes = j
    end
  end

  protected

  def suppress_warnings
    original_verbosity = $VERBOSE
    begin
      $VERBOSE = nil
      yield
    ensure
      $VERBOSE = original_verbosity
    end
  end
end
