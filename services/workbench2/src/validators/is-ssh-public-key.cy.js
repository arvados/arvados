// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { isSshPublicKey } from './is-ssh-public-key';

// Doing the inverse of getMpIntSegment()
function toSegment(text) {
    // First create the "size" uint32 field; only works for size not exceeding
    // 0xffff (65535).
    const sizeField = String.fromCharCode(0, 0, 0, text.length);
    return sizeField + text;
}

const ERROR_MESSAGE = 'Public key is invalid';

// 3072-bit RSA key
const goodRsaKey = 'ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQDPpavAS1wUq2+j7PgwkDS+9lm43AkdGxZo+T8qm6ZcB009EUEXya3lQolA52gg/i5aGZg4LT3t1OKxbsaClMd7sNZXYrMW9vd/utvGgAlNEbE/yXsEl2kpxt8lz7RI1XLnoWcV+aKyrsiKdrMKnZyG8CBxKdtzxHzWRl4N1BGrFJf/RnUWJv2VvM/h4/O+KXIjFokPkJ1F8yQChp5OKGkBKGXQ1vV4LjXqEXGVlgiQFM4U2NvCA8hXQR8mYm1vOsTYJzoSsnb+ewbXlVH5d7XsR5S2ULOr88vuYN/P4DF/Q3pEBi7BOyee61P3eHvhCNtb+jQMt59Vj/96y5C/reTMRo2R3B4bmX+Zxr3+DCC5tO1y+U5V39fu7cweimKXc78QDGGAVN0kz4P6P137b5WkCYIozeiBvWRsbGIlHjlGu9+0WuotdluD+OrTguuZ2zr8f32ijddO6y0J+aIdmTxQPxtmcQuRtpRfquoJGLhWAJH6mNZKbWkqqVfd5BA0TYs=';

const goodEd25519Key = "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIFaXqN760gkKXxNXpeSSOmdIWbyBH/1IatlX6/BCkREQ";

describe('RSA public key validator', () => {
    it('should accept keys with comment', () => {
        expect(isSshPublicKey(goodRsaKey + " firstlast@example.com")).to.be.undefined;
        expect(isSshPublicKey(goodRsaKey + " foo\tbar\t")).to.be.undefined;
        expect(isSshPublicKey(goodRsaKey + "\n")).to.be.undefined;
    });

    it('should accept keys without comment', () => {
        expect(isSshPublicKey(goodRsaKey)).to.be.undefined;
    });

    it('should reject invalid keys', () => {
        const badKey = 'ssh-rsa bad';
        expect(isSshPublicKey(badKey)).to.equal(ERROR_MESSAGE);
    });
});

describe('Ed25519 public key validator', () => {
    it('should accept keys with comment', () => {
        expect(isSshPublicKey(goodEd25519Key + " foobar")).to.be.undefined;
    });

    it('should accept keys without comment', () => {
        expect(isSshPublicKey(goodEd25519Key)).to.be.undefined;
    });
});

describe("Validator capable of rejecting malformed or unsupported keys", () => {
    it("should reject key with unsupported type", () => {
        const unsupportedKey = "ssh-dss AAAAB3NzaC1kc3MAAACBANxTbA537hM9OYuWVhe0he6wkqOSNt6pRLsx+2p0/7DQKs4UoIKmVWSPLbVES0XrjGkoALEPDHb3mI6vZANXg3/LwtpbwJsnIG5i46cIUC0Uue/4ed0JHp11rGnUAIZybOV4R1BtA1l8K2pbXfwXMqy21KU4hOPHEua4FvIqLrC/AAAAFQDyS1YjrWrF3fpHAygcMSeOKYfPCQAAAIAIBlcKQHh1hYgGOA4+Ae686JkiyIXUzXipcYqVAqpPacr9Cu+zlwV9FFvkNvCJTcmQant+AwoTJ2+rPQUvAWZGdnVtmW5xHKjTO49+s8CBq/iaHF7r/t8GqUqQPFhDHEEDyQ439cmWv2NL4cC7mtADaEDSRUgpIJDgNfxo/0uW1AAAAIEAyjb0KSJjmtZL6jySmQzgHKDfQd+A9p4mCceA4v4eKeBEXOwzRUxxJEZ/PExvDwz+2r/mfRd1iow0okcL475wRRSqaPSCyddSkst2WA2hxA4ukGYvQP4y9tYMmNYpiOBU19Lpz93jQ6ejGA/wj4xAUcD5nQxYsgb1CI2uSgJyaTE=";
        expect(isSshPublicKey(unsupportedKey)).to.equal("Unknown key type (currently supported: RSA, Ed25519)");
    });

    it("should reject key with inconsistent key-type label", () => {
        // I.e., change the text label in the input without modifying the label
        // in the binary key data.
        let badKey = goodEd25519Key.replace("ssh-ed25519 ", "ssh-rsa ");
        expect(isSshPublicKey(badKey)).to.equal(ERROR_MESSAGE);

        badKey = goodRsaKey.replace("ssh-rsa ", "ssh-ed25519 ");
        expect(isSshPublicKey(badKey)).to.equal(ERROR_MESSAGE);
    });

    it("should reject key data that cannot be decoded", () => {
        // Inject illegal character "@" in base64-encoded data
        const badKey = goodEd25519Key.replace("ssh-ed25519 AAAA", "ssh-ed25519 @AAA");
        expect(isSshPublicKey(badKey)).to.equal(ERROR_MESSAGE);
    });

    it("should reject Ed25519 key with trailing data", () => {
        // An Ed25519 key is 51 = 3 * 17 bytes, which don't require padding
        // when base64-encoded, so we can simply concatenate to the end. This
        // doesn't change the size field.
        const badKey = goodEd25519Key + window.btoa("foo");
        cy.log("Bad key (trailing data)", badKey);
        expect(isSshPublicKey(badKey)).to.equal(ERROR_MESSAGE);
    });

    it("should reject RSA key with trailing data", () => {
        // Append "foo" to the binary key data (i.e. the modulus field),
        // without changing the size field.
        const badKeyData = window.atob(goodRsaKey.split(" ")[1]) + "foo";
        const badKey = `ssh-rsa ${window.btoa(badKeyData)}`;
        cy.log("Bad key (trailing data)", badKey);
        expect(isSshPublicKey(badKey)).to.equal(ERROR_MESSAGE);
    });

    it("should reject truncated Ed25519 key", () => {
        // Discard last 3 bytes of key data without changing the size field.
        // This can be done by simply removing the last 4 letters from the
        // base64-encoded data.
        const badKey = goodEd25519Key.slice(0, -4);
        cy.log("Bad key (truncated)", badKey);
        expect(isSshPublicKey(badKey)).to.equal(ERROR_MESSAGE);
    });

    it("should reject truncated RSA key", () => {
        // Discard last byte of key data without changing the size field.
        const badKeyData = window.atob(goodRsaKey.split(" ")[1]).slice(0, -1);
        const badKey = `ssh-rsa ${window.btoa(badKeyData)}`;
        cy.log("Bad key (truncated)", badKey);
        expect(isSshPublicKey(badKey)).to.equal(ERROR_MESSAGE);
    });

    it("should reject Ed25519 key that's not precisely 32 bytes", () => {
        // Append "foo" to the 32-byte binary key (i.e. the last 32 bytes of
        // the binary key data).
        const badKeyPayload = window.atob(goodEd25519Key.split(" ")[1]).slice(-32) + "foo";
        // Reconstruct a "key" with the actual size. This truthfully puts the
        // value 35 (bytes) into the size field.
        const badKeyData = toSegment("ssh-ed25519") + toSegment(badKeyPayload);
        const badKey = `ssh-ed25519 ${window.btoa(badKeyData)}`;
        cy.log("Bad key (nonstandard size)", badKey);
        expect(isSshPublicKey(badKey)).to.equal(ERROR_MESSAGE);
    });

    it("should reject RSA key with malformed modulus whose most significant bit is set", () => {
        const goodKeyData = window.atob(goodRsaKey.split(" ")[1]);
        const badByteIdx = (
            4 + 7  // size field + "ssh-rsa"
            + 4 + 3  // size field + 0x010001 (exponent)
            + 4  // size field for modulus
        );
        const badKeyChars = goodKeyData.split("");
        badKeyChars[badByteIdx] = String.fromCharCode(
            goodKeyData.charCodeAt(badByteIdx) | 0x0080  // set leading bit.
        );
        const badKey = `ssh-rsa ${window.btoa(badKeyChars.join(""))}`;
        cy.log("Bad key (wrong modulus sign)", badKey);
        expect(isSshPublicKey(badKey)).to.equal(ERROR_MESSAGE);
    });

    it("should reject RSA key with too-short modulus", () => {
        const badKeyData = (
            toSegment("ssh-rsa")
            + toSegment("\u0001\u0000\u0001")  // exponent 0x010001
            + toSegment("\u0000".repeat(127))  // fake modulus
        );
        const badKey = `ssh-rsa ${window.btoa(badKeyData)}`;
        cy.log("Bad key (wrong modulus size)", badKey);
        expect(isSshPublicKey(badKey)).to.equal(ERROR_MESSAGE);
    });

    it("should reject incomplete Ed25519 key without any key data", () => {
        const badKeyData = toSegment("ssh-ed25519");
        const badKey = `ssh-ed25519 ${window.btoa(badKeyData)}`;
        cy.log("Bad key (no actual key data)", badKey);
        expect(isSshPublicKey(badKey)).to.equal(ERROR_MESSAGE);
    });

    it("should reject incomplete RSA key without modulus", () => {
        const badKeyData = toSegment("ssh-rsa") + toSegment("\u0001\u0000\u0001");
        const badKey = `ssh-rsa ${window.btoa(badKeyData)}`;
        cy.log("Bad key (no modulus)", badKey);
        expect(isSshPublicKey(badKey)).to.equal(ERROR_MESSAGE);
    });
});
