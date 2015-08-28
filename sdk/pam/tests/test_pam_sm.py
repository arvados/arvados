import arvados_pam
import mock
from . import mocker

class PamSMTest(mocker.Mocker):
    def attempt(self):
        return arvados_pam.pam_sm_authenticate(self.pamh, 0, self.argv)

    def test_success(self):
        self.assertEqual(self.pamh.PAM_SUCCESS, self.attempt())

    def test_bad_user(self):
        self.pamh.get_user = mock.MagicMock(return_value='badusername')
        self.assertEqual(self.pamh.PAM_AUTH_ERR, self.attempt())

    def test_bad_vm(self):
        self.argv[2] = 'testvm22.shell'
        self.assertEqual(self.pamh.PAM_AUTH_ERR, self.attempt())

    def setUp(self):
        super(PamSMTest, self).setUp()
        self.pamh = mock.MagicMock()
        self.pamh.get_user = mock.MagicMock(return_value='active')
        self.pamh.PAM_SUCCESS = 12345
        self.pamh.PAM_AUTH_ERR = 54321
        self.argv = [__file__, 'zzzzz.arvadosapi.com', 'testvm2.shell']
