namespace :test do
  new_task = Rake::TestTask.new(tasks: "test:prepare") do |t|
    t.libs << "test"
    t.pattern = "test/tasks/**/*_test.rb"
  end
end
