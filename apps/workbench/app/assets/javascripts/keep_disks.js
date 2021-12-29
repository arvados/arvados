// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

(function() {
  var cache_age_axis_label, cache_age_hover, cache_age_in_days, float_as_percentage;

  cache_age_in_days = function(milliseconds_age) {
    var ONE_DAY;
    ONE_DAY = 1000 * 60 * 60 * 24;
    return milliseconds_age / ONE_DAY;
  };

  cache_age_hover = function(milliseconds_age) {
    return 'Cache age ' + cache_age_in_days(milliseconds_age).toFixed(1) + ' days.';
  };

  cache_age_axis_label = function(milliseconds_age) {
    return cache_age_in_days(milliseconds_age).toFixed(0) + ' days';
  };

  float_as_percentage = function(proportion) {
    return (proportion.toFixed(4) * 100) + '%';
  };

  $.renderHistogram = function(histogram_data) {
    return Morris.Area({
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
    });
  };

}).call(this);
