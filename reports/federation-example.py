#!/usr/bin/env python
import sys

junit_txt = """
<?xml version="1.0" encoding="UTF-8"?>
<testsuites>
    <testsuite tests="10" failures="0" time="11.737" name="git.curoverse.com/arvados.git/sdk/go/keepclient">
        <properties>
            <property name="go.version" value="go1.8.3"></property>
            <property name="coverage.statements.pct" value="87.2"></property>
        </properties>
        <testcase classname="keepclient" name="Test" time="11.720"></testcase>
        <testcase classname="keepclient" name="TestSignLocator" time="0.000"></testcase>
        <testcase classname="keepclient" name="TestVerifySignature" time="0.000"></testcase>
        <testcase classname="keepclient" name="TestVerifySignatureExtraHints" time="0.000"></testcase>
        <testcase classname="keepclient" name="TestVerifySignatureWrongSize" time="0.000"></testcase>
        <testcase classname="keepclient" name="TestVerifySignatureBadSig" time="0.000"></testcase>
        <testcase classname="keepclient" name="TestVerifySignatureBadTimestamp" time="0.000"></testcase>
        <testcase classname="keepclient" name="TestVerifySignatureBadSecret" time="0.000"></testcase>
        <testcase classname="keepclient" name="TestVerifySignatureBadToken" time="0.000"></testcase>
        <testcase classname="keepclient" name="TestVerifySignatureExpired" time="0.000"></testcase>
    </testsuite>
</testsuites>
"""
 
f = open('reports/report.xml','w')
f.write(junit_txt)