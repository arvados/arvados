import arvados
import os
import robust_put
import stat

# Implements "Virtual Working Directory"
# Provides a way of emulating a shared writable directory in Keep based
# on a "check out, edit, check in, merge" model.
# At the moment, this only permits adding new files, applications
# cannot modify or delete existing files.

# Create a symlink tree rooted at target_dir mirroring arv-mounted
# source_collection.  target_dir must be empty, and will be created if it
# doesn't exist.
def checkout(source_collection, target_dir, keepmount=None):
    # create symlinks
    if keepmount is None:
        keepmount = os.environ['TASK_KEEPMOUNT']

    if not os.path.exists(target_dir):
        os.makedirs(target_dir)

    l = os.listdir(target_dir)
    if len(l) > 0:
        raise Exception("target_dir must be empty before checkout, contains %s" % l)

    stem = os.path.join(keepmount, source_collection)
    for root, dirs, files in os.walk(os.path.join(keepmount, source_collection), topdown=True):
        rel = root[len(stem)+1:]
        for d in dirs:
            os.mkdir(os.path.join(target_dir, rel, d))
        for f in files:
            os.symlink(os.path.join(root, f), os.path.join(target_dir, rel, f))

# Delete all symlinks and check in any remaining normal files.
# If merge == True, merge the manifest with source_collection and return a
# CollectionReader for the combined collection.
def checkin(source_collection, target_dir, merge=True):
    # delete symlinks, commit directory, merge manifests and return combined
    # collection.
    for root, dirs, files in os.walk(target_dir):
        for f in files:
            s = os.lstat(os.path.join(root, f))
            if stat.S_ISLNK(s.st_mode):
                os.unlink(os.path.join(root, f))

    uuid = robust_put.upload(target_dir)
    if merge:
        cr1 = arvados.CollectionReader(source_collection)
        cr2 = arvados.CollectionReader(uuid)
        combined = arvados.CollectionReader(cr1.manifest_text() + cr2.manifest_text())
        return combined
    else:
        return arvados.CollectionReader(uuid)
