// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

package main

/*
#cgo LDFLAGS: -lpam -fPIC
#include <security/pam_ext.h>
char *stringindex(char** a, int i) { return a[i]; }
const char *get_user(pam_handle_t *pamh) {
  const char *user;
  if (pam_get_item(pamh, PAM_USER, (const void**)&user) != PAM_SUCCESS)
    return NULL;
  return user;
}
const char *get_authtoken(pam_handle_t *pamh) {
  const char *token;
  if (pam_get_authtok(pamh, PAM_AUTHTOK, &token, NULL) != PAM_SUCCESS)
    return NULL;
  return token;
}
*/
import "C"
