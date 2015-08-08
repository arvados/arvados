import sys
sys.argv=['']

import arvados
import os
import syslog

def auth_log(msg):
    """Send errors to default auth log"""
    syslog.openlog(facility=syslog.LOG_AUTH)
    syslog.syslog('arvados_pam: ' + msg)
    syslog.closelog()

def config_file():
    return file('/etc/default/arvados_pam')

def config():
    txt = config_file().read()
    c = dict()
    for x in txt.splitlines(False):
        if not x.strip().startswith('#'):
            kv = x.split('=', 2)
            c[kv[0].strip()] = kv[1].strip()
    return c

class AuthEvent(object):
    def __init__(self, client_host, api_host, shell_host, username, token):
        self.client_host = client_host
        self.api_host = api_host
        self.shell_hostname = shell_host
        self.username = username
        self.token = token
        self.vm = None
        self.user = None

    def can_login(self):
        ok = False
        try:
            self.arv = arvados.api('v1', host=self.api_host, token=self.token, cache=None)
            self._lookup_vm()
            if self._check_login_permission():
                self.result = 'Authenticated'
                ok = True
            else:
                self.result = 'Denied'
        except Exception as e:
            self.result = 'Error: ' + repr(e)
        auth_log(self.message())
        return ok

    def _lookup_vm(self):
        """Load the VM record for this host into self.vm. Raise if not possible."""

        vms = self.arv.virtual_machines().list(filters=[['hostname','=',self.shell_hostname]]).execute()
        if vms['items_available'] > 1:
            raise Exception("ambiguous VM hostname matched %d records" % vms['items_available'])
        if vms['items_available'] == 0:
            raise Exception("VM hostname not found")
        self.vm = vms['items'][0]
        if self.vm['hostname'] != self.shell_hostname:
            raise Exception("API returned record with wrong hostname")

    def _check_login_permission(self):
        """Check permission to log in. Return True if permission is granted."""
        self._lookup_vm()
        self.user = self.arv.users().current().execute()
        filters = [
            ['link_class','=','permission'],
            ['name','=','can_login'],
            ['head_uuid','=',self.vm['uuid']],
            ['tail_uuid','=',self.user['uuid']]]
        for l in self.arv.links().list(filters=filters, limit=10000).execute()['items']:
            if (l['properties']['username'] == self.username and
                l['tail_uuid'] == self.user['uuid'] and
                l['head_uuid'] == self.vm['uuid'] and
                l['link_class'] == 'permission' and
                l['name'] == 'can_login'):
                return True
        return False

    def message(self):
        if len(self.token) > 40:
            log_token = self.token[0:15]
        else:
            log_token = '<invalid>'
        log_label = [self.client_host, self.api_host, self.shell_hostname, self.username, log_token]
        if self.vm:
            log_label += [self.vm.get('uuid')]
        if self.user:
            log_label += [self.user.get('uuid'), self.user.get('full_name')]
        return str(log_label) + ': ' + self.result


def pam_sm_authenticate(pamh, flags, argv):
    try:
        user = pamh.get_user()
    except pamh.exception as e:
        return e.pam_result

    if not user:
        return pamh.PAM_USER_UNKNOWN

    try:
        resp = pamh.conversation(pamh.Message(pamh.PAM_PROMPT_ECHO_OFF, ''))
    except pamh.exception as e:
        return e.pam_result

    try:
        config = config()
        api_host = config['ARVADOS_API_HOST'].strip()
        shell_host = config['HOSTNAME'].strip()
    except Exception as e:
        auth_log("loading config: " + repr(e))
        return False

    if AuthEvent(pamh.rhost, api_host, shell_host, user, resp.resp).can_login():
        return pamh.PAM_SUCCESS
    else:
        return pamh.PAM_AUTH_ERR

def pam_sm_setcred(pamh, flags, argv):
    return pamh.PAM_SUCCESS

def pam_sm_acct_mgmt(pamh, flags, argv):
    return pamh.PAM_SUCCESS

def pam_sm_open_session(pamh, flags, argv):
    return pamh.PAM_SUCCESS

def pam_sm_close_session(pamh, flags, argv):
    return pamh.PAM_SUCCESS

def pam_sm_chauthtok(pamh, flags, argv):
    return pamh.PAM_SUCCESS
