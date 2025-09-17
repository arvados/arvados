// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { injectTokenParam } from './service-menu';

describe('ServiceMenu', () => {
    it('injects tokens into valid URLs', () => {
        const testCases = [{
            // Test normal case
            url: "http://example.com/",
            token: "v2/xxxxx-gj3su-000000000000000/00000000000000000000000000000000000000000000000000",
            result: "http://example.com/?arvados_api_token=v2/xxxxx-gj3su-000000000000000/00000000000000000000000000000000000000000000000000",
        },{
            // Test no trailing slash - URL constructor will add trailing slash
            url: "https://example.com",
            token: "foobar",
            result: "https://example.com/?arvados_api_token=foobar",
        },{
            // Test with basic auth
            url: "https://user:pass@example.com/",
            token: "baz",
            result: "https://user:pass@example.com/?arvados_api_token=baz",
        },{
            // Test with existing params
            url: "https://example.com/?foo=bar",
            token: "foo123",
            result: "https://example.com/?arvados_api_token=foo123&foo=bar",
        },{
            // Test with existing params and no slash - URL constructor will add slash
            url: "https://example.com?foo=bar",
            token: "foo123",
            result: "https://example.com/?arvados_api_token=foo123&foo=bar",
        },{
            // Test with no params but with question mark
            url: "http://example.com/?",
            token: "foobar",
            result: "http://example.com/?arvados_api_token=foobar",
        }];

        return Promise.all(testCases.map(async testCase => {
            const result = await injectTokenParam(testCase.url, testCase.token);
            expect(result).to.equal(testCase.result);
        }));
    });

    it('raises exceptions for invalid situations', () => {
        const invalidCases = [{
            url: "http://example.com",
            token: "",
            msg: "User token required",
        },{
            url: "",
            token: "foo",
            msg: "URL cannot be empty",
        }];

        return Promise.all(invalidCases.map(testCase => {
            const promise = injectTokenParam(testCase.url, testCase.token);

            return promise.then(() => {
                    throw new Error('Expected injectTokenParam() to return error but it did not. '
                        + `Expected error: "${testCase.msg}" given url "${testCase.url}" and token "${testCase.token}"`);
                }, (err) => {
                    // Verify the promise rejection reason
                    expect(err).to.equal(testCase.msg);
                }
            );
        }));

    });
});
