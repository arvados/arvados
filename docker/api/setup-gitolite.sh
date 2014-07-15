#!/bin/bash

ssh-keygen -q -N '' -t rsa -f /root/.ssh/id_rsa

useradd git
mkdir /home/git

# Set up gitolite repository
cp ~root/.ssh/id_rsa.pub ~git/root-authorized_keys.pub
chown git:git /home/git -R
su - git -c "mkdir -p ~/bin"

su - git -c "git clone git://github.com/sitaramc/gitolite"
su - git -c "gitolite/install -ln ~/bin"
su - git -c "PATH=/home/git/bin:$PATH gitolite setup -pk ~git/root-authorized_keys.pub"

# Now set up the gitolite repo(s) we use
mkdir -p /usr/local/arvados/gitolite-tmp/
# Make ssh store the host key
ssh -o "StrictHostKeyChecking no" git@api info
# Now check out the tree
git clone git@api:gitolite-admin.git /usr/local/arvados/gitolite-tmp/gitolite-admin/
cd /usr/local/arvados/gitolite-tmp/gitolite-admin
mkdir keydir/arvados
mkdir conf/admin
mkdir conf/auto
echo " 

@arvados_git_user = arvados_git_user

repo @all
     RW+                 = @arvados_git_user

" > conf/admin/arvados.conf
echo '
include "auto/*.conf"
include "admin/*.conf"
' >> conf/gitolite.conf

#su - git -c "ssh-keygen -t rsa"
cp /root/.ssh/id_rsa.pub keydir/arvados/arvados_git_user.pub
# Replace the 'root' key with the user key, just in case
cp /root/.ssh/authorized_keys keydir/root-authorized_keys.pub
# But also make sure we have the root key installed so it can access all keys
git add keydir/root-authorized_keys.pub
git add keydir/arvados/arvados_git_user.pub
git add conf/admin/arvados.conf
git add keydir/arvados/
git add conf/gitolite.conf
git commit -a -m 'git server setup'
git push

echo "ARVADOS_API_HOST_INSECURE=yes" > /etc/cron.d/gitolite-update
echo "*/5 * * * * root /bin/bash -c 'source /etc/profile.d/rvm.sh && /usr/local/arvados/update-gitolite.rb production'" >> /etc/cron.d/gitolite-update

