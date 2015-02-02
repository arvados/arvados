require 'bundler'

$ARV_API_SERVER_DIR = File.expand_path('../..', __FILE__)
SERVER_PID_PATH = 'tmp/pids/passenger.3002.pid'

class WebsocketTestRunner < MiniTest::Unit
  def _system(*cmd)
    Bundler.with_clean_env do
      if not system({'ARVADOS_WEBSOCKETS' => 'ws-only', 'RAILS_ENV' => 'test'}, *cmd)
        raise RuntimeError, "Command failed with exit status #{$?}: #{cmd.inspect}"
      end
    end
  end

  def _run(args=[])
    server_pid = Dir.chdir($ARV_API_SERVER_DIR) do |apidir|
      # Only passenger seems to be able to run the websockets server successfully.
      _system('passenger', 'start', '-d', '-p3002')
      timeout = Time.now.tv_sec + 10
      begin
        sleep 0.2
        begin
          server_pid = IO.read(SERVER_PID_PATH).to_i
          good_pid = (server_pid > 0) and (Process.kill(0, pid) rescue false)
        rescue Errno::ENOENT
          good_pid = false
        end
      end while (not good_pid) and (Time.now.tv_sec < timeout)
      if not good_pid
        raise RuntimeError, "could not find API server Rails pid"
      end
      server_pid
    end
    begin
      super(args)
    ensure
      Dir.chdir($ARV_API_SERVER_DIR) do
        _system('passenger', 'stop', '-p3002')
      end
      # DatabaseCleaner leaves the database empty. Prefer to leave it full.
      dc = DatabaseController.new
      dc.define_singleton_method :render do |*args| end
      dc.reset
    end
  end
end

MiniTest::Unit.runner = WebsocketTestRunner.new
