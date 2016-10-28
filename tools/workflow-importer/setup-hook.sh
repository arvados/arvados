#!/bin/sh
set -e
GIT_DIR=$1

if test -d "$GIT_DIR/.git" ; then
    GIT_DIR="$GIT_DIR/.git"
fi

virtualenv $GIT_DIR/hooks/workflowimport
. $GIT_DIR/hooks/workflowimport/bin/activate
pip install -U setuptools
python setup.py install

cat >$GIT_DIR/hooks/post-commit <<EOF
#!/bin/sh
if test -z "\$ARVADOS_API_HOST" ; then
  ARVADOS_API_HOST="$ARVADOS_API_HOST"
fi
if test -z "\$ARVADOS_API_HOST_INSECURE" ; then
  ARVADOS_API_HOST_INSECURE="$ARVADOS_API_HOST_INSECURE"
fi
if test -z "\$ARVADOS_API_TOKEN" ; then
  ARVADOS_API_TOKEN="$ARVADOS_API_TOKEN"
fi
export ARVADOS_API_HOST ARVADOS_API_TOKEN ARVADOS_API_HOST_INSECURE
unset GIT_DIR
EOF

cp $GIT_DIR/hooks/post-commit $GIT_DIR/hooks/post-update
chmod +x $GIT_DIR/hooks/post-commit
chmod +x $GIT_DIR/hooks/post-update

cat >>$GIT_DIR/hooks/post-commit <<EOF
exec $GIT_DIR/hooks/workflowimport/bin/workflowimporter.py \$PWD \$(git branch | grep "^*" | cut -c3-)
EOF

cat >>$GIT_DIR/hooks/post-update <<EOF
exec $GIT_DIR/hooks/workflowimport/bin/workflowimporter.py \$PWD \$1
EOF
