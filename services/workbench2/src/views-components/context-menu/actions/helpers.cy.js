// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { sanitizeToken, getCollectionItemClipboardUrl, getInlineFileUrl } from "./helpers";

describe('helpers', () => {
    // given
    const url = 'https://example.com/c=zzzzz-4zz18-0123456789abcde/t=v2/a/b/LIMS/1.html';
    const urlWithPdh = 'https://example.com/c=012345678901234567890123456789aa+0/t=v2/a/b/LIMS/1.html';

    describe('sanitizeToken', () => {
        it('should sanitize token from the url', () => {
            // when
            const result = sanitizeToken(url);

            // then
            expect(result).to.equal('https://example.com/c=zzzzz-4zz18-0123456789abcde/LIMS/1.html?api_token=v2/a/b');
        });
    });

    describe('getClipboardUrl', () => {
        it('should add redirectTo query param', () => {
            // when
            const result = getCollectionItemClipboardUrl(url, "https://example.com", "https://*.example.com");

            // then
            expect(result).to.equal('http://localhost:8080?redirectToDownload=https://example.com/c=zzzzz-4zz18-0123456789abcde/LIMS/1.html');
        });
    });

    describe('getInlineFileUrl', () => {
        it('should add the collection\'s uuid to the hostname', () => {
            // when
            const webDavUrlA = 'https://*.collections.example.com/';
            const webDavUrlB = 'https://*--collections.example.com/';
            const webDavDownloadUrl = 'https://example.com/';

            // then
            expect(getInlineFileUrl(url, webDavDownloadUrl, webDavUrlA))
                .to.equal('https://zzzzz-4zz18-0123456789abcde.collections.example.com/t=v2/a/b/LIMS/1.html');
            expect(getInlineFileUrl(url, webDavDownloadUrl, webDavUrlB))
                .to.equal('https://zzzzz-4zz18-0123456789abcde--collections.example.com/t=v2/a/b/LIMS/1.html');
            expect(getInlineFileUrl(urlWithPdh, webDavDownloadUrl, webDavUrlA))
                .to.equal('https://012345678901234567890123456789aa-0.collections.example.com/t=v2/a/b/LIMS/1.html');
            expect(getInlineFileUrl(urlWithPdh, webDavDownloadUrl, webDavUrlB))
                .to.equal('https://012345678901234567890123456789aa-0--collections.example.com/t=v2/a/b/LIMS/1.html');
        });

        it('should keep the url the same when no inline url available', () => {
            // when
            const webDavUrl = '';
            const webDavDownloadUrl = 'https://example.com/';
            const result = getInlineFileUrl(url, webDavDownloadUrl, webDavUrl);

            // then
            expect(result).to.equal('https://example.com/c=zzzzz-4zz18-0123456789abcde/t=v2/a/b/LIMS/1.html');
        });

        it('should replace the url when available', () => {
            // when
            const webDavUrl = 'https://download.example.com/';
            const webDavDownloadUrl = 'https://example.com/';
            const result = getInlineFileUrl(url, webDavDownloadUrl, webDavUrl);

            // then
            expect(result).to.equal('https://download.example.com/c=zzzzz-4zz18-0123456789abcde/t=v2/a/b/LIMS/1.html');
        });
    });
});
