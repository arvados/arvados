# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

import sys
sys.argv=['']

from . import auth_event

def pam_sm_authenticate(pamh, flags, argv):
    config = {}
    config['arvados_api_host'] = argv[1]
    config['virtual_machine_hostname'] = argv[2]
    if len(argv) > 3:
        for k in argv[3:]:
            config[k] = True

    try:
        username = pamh.get_user(None)
    except pamh.exception, e:
        return e.pam_result

    if not username:
        return pamh.PAM_USER_UNKNOWN

    try:
        prompt = '' if config.get('noprompt') else 'Arvados API token: '
        token = pamh.conversation(pamh.Message(pamh.PAM_PROMPT_ECHO_OFF, prompt)).resp
    except pamh.exception as e:
        return e.pam_result

    if auth_event.AuthEvent(
            config=config,
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
