package main

// tasks to run on a controller node
var ctlTasks = []task{
	&download{
		URL:  "https://releases.hashicorp.com/consul/0.7.2/consul_0.7.2_linux_amd64.zip",
		Dest: "/usr/local/bin/consul",
		Size: 29079005,
		Mode: 0755,
	},
}
