import shutil
import os
import sys
import re

if os.path.exists("PKG-INFO"):
    with open("PKG-INFO", "r") as p:
        for l in p:
            g = re.match(r'Version: (.*)', l)
            if g != None:
                minor_version = g.group(1)
else:
    with os.popen("git log --format=format:%ct.%h -n1 .") as m:
        minor_version = m.read()

# setup.py and setup_fuse.py both share the build/ directory (argh!) so
# make sure to delete it to avoid scooping up the wrong files.
build_dir = os.path.join(os.path.dirname(sys.argv[0]), 'build')
shutil.rmtree(build_dir, ignore_errors=True)
