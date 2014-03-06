#! /usr/bin/env ruby

require 'tempfile'

docker_path = ""

def sudo(*cmd)
  # user can pass a single list in as an argument
  # to allow usage like: sudo %w(apt-get install foo)
  if cmd.length = 1 and cmd[0].class == Array
    cmd = cmd[0]
  end
  system '/usr/bin/sudo', *cmd
end

# Check that:
#   * LXC is installed.
def lxc_installed?
  lxc_path = %x(which lxc)
  not lxc_path.empty?
end

if not lxc_installed?
  warn "Installing LXC (you may need to enter your password)."
  sudo %w(apt-get install lxc)
end

# Check that:
#   * IP forwarding is enabled in the kernel.

def ip_forwarding_enabled?
  %x(/sbin/sysctl --values net.ipv4.ip_forward) == "1\n"
end

if not ip_forwarding_enabled?
  warn "NOTE: IP forwarding must be enabled in the kernel."
  warn "Turning IP forwarding on. You may be asked to enter your password."
  sudo %w(/sbin/sysctl net.ipv4.ip_forward=1)
fi

# Check that:
#   * Docker is installed
#   * Docker can be found in the user's path
#   * The user is in the docker group
#   * cgroup is mounted
#   * the docker daemon is running

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
  when 'Debian 7.4'
  end
end

docker_path = %x(which docker).chomp
if docker_path.empty?
  warn "Docker not found."
  
