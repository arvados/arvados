// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import moment from "moment";
import { Validator } from "redux-form"; // optional type hint

export const isValidFutureDate: Validator = (value) => {
  const momentDate = moment(value);
  return momentDate.isValid() && momentDate.isAfter(moment()) ? undefined : "Invalid date";
};
