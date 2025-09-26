// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import moment from "moment";
import { Validator } from "redux-form"; // optional type hint

export const isValidDate: Validator = (value) => {
  if (value == null || value === "") return undefined; // treat empty as "no error" (use `required` separately)
  return moment(value).isValid() ? undefined : "Invalid date";
};
