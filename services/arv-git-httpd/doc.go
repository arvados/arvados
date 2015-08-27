/*
arv-git-httpd provides authenticated access to Arvados-hosted git repositories.

See http://doc.arvados.org/install/install-arv-git-httpd.html.

Example:

	arv-git-httpd -address=:8000 -repo-root=/var/lib/arvados/git

Options:

	-address [host]:[port]

Listen at the given host and port.

Host can be a domain name, an IP address, or empty (listen on all
addresses).

Port can be a name, a port number, or 0 (choose an available port).

	-repo-root path

Directory containing git repositories. When a client requests either
"foo/bar.git" or "foo/bar/.git", git-http-backend will be invoked on
"path/foo/bar.git" or (if that doesn't exist) "path/foo/bar/.git".

	-git-command path

Location of the CGI program to execute for each authorized request
(normally this is gitolite-shell if repositories are controlled by
gitolite, otherwise git). It is invoked with a single argument,
'http-backend'.  Default is /usr/bin/git.

*/
package main
