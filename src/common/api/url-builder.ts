// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

export default class UrlBuilder {
	private url: string = "";
	private query: string = "";

	constructor(host: string) {
		this.url = host;
	}

	public addParam(param: string, value: string) {
		if (this.query.length === 0) {
			this.query += "?";
		} else {
			this.query += "&";
		}
		this.query += `${param}=${value}`;
		return this;
	}

	public get() {
		return this.url + this.query;
	}
}
