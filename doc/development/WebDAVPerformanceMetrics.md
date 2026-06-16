[comment]: # (Copyright © The Arvados Authors. All rights reserved.)
[comment]: # ()
[comment]: # (SPDX-License-Identifier: CC-BY-SA-3.0)

# WebDAV performance metrics

Monitoring WebDAV performance can be challenging. The obvious metric (file transfer speed) can be misleading because it's greatly affected by file size and cluster-to-client bandwidth.

As of [#15317](https://dev.arvados.org/issues/15317) keep-web produces a set of prometheus metrics that aim to expose cluster performance in a way that isolates those confounding factors.

Most notably:

## `arvados_keepweb_download_apparent_backend_speed`

This statistic represents the speed of transferring data from keep/cache into keep-web, as computed by subtracting the time spent waiting to send data to the client. When the client side is slow, this "apparent backend speed" can be higher than the maximum achievable transfer speed even on the backend -- unrealistically high numbers just mean backend speed is not a bottleneck. But when the client is receiving data fast enough that the keep/cache backend can't keep up, this stat should approach the actual bottleneck speed.

In other words, the apparent backend speed is expected to be bimodal, and when it is low -- i.e., close to actual download speed -- this means the backend is the bottleneck determining actual download speed. In that case it may be beneficial to increase cache size, upgrade keep-web server hardware, etc.

This statistic is bucketed by file size range (`size_range="0"` for files under 1 MB, `size_range="1M"` for files >= 1 MB and < 10 MB, `"10M"` for 10-100MB, `"100M"` for 100MB+) to help isolate the different overheads involved in serving different file sizes.

To graph the median apparent backend speed for large files:

```
histogram_quantile(0.5, sum(rate(arvados_keepweb_download_apparent_backend_speed_bucket{size_range="100M"}[5m])) by (le))
```
