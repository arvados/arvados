#! /usr/bin/env ruby

require 'optparse'
require 'tempfile'
require 'yaml'

def main options
  if not ip_forwarding_enabled?
    warn "NOTE: IP forwarding must be enabled in the kernel."
    warn "Turning IP forwarding on now."
    sudo %w(/sbin/sysctl net.ipv4.ip_forward=1)
  end

  # Check that:
  #   * Docker is installed and can be found in the user's path
  #   * Docker can be run as a non-root user
  #      - TODO: put the user in the docker group if necessary
  #      - TODO: mount cgroup automatically
  #      - TODO: start the docker service if not started

  docker_path = %x(which docker.io).chomp

  if docker_path.empty?
    docker_path = %x(which docker).chomp
  end

  if docker_path.empty?
    warn "Docker not found."
    warn ""
    warn "Please make sure that Docker has been installed and"
    warn "can be found in your PATH."
    warn ""
    warn "Installation instructions for a variety of platforms can be found at"
    warn "http://docs.docker.io/en/latest/installation/"
    exit 1
  elsif not docker_ok? docker_path
    warn "WARNING: docker could not be run."
    warn "Please make sure that:"
    warn "  * You have permission to read and write /var/run/docker.sock"
    warn "  * a 'cgroup' volume is mounted on your machine"
    warn "  * the docker daemon is running"
    exit 2
  end

  # Check that debootstrap is installed.
  if not debootstrap_ok?
    warn "Installing debootstrap."
    sudo '/usr/bin/apt-get', 'install', 'debootstrap'
  end

  # Generate a config.yml if it does not exist or is empty
  if not File.size? 'config.yml'
    print "Generating config.yml.\n"
    print "Arvados needs to know the email address of the administrative user,\n"
    print "so that when that user logs in they are automatically made an admin.\n"
    print "This should be an email address associated with a Google account.\n"
    print "\n"
    admin_email_address = ""
    until is_valid_email? admin_email_address
      print "Enter your Google ID email address here: "
      admin_email_address = gets.strip
      if not is_valid_email? admin_email_address
        print "That doesn't look like a valid email address. Please try again.\n"
      end
    end

    print "Arvados needs to know the shell login name for the administrative user.\n"
    print "This will also be used as the name for your git repository.\n"
    print "\n"
    user_name = ""
    until is_valid_user_name? user_name
      print "Enter a shell login name here: "
      user_name = gets.strip
      if not is_valid_user_name? user_name
        print "That doesn't look like a valid shell login name. Please try again.\n"
      end
    end

    File.open 'config.yml', 'w' do |config_out|
      config_out.write "# If a _PW or _SECRET variable is set to an empty string, a password\n"
      config_out.write "# will be chosen randomly at build time. This is the\n"
      config_out.write "# recommended setting.\n\n"
      config = YAML.load_file 'config.yml.example'
      config['API_AUTO_ADMIN_USER'] = admin_email_address
      config['ARVADOS_USER_NAME'] = user_name
      config['API_HOSTNAME'] = generate_api_hostname
      config['API_WORKBENCH_ADDRESS'] = 'http://localhost:9899'
      config['PUBLIC_KEY_PATH'] = find_or_create_ssh_key(config['API_HOSTNAME'])
      config.each_key do |var|
        config_out.write "#{var}: #{config[var]}\n"
      end
    end
  end

  # If all prerequisites are met, go ahead and build.
  if ip_forwarding_enabled? and
      docker_ok? docker_path and
      debootstrap_ok? and
      File.exists? 'config.yml'
    exit 0
  else
    exit 6
  end
end

# sudo
#   Execute the arg list 'cmd' under sudo.
#   cmd can be passed either as a series of arguments or as a
#   single argument consisting of a list, e.g.:
#     sudo 'apt-get', 'update'
#     sudo(['/usr/bin/gpasswd', '-a', ENV['USER'], 'docker'])
#     sudo %w(/usr/bin/apt-get install lxc-docker)
#
def sudo(*cmd)
  # user can pass a single list in as an argument
  # to allow usage like: sudo %w(apt-get install foo)
  warn "You may need to enter your password here."
  if cmd.length == 1 and cmd[0].class == Array
    cmd = cmd[0]
  end
  system '/usr/bin/sudo', *cmd
end

# is_valid_email?
#   Returns true if its arg looks like a valid email address.
#   This is a very very loose sanity check.
#
def is_valid_email? str
  str.match /^\S+@\S+\.\S+$/
end

# is_valid_user_name?
#   Returns true if its arg looks like a valid unix username.
#   This is a very very loose sanity check.
#
def is_valid_user_name? str
  # borrowed from Debian's adduser (version 3.110)
  str.match /^[_.A-Za-z0-9][-\@_.A-Za-z0-9]*\$?$/
end

# generate_api_hostname
#   Generates a 5-character randomly chosen API hostname.
#
def generate_api_hostname
  rand(2**256).to_s(36)[0...5]
end

# ip_forwarding_enabled?
#   Returns 'true' if IP forwarding is enabled in the kernel
#
def ip_forwarding_enabled?
  %x(/sbin/sysctl -n net.ipv4.ip_forward) == "1\n"
end

# debootstrap_ok?
#   Returns 'true' if debootstrap is installed and working.
#
def debootstrap_ok?
  return system '/usr/sbin/debootstrap --version > /dev/null 2>&1'
end

# docker_ok?
#   Returns 'true' if docker can be run as the current user.
#
def docker_ok?(docker_path)
  return system "#{docker_path} images > /dev/null 2>&1"
end

# find_or_create_ssh_key arvados_name
#   Returns the SSH public key appropriate for this Arvados instance,
#   generating one if necessary.
#
def find_or_create_ssh_key arvados_name
  ssh_key_file = "#{ENV['HOME']}/.ssh/arvados_#{arvados_name}_id_rsa"
  unless File.exists? ssh_key_file
    system 'ssh-keygen',
           '-f', ssh_key_file,
           '-C', "arvados@#{arvados_name}",
           '-P', ''
  end

  return "#{ssh_key_file}.pub"
end

# install_docker
#   Determines which Docker package is suitable for this Linux distro
#   and installs it, resolving any dependencies.
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
      exit 3
    end
    if linux_release.match '^12' and kernel_release.start_with? '3.2'
      # Ubuntu Precise ships with a 3.2 kernel and must be upgraded.
      warn "Your kernel #{kernel_release} must be upgraded to run Docker."
      warn "To do this:"
      warn "  sudo apt-get update"
      warn "  sudo apt-get install linux-image-generic-lts-raring linux-headers-generic-lts-raring"
      warn "  sudo reboot"
      exit 4
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
    exit 5
  end
end


if __FILE__ == $PROGRAM_NAME
  options = { :makefile => File.join(File.dirname(__FILE__), 'Makefile') }
  OptionParser.new do |opts|
    opts.on('-m', '--makefile MAKEFILE-PATH',
            'Path to the Makefile used to build Arvados Docker images') do |mk|
      options[:makefile] = mk
    end
  end
  main options
end
