### Copyright (C) The Arvados Authors. All rights reserved.

SPDX-License-Identifier: AGPL-3.0 ###

cache_age_in_days = (milliseconds_age) ->
  ONE_DAY = 1000 * 60 * 60 * 24
  milliseconds_age / ONE_DAY

cache_age_hover = (milliseconds_age) ->
  'Cache age ' + cache_age_in_days(milliseconds_age).toFixed(1) + ' days.'

cache_age_axis_label = (milliseconds_age) ->
  cache_age_in_days(milliseconds_age).toFixed(0) + ' days'

float_as_percentage = (proportion) ->
  (proportion.toFixed(4) * 100) + '%'

$.renderHistogram = (histogram_data) ->
  Morris.Area({
    element: 'cache-age-vs-disk-histogram',
    pointSize: 0,
    lineWidth: 0,
    data: histogram_data,
    xkey: 'age',
    ykeys: ['persisted', 'cache'],
    labels: ['Persisted Storage Disk Utilization', 'Cached Storage Disk Utilization'],
    ymax: 1,
    ymin: 0,
    xLabelFormat: cache_age_axis_label,
    yLabelFormat: float_as_percentage,
    dateFormat: cache_age_hover
  })
