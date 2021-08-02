// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { isRsaKey } from './is-rsa-key';

describe('rsa-key-validator', () => {
    const rsaKey = 'ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQDPpavAS1wUq2+j7PgwkDS+9lm43AkdGxZo+T8qm6ZcB009EUEXya3lQolA52gg/i5aGZg4LT3t1OKxbsaClMd7sNZXYrMW9vd/utvGgAlNEbE/yXsEl2kpxt8lz7RI1XLnoWcV+aKyrsiKdrMKnZyG8CBxKdtzxHzWRl4N1BGrFJf/RnUWJv2VvM/h4/O+KXIjFokPkJ1F8yQChp5OKGkBKGXQ1vV4LjXqEXGVlgiQFM4U2NvCA8hXQR8mYm1vOsTYJzoSsnb+ewbXlVH5d7XsR5S2ULOr88vuYN/P4DF/Q3pEBi7BOyee61P3eHvhCNtb+jQMt59Vj/96y5C/reTMRo2R3B4bmX+Zxr3+DCC5tO1y+U5V39fu7cweimKXc78QDGGAVN0kz4P6P137b5WkCYIozeiBvWRsbGIlHjlGu9+0WuotdluD+OrTguuZ2zr8f32ijddO6y0J+aIdmTxQPxtmcQuRtpRfquoJGLhWAJH6mNZKbWkqqVfd5BA0TYs=';
    const badKey = 'ssh-rsa bad'

    const ERROR_MESSAGE = 'Public key is invalid';

    describe('rsaKeyValidation', () => {
        it('should accept keys with comment', () => {
            // then
            expect(isRsaKey(rsaKey + " firstlast@example.com")).toBeUndefined();
        });

        it('should accept keys without comment', () => {
            // then
            expect(isRsaKey(rsaKey)).toBeUndefined();
        });

        it('should reject keys with trailing whitespace', () => {
            // then
            expect(isRsaKey(rsaKey + " ")).toBe(ERROR_MESSAGE);
            expect(isRsaKey(rsaKey + "\n")).toBe(ERROR_MESSAGE);
            expect(isRsaKey(rsaKey + "\r\n")).toBe(ERROR_MESSAGE);
            expect(isRsaKey(rsaKey + "\t")).toBe(ERROR_MESSAGE);
        });

        it('should reject invalid keys', () => {
            // then
            expect(isRsaKey(badKey)).toBe(ERROR_MESSAGE);
        });

    });

});
