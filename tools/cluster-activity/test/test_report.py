# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

import datetime

from unittest import mock

from io import StringIO

from arvados_cluster_activity.report import ClusterActivityReport

class _TestingClusterActivityReport(ClusterActivityReport):
    def report_from_api(self, since, to, include_steps, exclude):
        items = [
            {
                "Project": "WGS chr19 test for 2.7.2~rc3",
                "ProjectUUID": "pirca-j7d0g-cukk4aw4iamj90c",
                "Workflow": "WGS processing workflow scattered over samples (v1.1-2-gcf002b3)",
                "WorkflowUUID": "none",
                "Step": "workflow runner",
                "StepUUID": "pirca-xvhdp-zyv7bm0tl3lm2nv",
                "Sample": "Sample1",
                "SampleUUID": "pirca-xvhdp-zyv7bm0tl3lm2nv",
                "User": "User1",
                "UserUUID": "jutro-tpzed-a4qnxq3pcfcgtkz",
                "Submitted": "2024-04-05 20:38:07",
                "Started": "2024-04-05 20:40:31",
                "Finished": "2024-08-22 12:34:56",
                "Runtime": "1:19:21",
                "Cost": 0.113,
                "CumulativeCost": 1.371,
            },

            # WGS chr19 test for 2.7.2~rc3,pirca-j7d0g-cukk4aw4iamj90c,workflow run from command line,none,bwamem-samtools-view,pirca-xvhdp-e63h0f57of5cr3t,WGS processing workflow scattered over samples (v1.1-2-gcf002b3),WGS processing workflow scattered over samples (v1.1-2-gcf002b3),Peter Amstutz,jutro-tpzed-a4qnxq3pcfcgtkz,2024-04-05 20:40:42,2024-04-05 20:43:20,0:08:52,0.121,0.121
            {
                "Project": "WGS chr19 test for 2.7.2~rc3",
                "ProjectUUID": "pirca-j7d0g-cukk4aw4iamj90c",
                "Workflow": "WGS processing workflow scattered over samples (v1.1-2-gcf002b3)",
                "WorkflowUUID": "none",
                "Step": "bwamem-samtools-view",
                "StepUUID": "pirca-xvhdp-e63h0f57of5cr3t",
                "Sample": "Sample1",
                "SampleUUID": "pirca-xvhdp-zyv7bm0tl3lm2nv",
                "User": "User1",
                "UserUUID": "jutro-tpzed-a4qnxq3pcfcgtkz",
                "Submitted": "2024-04-05 20:40:42",
                "Started": "2024-04-05 20:43:20",
                "Finished": "2024-04-05 20:51:40",
                "Runtime": "0:08:22",
                "Cost": 0.116,
                "CumulativeCost": 0.116,
            },
        ]

        for i in items:
            self.collect_summary_stats(i)
            yield i

        self.total_users = 4
        self.total_projects = 6

    def collect_graph(self, since, to, metric, resample_to, extra=None):
        items = [[datetime.datetime(year=2024, month=4, day=6, hour=11, minute=0, second=0), 3],
                 [datetime.datetime(year=2024, month=4, day=6, hour=11, minute=5, second=0), 5],
                 [datetime.datetime(year=2024, month=4, day=6, hour=11, minute=10, second=0), 2],
                 [datetime.datetime(year=2024, month=4, day=6, hour=11, minute=15, second=0), 5],
                 [datetime.datetime(year=2024, month=4, day=6, hour=11, minute=20, second=0), 3]]

        for i in items:
            if extra:
                extra(i[0], i[1])

        return items

    def today(self):
        return datetime.date(year=2024, month=4, day=6)

@mock.patch("arvados.api")
def test_report(apistub):

    write_report = False

    apistub().config.return_value = {
        "ClusterID": "xzzz1",
        "Services": {
            "Workbench2": {
                "ExternalURL": "https://xzzz1.arvadosapi.com"
            },
        },
    }

    prom_client = mock.MagicMock()
    report_obj = _TestingClusterActivityReport(prom_client)

    ## test CSV report
    csvreport = StringIO()
    report_obj.csv_report(datetime.datetime(year=2024, month=4, day=4),
                          datetime.datetime(year=2024, month=4, day=6),
                          csvreport,
                          True,
                          None,
                          "")
    if write_report:
        with open("test/test_report.csv", "wt") as f:
            f.write(csvreport.getvalue())

    with open("test/test_report.csv", "rt", newline='') as f:
        assert csvreport.getvalue() == f.read()


    ## test HTML report
    htmlreport = report_obj.html_report(datetime.datetime(year=2024, month=4, day=4),
                                        datetime.datetime(year=2024, month=4, day=6),
                                        "",
                                        True)

    if write_report:
        with open("test/test_report.html", "wt") as f:
            f.write(htmlreport)

    with open("test/test_report.html", "rt") as f:
        assert f.read() == htmlreport
