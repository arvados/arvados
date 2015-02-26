#!/usr/bin/env python

import traceback
import unittest

import arvados.errors as arv_error
import arvados_testutil as tutil

class KeepRequestErrorTestCase(unittest.TestCase):
    REQUEST_ERRORS = [
        ('http://keep1.zzzzz.example.org/', IOError("test IOError")),
        ('http://keep3.zzzzz.example.org/', MemoryError("test MemoryError")),
        ('http://keep5.zzzzz.example.org/', tutil.fake_requests_response(
                500, "test 500")),
        ('http://keep7.zzzzz.example.org/', IOError("second test IOError")),
        ]

    def check_get_message(self, *args):
        test_exc = arv_error.KeepRequestError("test message", *args)
        self.assertEqual("test message", test_exc.message)

    def test_get_message_with_request_errors(self):
        self.check_get_message(self.REQUEST_ERRORS[:])

    def test_get_message_without_request_errors(self):
        self.check_get_message()

    def check_get_request_errors(self, *args):
        expected = dict(args[0]) if args else {}
        test_exc = arv_error.KeepRequestError("test service exceptions", *args)
        self.assertEqual(expected, test_exc.request_errors())

    def test_get_request_errors(self):
        self.check_get_request_errors(self.REQUEST_ERRORS[:])

    def test_get_request_errors_none(self):
        self.check_get_request_errors({})

    def test_empty_exception(self):
        test_exc = arv_error.KeepRequestError()
        self.assertFalse(test_exc.message)
        self.assertEqual({}, test_exc.request_errors())

    def traceback_str(self, exc):
        return traceback.format_exception_only(type(exc), exc)[-1]

    def test_traceback_str_without_request_errors(self):
        message = "test plain traceback string"
        test_exc = arv_error.KeepRequestError(message)
        exc_report = self.traceback_str(test_exc)
        self.assertTrue(exc_report.startswith("KeepRequestError: "))
        self.assertIn(message, exc_report)

    def test_traceback_str_with_request_errors(self):
        message = "test traceback shows Keep services"
        test_exc = arv_error.KeepRequestError(message, self.REQUEST_ERRORS[:])
        exc_report = self.traceback_str(test_exc)
        self.assertTrue(exc_report.startswith("KeepRequestError: "))
        for expect_substr in [message, "raised IOError", "raised MemoryError",
                              "test MemoryError", "second test IOError",
                              "responded with 500 Internal Server Error"]:
            self.assertIn(expect_substr, exc_report)
        # Assert the report maintains order of listed services.
        last_index = -1
        for service_key, _ in self.REQUEST_ERRORS:
            service_index = exc_report.find(service_key)
            self.assertGreater(service_index, last_index)
            last_index = service_index
