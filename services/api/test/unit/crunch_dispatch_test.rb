require 'test_helper'
require 'crunch_dispatch'
require 'helpers/git_test_helper'

class CrunchDispatchTest < ActiveSupport::TestCase
  include GitTestHelper

  test 'choose cheaper nodes first' do
    act_as_system_user do
      # Replace test fixtures with a set suitable for testing dispatch
      Node.destroy_all

      # Idle nodes with different prices
      [['compute1', 3.20, 32],
       ['compute2', 1.60, 16],
       ['compute3', 0.80, 8]].each do |hostname, price, cores|
        Node.create!(hostname: hostname,
                     info: {
                       'slurm_state' => 'idle',
                     },
                     properties: {
                       'cloud_node' => {
                         'price' => price,
                       },
                       'total_cpu_cores' => cores,
                       'total_ram_mb' => cores*1024,
                       'total_scratch_mb' => cores*10000,
                     })
      end

      # Node with no price information
      Node.create!(hostname: 'compute4',
                   info: {
                     'slurm_state' => 'idle',
                   },
                   properties: {
                     'total_cpu_cores' => 8,
                     'total_ram_mb' => 8192,
                     'total_scratch_mb' => 80000,
                   })

      # Cheap but busy node
      Node.create!(hostname: 'compute5',
                   info: {
                     'slurm_state' => 'alloc',
                   },
                   properties: {
                     'cloud_node' => {
                       'price' => 0.10,
                     },
                     'total_cpu_cores' => 32,
                     'total_ram_mb' => 32768,
                     'total_scratch_mb' => 320000,
                   })
    end

    dispatch = CrunchDispatch.new
    [[1, 16384, ['compute2']],
     [2, 16384, ['compute2', 'compute1']],
     [2, 8000, ['compute4', 'compute3']],
    ].each do |min_nodes, min_ram, expect_nodes|
      job = Job.new(runtime_constraints: {
                      'min_nodes' => min_nodes,
                      'min_ram_mb_per_node' => min_ram,
                    })
      nodes = dispatch.nodes_available_for_job_now job
      assert_equal expect_nodes, nodes
    end
  end

  test 'respond to TERM' do
    lockfile = Rails.root.join 'tmp', 'dispatch.lock'
    ENV['CRUNCH_DISPATCH_LOCKFILE'] = lockfile.to_s
    begin
      pid = Process.fork do
        begin
          # Abandon database connections inherited from parent
          # process.  Credit to
          # https://github.com/kstephens/rails_is_forked
          ActiveRecord::Base.connection_handler.connection_pools.each_value do |pool|
            pool.instance_eval do
              @reserved_connections = {}
              @connections = []
            end
          end
          ActiveRecord::Base.establish_connection

          dispatch = CrunchDispatch.new
          dispatch.stubs(:did_recently).returns true
          dispatch.run []
        ensure
          Process.exit!
        end
      end
      assert_with_timeout 5, "Dispatch did not lock #{lockfile}" do
        !can_lock(lockfile)
      end
    ensure
      Process.kill("TERM", pid)
    end
    assert_with_timeout 20, "Dispatch did not unlock #{lockfile}" do
      can_lock(lockfile)
    end
  end

  test 'override --cgroup-root with CRUNCH_CGROUP_ROOT' do
    ENV['CRUNCH_CGROUP_ROOT'] = '/path/to/cgroup'
    Rails.configuration.crunch_job_wrapper = :none
    act_as_system_user do
      j = Job.create(repository: 'active/foo',
                     script: 'hash',
                     script_version: '4fe459abe02d9b365932b8f5dc419439ab4e2577',
                     script_parameters: {})
      ok = false
      Open3.expects(:popen3).at_least_once.with do |*args|
        if args.index(j.uuid)
          ok = ((i = args.index '--cgroup-root') and
                (args[i+1] == '/path/to/cgroup'))
        end
        true
      end.raises(StandardError.new('all is well'))
      dispatch = CrunchDispatch.new
      dispatch.parse_argv ['--jobs']
      dispatch.refresh_todo
      dispatch.start_jobs
      assert ok
    end
  end

  def assert_with_timeout timeout, message
    t = 0
    while (t += 0.1) < timeout
      if yield
        return
      end
      sleep 0.1
    end
    assert false, message + " (waited #{timeout} seconds)"
  end

  def can_lock lockfile
    lockfile.open(File::RDWR|File::CREAT, 0644) do |f|
      return f.flock(File::LOCK_EX|File::LOCK_NB)
    end
  end
end
