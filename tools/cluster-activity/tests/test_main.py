# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0
import argparse
import datetime
import os
import re
import subprocess
from unittest.mock import patch

import pytest

import arvados_cluster_activity.main as aca_main


class TestSanity:
    """Test that the `arv-cluster-activity` script is installed and can be
    run as a standalone CLI app, that it understands the "-h" and "--version"
    CLI arguments, and that it exits with status 2 upon invalid CLI arguments.
    """
    script = "arv-cluster-activity"

    def run(self, *args, **kwargs) -> subprocess.CompletedProcess:
        return subprocess.run(
            [self.script, *args],
            capture_output=True, text=True, check=False, **kwargs
        )

    def test_help(self):
        completed_process = self.run("-h")
        assert completed_process.returncode == 0
        assert f"usage: {self.script}" in completed_process.stdout
        assert not completed_process.stderr

    def test_version(self):
        completed_process = self.run("--version")
        assert completed_process.returncode == 0
        assert re.match(
            (
                rf"^{re.escape(self.script)}"
                r" [0-9]+\.[0-9]+\.[0-9]+(\.dev[0-9]+)?$\n"
            ),
            completed_process.stdout
        )
        assert not completed_process.stderr

    def test_invalid_argument(self):
        completed_process = self.run("--x-invalid-argument")
        assert completed_process.returncode == 2
        assert not completed_process.stdout
        assert f"usage: {self.script}" in completed_process.stderr


class TestDateArgs:
    @pytest.mark.parametrize("date", (
        "0001-01-01", "1900-01-01", "1999-12-31", "2020-02-29"
    ))
    def test_valid_date_arg(self, date):
        y, m, d = map(int, date.split("-"))
        actual = aca_main._ArgTypes.date_arg(date)
        assert actual.tzinfo == datetime.timezone.utc
        assert actual.year == y
        assert actual.month == m
        assert actual.day == d
        assert actual.time() == datetime.time(0)

    @pytest.mark.parametrize("invalid_date", (
        "1900-02-29", "1999-11-31", "2026-13-01", "2026-01-42",
        "2026-07-21T12:34:56.7890Z", "foo", "1-1-1", "2025-DEC-31"
    ))
    def test_invalid_date_arg(self, invalid_date):
        with pytest.raises(
            argparse.ArgumentTypeError,
            match=re.compile(
                rf"^invalid date: {re.escape(repr(invalid_date))}$"
            )
        ):
            aca_main._ArgTypes.date_arg(invalid_date)

    def test_today_date_arg(self):
        now = "1987-01-23T23:45:06.789-10:00"
        expected = "1987-01-24T00:00:00+00:00"  # start of UTC day

        fake_now = datetime.datetime.fromisoformat(now).astimezone(
            datetime.timezone.utc
        )
        orig_replace = datetime.datetime.replace
        with patch("arvados_cluster_activity.main.datetime") as m:
            m.now.return_value = fake_now
            m.replace = orig_replace
            actual = aca_main._ArgTypes.date_arg()
        assert actual == datetime.datetime.fromisoformat(expected)

    @pytest.mark.parametrize("invalid_input", ("-1", "0x01", "foo"))
    def test_invalid_positive_days(self, invalid_input):
        with pytest.raises(ValueError):
            aca_main._ArgTypes.positive_days(invalid_input)


class TestLoadPrometheusAuth:
    def test_valid(self, tmp_path):
        key = "PROMETHEUS_FOO"
        val = "BAR"
        filler = "FOO"
        export_key = "PROMETHEUS_A"
        export_val = "B"
        prom_file = tmp_path / "prom"
        prom_file.write_text(
            f"{key}={val}\n"
            f"{filler}\n"
            f"export {export_key}={export_val}\n"
        )
        with patch("os.environ", {}):
            aca_main._ArgTypes.load_prometheus_auth(str(prom_file))
            assert os.environ == {key: val, export_key: export_val}

    def test_bad_path(self, tmp_path):
        old_environ = os.environ.copy()
        with pytest.raises(argparse.ArgumentTypeError):
            aca_main._ArgTypes.load_prometheus_auth(f"{tmp_path!s}/no_file")
        assert os.environ == old_environ


def test_invalid_columns():
    with pytest.raises(
        ValueError,
        match=rf"^invalid column name: {repr('Foo')}$"
    ):
        aca_main._ArgTypes.columns("Workflow,Foo,User")


def test_main_with_args_end_and_days():
    n_days = 1
    end = "2026-01-01"
    expected_end = datetime.datetime(2026, 1, 1, tzinfo=datetime.timezone.utc)
    expected_start = expected_end - datetime.timedelta(days=n_days)
    with (
        patch("arvados_cluster_activity.main.ClusterActivityReport") as m,
        patch("arvados_cluster_activity.main.get_prometheus_client") as gpc
    ):
        gpc.return_value = None
        aca_main.main(["--days", str(n_days), "--end", end])
    m.assert_called_once()
    m.assert_called_with(
        expected_start, expected_end, prom_client=None, exclude=None
    )
