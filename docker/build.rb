#! /usr/bin/env ruby

require 'tempfile'
require 'yaml'

def sudo(*cmd)
  # user can pass a single list in as an argument
  # to allow usage like: sudo %w(apt-get install foo)
  if cmd.length == 1 and cmd[0].class == Array
    cmd = cmd[0]
  end
  system '/usr/bin/sudo', *cmd
end

def is_valid_email? str
  str.match /^\S+@\S+\.\S+$/
end

def generate_api_hostname
  rand(2**256).to_s(36)[0...5]
end

# ip_forwarding_enabled?
#   Returns 'true' if IP forwarding is enabled in the kernel
#
def ip_forwarding_enabled?
  %x(/sbin/sysctl --values net.ipv4.ip_forward) == "1\n"
end

def find_ssh_key key_name
  # If the user already has a key loaded in their agent, use one of those
  agent_keys = `ssh-add -l`
  if agent_keys.empty?
    # Use a key named arvados_{key_name}_id_rsa, generating
    # a passphraseless key if necessary.
    ssh_key_file = "#{ENV['HOME']}/.ssh/arvados_#{key_name}_id_rsa"
    unless File.exists? ssh_key_file
      system 'ssh_keygen', '-f', ssh_key_file, '-P', ''
    end
  else
    # choose an agent key at random
    ssh_key_file = agent_keys.split("\n").first.split[2]
  end

  return File.exists?("#{ssh_key_file}.pub") ? "#{ssh_key_file}.pub" : nil
end

if not ip_forwarding_enabled?
  warn "NOTE: IP forwarding must be enabled in the kernel."
  warn "Turning IP forwarding on. You may be asked to enter your password."
  sudo %w(/sbin/sysctl net.ipv4.ip_forward=1)
end

# Check that:
#   * Docker is installed and can be found in the user's path
#   * Docker can be run as a non-root user
#      - TODO: put the user is in the docker group if necessary
#      - TODO: mount cgroup automatically
#      - TODO: start the docker service if not started

docker_path = %x(which docker).chomp
if docker_path.empty?
  warn "Docker not found."
  warn ""
  warn "Please make sure that Docker has been installed and"
  warn "can be found in your PATH."
  warn ""
  warn "Installation instructions for a variety of platforms can be found at"
  warn "http://docs.docker.io/en/latest/installation/"
  exit
elsif not system 'docker images > /dev/null 2>&1'
  warn "WARNING: docker could not be run."
  warn "Please make sure that:"
  warn "  * You have permission to read and write /var/run/docker.sock"
  warn "  * a 'cgroup' volume is mounted on your machine"
  warn "  * the docker daemon is running"
end

# Generate a config.yml if it does not exist
if not File.exists? 'config.yml'
  print "Generating config.yml.\n"
  print "Arvados needs to know the email address of the administrative user,\n"
  print "so that when that user logs in they are automatically made an admin.\n"
  print "This should be the email address you use to log in to Google.\n"
  print "\n"
  admin_email_address = ""
  until is_valid_email? admin_email_address
    print "Enter your Google ID email address here: "
    admin_email_address = gets.strip
    if not is_valid_email? admin_email_address
      print "That doesn't look like a valid email address. Please try again.\n"
    end
  end

  File.open 'config.yml', 'w' do |config_out|
    config = YAML.load_file 'config.yml.example'
    config['API_AUTO_ADMIN_USER'] = admin_email_address
    config['API_HOSTNAME'] = generate_api_hostname
    config['PUBLIC_KEY_PATH'] = find_ssh_key(config['API_HOSTNAME'])
    config.each_key do |var|
      if var.end_with?('_PW') or var.end_with?('_SECRET')
        config[var] = rand(2**256).to_s(36)
      end
      config_out.write "#{var}: #{config[var]}\n"
    end
  end
end

# If all prerequisites are met, go ahead and build.
if ip_forwarding_enabled? and
    not docker_path.empty? and
    File.exists? 'config.yml'
  warn "Building Arvados."
  system '/usr/bin/make', *ARGV
end

# install_docker
#   Determine which Docker package is suitable for this Linux distro
#   and install, resolving any dependencies.
#   NOTE: not in use yet.

def install_docker
  linux_distro = %x(lsb_release --id).split.last
  linux_release = %x(lsb_release --release).split.last
  linux_version = linux_distro + " " + linux_release
  kernel_release = `uname -r`
  
  case linux_distro
  when 'Ubuntu'
    if not linux_release.match '^1[234]\.'
      warn "Arvados requires at least Ubuntu 12.04 (Precise Pangolin)."
      warn "Your system is Ubuntu #{linux_release}."
      exit
    end
    if linux_release.match '^12' and kernel_release.start_with? '3.2'
      # Ubuntu Precise ships with a 3.2 kernel and must be upgraded.
      warn "Your kernel #{kernel_release} must be upgraded to run Docker."
      warn "To do this:"
      warn "  sudo apt-get update"
      warn "  sudo apt-get install linux-image-generic-lts-raring linux-headers-generic-lts-raring"
      warn "  sudo reboot"
      exit
    else
      # install AUFS
      sudo 'apt-get', 'update'
      sudo 'apt-get', 'install', "linux-image-extra-#{kernel_release}"
    end

    # add Docker repository
    sudo %w(/usr/bin/apt-key adv
              --keyserver keyserver.ubuntu.com
              --recv-keys 36A1D7869245C8950F966E92D8576A8BA88D21E9)
    source_file = Tempfile.new('arv')
    source_file.write("deb http://get.docker.io/ubuntu docker main\n")
    source_file.close
    sudo '/bin/mv', source_file.path, '/etc/apt/sources.list.d/docker.list'
    sudo %w(/usr/bin/apt-get update)
    sudo %w(/usr/bin/apt-get install lxc-docker)

    # Set up for non-root access
    sudo %w(/usr/sbin/groupadd docker)
    sudo '/usr/bin/gpasswd', '-a', ENV['USER'], 'docker'
    sudo %w(/usr/sbin/service docker restart)
  when 'Debian'
  else
    warn "Must be running a Debian or Ubuntu release in order to run Docker."
    exit
  end
end

