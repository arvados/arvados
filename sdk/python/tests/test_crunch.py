import arvados.crunch
import os
import shutil
import tempfile
import unittest

class TaskOutputDirTest(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.mkdtemp()
        os.environ['TASK_KEEPMOUNT_TMP'] = self.tmp

    def tearDown(self):
        os.environ.pop('TASK_KEEPMOUNT_TMP')
        shutil.rmtree(self.tmp)

    def test_env_var(self):
        out = arvados.crunch.TaskOutputDir()
        self.assertEqual(out.path, self.tmp)

        with open(os.path.join(self.tmp, '.arvados#collection'), 'w') as f:
            f.write('{\n  "manifest_text":"",\n  "uuid":null\n}\n')
        self.assertEqual(out.manifest_text(), '')

        # Special file must be re-read on each call to manifest_text().
        with open(os.path.join(self.tmp, '.arvados#collection'), 'w') as f:
            f.write(r'{"manifest_text":". unparsed 0:3:foo\n","uuid":null}')
        self.assertEqual(out.manifest_text(), ". unparsed 0:3:foo\n")
