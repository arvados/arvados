from builtins import object
import json
import os

class TaskOutputDir(object):
    """Keep-backed directory for staging outputs of Crunch tasks.

    Example, in a crunch task whose output is a file called "out.txt"
    containing "42":

        import arvados
        import arvados.crunch
        import os

        out = arvados.crunch.TaskOutputDir()
        with open(os.path.join(out.path, 'out.txt'), 'w') as f:
            f.write('42')
        arvados.current_task().set_output(out.manifest_text())
    """
    def __init__(self):
        self.path = os.environ['TASK_KEEPMOUNT_TMP']

    def __str__(self):
        return self.path

    def manifest_text(self):
        snapshot = os.path.join(self.path, '.arvados#collection')
        return json.load(open(snapshot))['manifest_text']
