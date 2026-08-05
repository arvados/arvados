# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

import datetime
import itertools

import pytest

from arvados_cluster_activity import report as acar

def timedelta_secs(secs):
    return datetime.timedelta(seconds=secs)


@pytest.mark.parametrize('value,expected', [
    (None, None),
    (123, '0.123 KB'),
    (2345, '2.345 KB'),
    (3456789, '3.457 MB'),
    (9876543210, '9.877 GB'),
])
def test_bytes_si_fmt(value, expected):
    assert acar.bytes_si_fmt(value) == expected


@pytest.mark.parametrize('value,expected', [
    (None, None),
    (512, '0.500 KiB'),
    (4096 + 256, '4.250 KiB'),
    ((2 << 20) + 2048, '2.002 MiB'),
    ((3 << 30) + (4 << 20), '3.004 GiB'),
])
def test_bytes_base2_fmt(value, expected):
    assert acar.bytes_base2_fmt(value) == expected


@pytest.mark.parametrize('value,expected', [
    (0, '$0.00'),
    (1, '$1.00'),
    (2.25, '$2.25'),
    (3.34321, '$3.34'),
    (1234.5678, '$1,234.57'),
    (10203040.567, '$10,203,040.57'),
])
def test_cost_fmt(value, expected):
    assert acar.cost_fmt(value) == expected


@pytest.mark.parametrize('isodate', [
    '2024-02-29',
    '2025-12-31',
    '2026-01-01',
])
def test_datespan_fmt_same(isodate):
    date = datetime.date.fromisoformat(isodate)
    assert acar.datespan_fmt(date, date) == isodate


@pytest.mark.parametrize('begin,end', [
    ('2024-01-15', '2024-02-29'),
    ('2025-03-01', '2025-04-01'),
    ('2026-06-01', '2026-09-30'),
])
def test_datespan_fmt_range(begin, end):
    date1 = datetime.date.fromisoformat(begin)
    date2 = datetime.date.fromisoformat(end)
    assert acar.datespan_fmt(date1, date2) == f'{begin} to {end}'


@pytest.mark.parametrize('offset', [0, 1, 2])
def test_datetime_fmt(offset):
    if offset:
        tz = datetime.timezone(datetime.timedelta(hours=offset))
    else:
        tz = datetime.timezone.utc
    delta = datetime.timedelta(days=offset, hours=offset)
    date = datetime.datetime(2024, 5, 10, 2, 30, 45, tzinfo=tz) + delta
    assert acar.datetime_fmt(date) == f'2024-05-{10 + offset} {2 + offset:02d}:30:45'


@pytest.mark.parametrize('dtype,in_out', itertools.product(
    [
        float,
        timedelta_secs,
    ], [
        (0, '0:00:00'),
        (4, '0:00:04'),
        (44, '0:00:44'),
        (444, '0:07:24'),
        (2444, '0:40:44'),
        (4444, '1:14:04'),
        (72345, '20:05:45'),
        (361836, '100:30:36'),
    ],
))
def test_duration_hms_fmt(dtype, in_out):
    secs, expected = in_out
    value = dtype(secs)
    assert acar.duration_hms_fmt(value) == expected


@pytest.mark.parametrize('dtype,in_out,suffix', itertools.product(
    [
        float,
        timedelta_secs,
    ], [
        (0, '0.0'),
        (1100, '0.3'),
        (2200, '0.6'),
        (3600, '1.0'),
        (5400, '1.5'),
        (36360, '10.1'),
        (720720, '200.2'),
    ], [
        '',
        'H',
        ' hours',
    ],
))
def test_duration_hrs_fmt(dtype, in_out, suffix):
    secs, exp_num = in_out
    value = dtype(secs)
    exp_suf = f' {suffix.lstrip()}'.rstrip()
    assert acar.duration_hrs_fmt(value, suffix) == f'{exp_num}{exp_suf}'


@pytest.mark.parametrize('tb_exp,period_div', itertools.product(
    [
        (25, 588.80),
        (75, 1740.80),
        (400, 9062.40),
        (600, 13465.60),
    ], [
        (acar.S3CostPeriod.MONTHLY, 1),
        (acar.S3CostPeriod.DAILY, 30),
        (acar.S3CostPeriod.HOURLY, 30 * 24),
    ],
))
def test_s3_cost(tb_exp, period_div):
    tb, exp_cost = tb_exp
    period, exp_div = period_div
    assert acar.s3_cost(tb << 40, period) == pytest.approx(exp_cost / exp_div)


@pytest.mark.parametrize('tb,expected', [
    (25, 588.80),
    (75, 1740.80),
    (400, 9062.40),
    (600, 13465.60),
])
def test_aws_monthly_cost(tb, expected):
    assert acar.aws_monthly_cost(tb << 40) == pytest.approx(expected)
