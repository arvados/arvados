import sys
sys.argv=['']

import arvados
import os
import syslog
import yaml

def auth_log(msg):
    """Send errors to default auth log"""
    syslog.openlog(facility=syslog.LOG_AUTH)
    syslog.syslog('arvados_pam: ' + msg)
    syslog.closelog()

def config_file():
    return file('/etc/default/arvados_pam.conf').read()

def config():
    return yaml.load(config_file())

class AuthEvent(object):
    def __init__(self, config, service, client_host, username, token):
        self.config = config
        self.service = service
        self.client_host = client_host
        self.username = username
        self.token = token

        self.api_host = None
        self.vm_uuid = None
        self.user = None

    def can_login(self):
        """Return truthy IFF credentials should be accepted."""
        ok = False
        try:
            self.api_host = self.config[self.service]['ARVADOS_API_HOST']
            self.arv = arvados.api('v1', host=self.api_host, token=self.token, cache=None)

            vmname = self.config[self.service]['virtual_machine_hostname']
            vms = self.arv.virtual_machines().list(filters=[['hostname','=',vmname]]).execute()
            if vms['items_available'] > 1:
                raise Exception("lookup hostname %s returned %d records" % (vmname, vms['items_available']))
            if vms['items_available'] == 0:
                raise Exception("lookup hostname %s not found" % vmname)
            vm = vms['items'][0]
            if vm['hostname'] != vmname:
                raise Exception("lookup hostname %s returned hostname %s" % (vmname, vm['hostname']))
            self.vm_uuid = vm['uuid']

            self.user = self.arv.users().current().execute()

            filters = [
                ['link_class','=','permission'],
                ['name','=','can_login'],
                ['head_uuid','=',self.vm_uuid],
                ['tail_uuid','=',self.user['uuid']]]
            for l in self.arv.links().list(filters=filters, limit=10000).execute()['items']:
                if (l['properties']['username'] == self.username and
                    l['tail_uuid'] == self.user['uuid'] and
                    l['head_uuid'] == self.vm_uuid and
                    l['link_class'] == 'permission' and
                    l['name'] == 'can_login'):
                    return self._report(True)

            return self._report(False)

        except Exception as e:
            return self._report(e)

    def _report(self, result):
        """Log the result. Return truthy IFF result is True.

        result must be True, False, or an exception.
        """
        self.result = result
        auth_log(self.message())
        return result == True

    def message(self):
        """Return a log message describing the event and its outcome."""
        if isinstance(self.result, Exception):
            outcome = 'Error: ' + repr(self.result)
        elif self.result == True:
            outcome = 'Allow'
        else:
            outcome = 'Deny'

        if len(self.token) > 40:
            log_token = self.token[0:15]
        else:
            log_token = '<invalid>'

        log_label = [self.client_host, self.api_host, self.vm_uuid, self.username, log_token]
        if self.vm_uuid:
            log_label += [self.vm_uuid]
        if self.user:
            log_label += [self.user.get('uuid'), self.user.get('full_name')]
        return str(log_label) + ': ' + outcome


def pam_sm_authenticate(pamh, flags, argv):
    try:
        username = pamh.get_user()
    except pamh.exception as e:
        return e.pam_result

    if not username:
        return pamh.PAM_USER_UNKNOWN

    try:
        token = pamh.conversation(pamh.Message(pamh.PAM_PROMPT_ECHO_OFF, '')).resp
    except pamh.exception as e:
        return e.pam_result

    if AuthEvent(config(),
                 service=pamh.service,
                 client_host=pamh.rhost,
                 username=username,
                 token=token).can_login():
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
