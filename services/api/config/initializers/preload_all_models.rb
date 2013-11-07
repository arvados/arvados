# See http://aaronvb.com/articles/37-rails-caching-and-undefined-class-module

if Rails.env == 'development'
  Dir.foreach("#{Rails.root}/app/models") do |model_file|
    require_dependency model_file if model_file.match /\.rb$/
  end 
end
