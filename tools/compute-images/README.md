# Arvados Compute Node Image Builder

This directory includes templates to build custom cloud images for Arvados compute nodes. For instructions, refer to the [Arvados cloud compute node image documentation](https://doc.arvados.org/install/crunch2-cloud/install-compute-node.html).

## Development

If you are developing the Ansible playbook, note that you can test it by [running the Ansible playbook independently](https:///doc.arvados.org/install/crunch2-cloud/install-compute-node.html#ansible-build) of Packer.

### Managed Node Requirements

For testing, you'll need a Debian or Ubuntu system where you don't mind messing with the system configuration. It can be a virtual machine. You must set up the following before you run Ansible (this is stuff that's typically preconfigured in the cloud):

* Install `openssh-server`, `python3`, and `sudo`
* Set up a user account for yourself that is allowed to SSH in and use `sudo`

### Configuration Requirements

You must have an Arvados cluster configuration. You can start by copying the defaults from the Arvados source in `arvados/lib/config/config.default.yml`. After you make your copy, you should change the example identifier `xxxxx` under `Clusters` to a unique five-alphanumeric identifier for your test cluster. It SHOULD start with `z` so it's easily identifiable as a test cluster. You may also change other settings that you specifically want to test such as `Containers.RuntimeEngine`.

Once you have this, you can start [following the Ansible build instructions](https:///doc.arvados.org/install/crunch2-cloud/install-compute-node.html#ansible-build). When you write `host_config.yml`, set `arvados_config_file` to the ABSOLUTE path of the cluster configuration file you wrote, and `arvados_cluster_id` to the cluster identifier you wrote in there under `Clusters`.

You must define at least one public SSH key for the Crunch user. The easiest way to do this is to set `compute_authorized_keys` in your `host_config.yml` and point it to one of your SSH public keys or `authorized_keys` file. If you set `Containers.DispatchPrivateKey` in your Arvados cluster configuration file, that's sufficient too.
