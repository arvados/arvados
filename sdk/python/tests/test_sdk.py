import mock
import os
import unittest

import arvados
import arvados.collection

class TestSDK(unittest.TestCase):

    @mock.patch('arvados.apisetup.api_from_config')
    @mock.patch('arvados.current_task')
    @mock.patch('arvados.current_job')
    def test_one_task_per_input_file_normalize(self, mock_job, mock_task, mock_api):
        # This manifest will be reduced from three lines to one when it is
        # normalized.
        nonnormalized_manifest = """. 5348b82a029fd9e971a811ce1f71360b+43 0:43:md5sum.txt
. 085c37f02916da1cad16f93c54d899b7+41 0:41:md5sum.txt
. 8b22da26f9f433dea0a10e5ec66d73ba+43 0:43:md5sum.txt
"""
        dummy_hash = 'ffffffffffffffffffffffffffffffff+0'

        mock_job.return_value = {
            'uuid': 'none',
            'script_parameters': {
                'input': dummy_hash
            }
        }
        mock_task.return_value = {
            'uuid': 'none',
            'sequence': 0,
        }
        # mock the API client to return a collection with a nonnormalized manifest.
        mock_api('v1').collections().get().execute.return_value = {
            'uuid': 'zzzzz-4zz18-mockcollection0',
            'portable_data_hash': dummy_hash,
            'manifest_text': nonnormalized_manifest,
        }

        # Because one_task_per_input_file normalizes this collection,
        # it should now create only one job task and not three.
        arvados.job_setup.one_task_per_input_file(and_end_task=False)
        mock_api('v1').job_tasks().create().execute.assert_called_once_with()
