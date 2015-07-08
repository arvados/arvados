import syslog
import sys
sys.argv=['']
import arvados
import os

def auth_log(msg):
 """Send errors to default auth log"""
 syslog.openlog(facility=syslog.LOG_AUTH)
 #syslog.openlog()
 syslog.syslog("libpam python Logged: " + msg)
 syslog.closelog()


def check_arvados_token(requested_username, token):
    auth_log("%s %s" % (requested_username, token))

    try:
 	f=file('/etc/default/arvados_pam')
	config=dict([l for l in f.readlines() if not l.startswith('#') or l.strip()==""])
	arvados_api_host=config['ARVADOS_API_HOST'].strip()
	hostname=config['HOSTNAME'].strip()
    except Exception as e:
	auth_log("problem getting default values" % (str(e)))

    try:
	arv = arvados.api('v1',host=arvados_api_host, token=token, cache=None)
    except Exception as e:
	auth_log(str(e))
	return False

    try:
	matches = arv.virtual_machines().list(filters=[['hostname','=',hostname]]).execute()['items']
    except Exception as e:
	auth_log(str(e))
	return False


    if len(matches) != 1:
        auth_log("libpam_arvados could not determine vm uuid for '%s'" % hostname)
        return False

    this_vm_uuid = matches[0]['uuid']
    auth_log("this_vm_uuid: %s" % this_vm_uuid)
    client_user_uuid = arv.users().current().execute()['uuid']

    filters = [
            ['link_class','=','permission'],
            ['name','=','can_login'],
            ['head_uuid','=',this_vm_uuid],
            ['tail_uuid','=',client_user_uuid]]

    for l in arv.links().list(filters=filters).execute()['items']:
         if requested_username == l['properties']['username']:
             return  True
    return False


def pam_sm_authenticate(pamh, flags, argv):
 try:
  user = pamh.get_user()
 except pamh.exception, e:
  return e.pam_result

 if not user:
  return pamh.PAM_USER_UNKNOWN

 try:
  resp = pamh.conversation(pamh.Message(pamh.PAM_PROMPT_ECHO_OFF, ''))
 except pamh.exception, e:
  return e.pam_result

 try:
  check = check_arvados_token(user, resp.resp)
 except Exception as e:
  auth_log(str(e))
  return False

 if not check:
  auth_log("Auth failed Remote Host: %s (%s:%s)" % (pamh.rhost, user, resp.resp))
  return pamh.PAM_AUTH_ERR

 auth_log("Success! Remote Host: %s (%s:%s)" % (pamh.rhost, user, resp.resp))
 return pamh.PAM_SUCCESS

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
