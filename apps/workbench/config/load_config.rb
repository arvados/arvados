# This file must be loaded _after_ secret_token.rb if secret_token is
# defined there instead of in config/application.yml.

$application_config = {}

%w(application.default application).each do |cfgfile|
  path = "#{::Rails.root.to_s}/config/#{cfgfile}.yml"
  if File.exists? path
    yaml = ERB.new(IO.read path).result(binding)
    confs = YAML.load(yaml)
    $application_config.merge!(confs['common'] || {})
    $application_config.merge!(confs[::Rails.env.to_s] || {})
  end
end

ArvadosWorkbench::Application.configure do
  nils = []
  $application_config.each do |k, v|
    # "foo.bar: baz" --> { config.foo.bar = baz }
    cfg = config
    ks = k.split '.'
    k = ks.pop
    ks.each do |kk|
      cfg = cfg.send(kk)
    end
    if cfg.respond_to?(k.to_sym) and !cfg.send(k).nil?
      # Config must have been set already in environments/*.rb.
      #
      # After config files have been migrated, this mechanism should
      # be deprecated, then removed.
    elsif v.nil?
      # Config variables are not allowed to be nil. Make a "naughty"
      # list, and present it below.
      nils << k
    else
      cfg.send "#{k}=", v
    end
  end
  if !nils.empty?
    raise <<EOS
Refusing to start in #{::Rails.env.to_s} mode with missing configuration.

The following configuration settings must be specified in
config/application.yml:
* #{nils.join "\n* "}

EOS
  end
end
