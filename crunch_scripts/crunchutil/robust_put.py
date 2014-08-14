import arvados
import arvados.commands.put as put
import os
import logging

def machine_progress(bytes_written, bytes_expected):
    return "upload wrote {} total {}\n".format(
        bytes_written, -1 if (bytes_expected is None) else bytes_expected)

class Args(object):
    def __init__(self, fn):
        self.filename = None
        self.paths = [fn]
        self.max_manifest_depth = 0

# Upload to Keep with error recovery.
# Return a uuid or raise an exception if there are too many failures.
def upload(source_dir):
    source_dir = os.path.abspath(source_dir)
    done = False
    if 'TASK_WORK' in os.environ:
        resume_cache = put.ResumeCache(os.path.join(arvados.current_task().tmpdir, "upload-output-checkpoint"))
    else:
        resume_cache = put.ResumeCache(put.ResumeCache.make_path(Args(source_dir)))
    reporter = put.progress_writer(machine_progress)
    bytes_expected = put.expected_bytes_for([source_dir])
    backoff = 1
    outuuid = None
    while not done:
        try:
            out = put.ArvPutCollectionWriter.from_cache(resume_cache, reporter, bytes_expected)
            out.do_queued_work()
            out.write_directory_tree(source_dir, max_manifest_depth=0)
            outuuid = out.finish()
            done = True
        except KeyboardInterrupt as e:
            logging.critical("caught interrupt signal 2")
            raise e
        except Exception as e:
            logging.exception("caught exception:")
            backoff *= 2
            if backoff > 256:
                logging.critical("Too many upload failures, giving up")
                raise e
            else:
                logging.warning("Sleeping for %s seconds before trying again" % backoff)
                time.sleep(backoff)
    return outuuid
