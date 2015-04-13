# See http://aaronvb.com/articles/37-rails-caching-and-undefined-class-module

# Config must be done before we load model class files; otherwise they
# won't be able to use Rails.configuration.* to initialize their
# classes.
require_relative 'load_config.rb'

if Rails.env == 'development'
  Dir.foreach("#{Rails.root}/app/models") do |model_file|
    require_dependency model_file if model_file.match /\.rb$/
  end
end
