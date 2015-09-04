import arvados
import os
import stat
import arvados.commands.run
import logging

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

def checkin(target_dir):
    """Write files in `target_dir` to Keep.

    Regular files or symlinks to files outside the keep mount are written to
    Keep as normal files (Keep does not support symlinks).

    Symlinks to files in the keep mount will result in files in the new
    collection which reference existing Keep blocks, no data copying necessary.

    Returns a new Collection object, with data flushed but the collection record
    not saved to the API.

    """

    outputcollection = arvados.collection.Collection(num_retries=5)

    if target_dir[-1:] != '/':
        target_dir += '/'

    collections = {}

    logger = logging.getLogger("arvados")

    last_error = None
    for root, dirs, files in os.walk(target_dir):
        for f in files:
            try:
                s = os.lstat(os.path.join(root, f))

                writeIt = False

                if stat.S_ISREG(s.st_mode):
                    writeIt = True
                elif stat.S_ISLNK(s.st_mode):
                    # 1. check if it is a link into a collection
                    real = os.path.split(os.path.realpath(os.path.join(root, f)))
                    (pdh, branch) = arvados.commands.run.is_in_collection(real[0], real[1])
                    if pdh is not None:
                        # 2. load collection
                        if pdh not in collections:
                            # 2.1 make sure it is flushed (see #5787 note 11)
                            fd = os.open(real[0], os.O_RDONLY)
                            os.fsync(fd)
                            os.close(fd)

                            # 2.2 get collection from API server
                            collections[pdh] = arvados.collection.CollectionReader(pdh,
                                                                                   api_client=outputcollection._my_api(),
                                                                                   keep_client=outputcollection._my_keep(),
                                                                                   num_retries=5)
                        # 3. copy arvfile to new collection
                        outputcollection.copy(branch, os.path.join(root[len(target_dir):], f), source_collection=collections[pdh])
                    else:
                        writeIt = True

                if writeIt:
                    reldir = root[len(target_dir):]
                    with outputcollection.open(os.path.join(reldir, f), "wb") as writer:
                        with open(os.path.join(root, f), "rb") as reader:
                            dat = reader.read(64*1024)
                            while dat:
                                writer.write(dat)
                                dat = reader.read(64*1024)
            except (IOError, OSError) as e:
                logger.error(e)
                last_error = e

    return (outputcollection, last_error)
