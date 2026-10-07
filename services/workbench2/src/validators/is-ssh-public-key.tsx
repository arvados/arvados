// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

// General note about using a string as binary byte array (for compatibility):
// We actually want lone surrogates. The String length property, slice method,
// split method, and indexing all work on the UTF-16 basis (may produce lone
// surrogates), but iterating produces Unicode characters that can be wider
// than 16 bits. (So String#split() is okay, while Array.from() isn't.)

// "mpint" stands for multi-precision integer, encoded as a 32-bit size
// followed by "size" bytes of data.

const ERROR_MESSAGE = 'Public key is invalid';

export const isSshPublicKey = (value: string): string | undefined => {
    // An OpenSSH public key is KEY-TYPE SPACE KEY-DATA [SPACE COMMENT],
    // where the optional comment is free text. We simply ignore the comment.
    const [keyType, keyString] = value.split(" ", 3);
    if (keyString === undefined) return ERROR_MESSAGE;

    let keyData: string | undefined;
    try {
        keyData = window.atob(keyString);
    } catch (e) {
        return ERROR_MESSAGE;
    }

    switch (keyType) {
        case "ssh-rsa":
            return isRsaKeyData(keyData) ? undefined : ERROR_MESSAGE;
        case "ssh-ed25519":
            return isEd25519KeyData(keyData) ? undefined : ERROR_MESSAGE;
        default:
            return "Unknown key type (currently supported: RSA, Ed25519)";
    }
};

// See RFC 4253; https://www.rfc-editor.org/info/rfc4253/#section-6.6
const isRsaKeyData = (inputData: string): boolean => {
    let pos = 0;
    // First segment always encodes the ASCII string "ssh-rsa".
    let segment = getMpIntSegment(inputData);
    if (segment === null || segment.data !== "ssh-rsa") return false;
    pos += segment.size;

    // Key content is two mpints, the public exponent (e) and the modulus (n).
    // Exponent e, in practice almost always 0x010001.
    segment = getMpIntSegment(inputData.slice(pos));
    // Check anyway. 5 == 4 + at least 1 byte of e.
    if (segment === null || isBadRsaMpInt(segment, 5)) return false;
    pos += segment.size;

    // Modulus n.
    segment = getMpIntSegment(inputData.slice(pos))
    // Min size of n is 1024 bits; see ssh-keygen(1). 132 == 4 + 1024 / 8
    if (segment === null || isBadRsaMpInt(segment, 132)) return false;
    pos += segment.size;

    if (pos !== inputData.length) return false;  // there's trailing data.
    return true;
};

// See RFC 8709; https://www.rfc-editor.org/info/rfc8709/#section-4
const isEd25519KeyData = (inputData: string): boolean => {
    let pos = 0;
    // First segment always encodes the ASCII string "ssh-ed25519".
    let segment = getMpIntSegment(inputData);
    if (segment === null || segment.data !== "ssh-ed25519") return false;
    pos += segment.size;

    // Key data is a 32-byte mpint.
    segment = getMpIntSegment(inputData.slice(pos))
    // Size check includes 4 bytes for "size" itself.
    if (segment === null || segment.size !== 36) return false;
    pos += segment.size;

    if (pos !== inputData.length) return false;  // there's trailing data.
    return true;
};

type MpInt = {
    size: number;  // size of this mpint segment in bytes, including the size field itself (4 bytes).
    data: string;  // payload of the given size.
};

const getMpIntSegment = (stream: string): MpInt | null => {
    if (stream.length < 4) return null;

    const size = 4 + arrayToUint(
        stream.slice(0, 4).split("").map((c) => c.charCodeAt(0))
    );
    if (stream.length < size) return null;  // size is out of bound.

    return { size, data: stream.slice(4, size) };
};

// Interpret uint8 array as little-endian unsigned integer.
// In practice, only use it for small-size arrays.
const arrayToUint = (intArray: readonly number[]): number => {
    const maxIdx = intArray.length - 1;
    if (maxIdx < 0) return 0;
    return intArray.reduce(
        (sum, curr, idx) => sum + curr * (1 << (8 * (maxIdx - idx))),
        0  // init
    );
};

const isBadRsaMpInt = (segment: Readonly<MpInt>, minSize: number): boolean => (
    // NOTE: minSize should include the 4 bytes of "size" itself.
    segment.size < minSize
    // When interpreted as a signed integer, sign (formally most significant)
    // bit must be zero. This is why a modulus may have 1-byte zero padding.
    || !!(segment.data.charCodeAt(0) & 0x0080)  // leading bit in UTF-16 encoding
);
